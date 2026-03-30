package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"agent-wallet-gas-sponsor/common"
)

// 价格信息结构
type PriceInfo struct {
	USD float64 `json:"usd"`
}

type CoinGeckoResponse struct {
	Solana   PriceInfo `json:"solana"`
	Ethereum PriceInfo `json:"ethereum"`
	Sui      PriceInfo `json:"sui"`
}

// Gas费用估算结果
type GasEstimate struct {
	Chain          string  `json:"chain"`
	EstimatedGas   float64 `json:"estimated_gas"`    // 原生代币数量
	NativeToken    string  `json:"native_token"`     // 原生代币名称 (SOL, ETH, SUI)
	TokenPriceUSD  float64 `json:"token_price_usd"`  // 原生代币USD价格
	USDCAmount     float64 `json:"usdc_amount"`      // 需要支付的USDC数量
	Description    string  `json:"description"`
}

// 根据交易数据估算gas费用
func estimateGasCost(req common.ExecuteRequest) (*GasEstimate, error) {
	switch req.Chain {
	case "solana":
		return estimateSolanaGas(req)
	case "evm", "ethereum":
		return estimateEVMGas(req)
	case "sui":
		return estimateSuiGas(req)
	default:
		return nil, fmt.Errorf("不支持的链: %s", req.Chain)
	}
}

// Solana gas估算
func estimateSolanaGas(req common.ExecuteRequest) (*GasEstimate, error) {
	// 基础交易费用约 0.000005 SOL
	baseGas := 0.000005
	
	// 根据交易复杂度调整
	// 这里可以解析 tx_data 来更精确估算
	complexityMultiplier := 1.0
	if len(req.TxData) > 1000 {
		complexityMultiplier = 2.0 // 复杂交易
	}
	
	estimatedSOL := baseGas * complexityMultiplier
	
	// 获取SOL价格
	solPrice, err := getTokenPrice("solana")
	if err != nil {
		// 如果获取价格失败，使用默认价格
		solPrice = 100.0 // 假设SOL = $100
	}
	
	usdcAmount := estimatedSOL * solPrice
	
	return &GasEstimate{
		Chain:          "solana",
		EstimatedGas:   estimatedSOL,
		NativeToken:    "SOL",
		TokenPriceUSD:  solPrice,
		USDCAmount:     usdcAmount,
		Description:    fmt.Sprintf("Solana transaction gas: %.6f SOL (~$%.4f)", estimatedSOL, usdcAmount),
	}, nil
}

// EVM gas估算 (Ethereum/Polygon等)
func estimateEVMGas(req common.ExecuteRequest) (*GasEstimate, error) {
	// 基础gas limit: 21000 (简单转账)
	baseGasLimit := 21000.0
	
	// 根据交易类型调整
	if req.TargetAddress != "" && len(req.TxData) > 0 {
		baseGasLimit = 100000.0 // 合约调用
	}
	
	// Gas price (gwei) - 这里应该从链上获取
	gasPriceGwei := 20.0 // 20 gwei
	gasPriceEth := gasPriceGwei / 1e9
	
	estimatedETH := (baseGasLimit * gasPriceEth)
	
	// 获取ETH价格
	ethPrice, err := getTokenPrice("ethereum")
	if err != nil {
		ethPrice = 2000.0 // 假设ETH = $2000
	}
	
	usdcAmount := estimatedETH * ethPrice
	
	return &GasEstimate{
		Chain:          "ethereum",
		EstimatedGas:   estimatedETH,
		NativeToken:    "ETH",
		TokenPriceUSD:  ethPrice,
		USDCAmount:     usdcAmount,
		Description:    fmt.Sprintf("Ethereum transaction gas: %.6f ETH (~$%.4f)", estimatedETH, usdcAmount),
	}, nil
}

// Sui gas估算
func estimateSuiGas(req common.ExecuteRequest) (*GasEstimate, error) {
	// Sui的gas费用通常很低
	baseGas := 0.001 // 0.001 SUI
	
	// 根据交易复杂度调整
	if len(req.TxData) > 500 {
		baseGas = 0.003 // 复杂交易
	}
	
	// 获取SUI价格
	suiPrice, err := getTokenPrice("sui")
	if err != nil {
		suiPrice = 1.5 // 假设SUI = $1.5
	}
	
	usdcAmount := baseGas * suiPrice
	
	return &GasEstimate{
		Chain:          "sui",
		EstimatedGas:   baseGas,
		NativeToken:    "SUI",
		TokenPriceUSD:  suiPrice,
		USDCAmount:     usdcAmount,
		Description:    fmt.Sprintf("Sui transaction gas: %.6f SUI (~$%.4f)", baseGas, usdcAmount),
	}, nil
}

// 从CoinGecko获取代币价格
func getTokenPrice(tokenId string) (float64, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", tokenId)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	
	var priceData map[string]PriceInfo
	if err := json.Unmarshal(body, &priceData); err != nil {
		return 0, err
	}
	
	if price, exists := priceData[tokenId]; exists {
		return price.USD, nil
	}
	
	return 0, fmt.Errorf("未找到代币价格: %s", tokenId)
}

// 添加一些安全边际和最小费用
func addSafetyMargin(estimate *GasEstimate) {
	// 添加20%的安全边际
	estimate.USDCAmount *= 1.2
	
	// 设置最小费用 (避免费用过低)
	minFee := 0.01 // 最少0.01 USDC
	if estimate.USDCAmount < minFee {
		estimate.USDCAmount = minFee
	}
	
	// 设置最大费用 (避免费用过高)
	maxFee := 10.0 // 最多10 USDC
	if estimate.USDCAmount > maxFee {
		estimate.USDCAmount = maxFee
	}
}
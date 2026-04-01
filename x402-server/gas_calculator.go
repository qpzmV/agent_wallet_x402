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
	common.LogInfo("开始估算 Gas 费用: chain=%s", req.Chain)
	
	switch req.Chain {
	case "solana":
		common.LogDebug("使用 Solana Gas 估算器")
		return estimateSolanaGas(req)
	case "evm", "ethereum":
		common.LogDebug("使用 EVM Gas 估算器")
		return estimateEVMGas(req)
	case "sui":
		common.LogDebug("使用 Sui Gas 估算器")
		return estimateSuiGas(req)
	default:
		common.LogError("不支持的链类型: %s", req.Chain)
		return nil, fmt.Errorf("不支持的链: %s", req.Chain)
	}
}

// Solana gas估算
func estimateSolanaGas(req common.ExecuteRequest) (*GasEstimate, error) {
	common.LogDebug("开始 Solana Gas 估算")
	
	// 基础交易费用约 0.000005 SOL
	baseGas := 0.000005
	
	// 根据交易复杂度调整
	// 这里可以解析 tx_data 来更精确估算
	complexityMultiplier := 1.0
	if len(req.TxData) > 1000 {
		complexityMultiplier = 2.0 // 复杂交易
		common.LogDebug("检测到复杂交易，调整 Gas 估算: multiplier=%.1f", complexityMultiplier)
	}
	
	estimatedSOL := baseGas * complexityMultiplier
	common.LogDebug("Solana Gas 估算: base=%.9f SOL, multiplier=%.1f, estimated=%.9f SOL", 
		baseGas, complexityMultiplier, estimatedSOL)
	
	// 获取SOL价格
	common.LogDebug("获取 SOL 价格")
	solPrice, err := getTokenPrice("solana")
	if err != nil {
		// 如果获取价格失败，使用默认价格
		solPrice = 100.0 // 假设SOL = $100
		common.LogWarn("获取 SOL 价格失败，使用默认价格: $%.2f", solPrice)
	} else {
		common.LogInfo("获取 SOL 价格成功: $%.2f", solPrice)
	}
	
	usdcAmount := estimatedSOL * solPrice
	
	result := &GasEstimate{
		Chain:          "solana",
		EstimatedGas:   estimatedSOL,
		NativeToken:    "SOL",
		TokenPriceUSD:  solPrice,
		USDCAmount:     usdcAmount,
		Description:    fmt.Sprintf("Solana transaction gas: %.6f SOL (~$%.4f)", estimatedSOL, usdcAmount),
	}
	
	common.LogInfo("Solana Gas 估算完成: %s", result.Description)
	return result, nil
}

// EVM gas估算 (Ethereum/Polygon等)
func estimateEVMGas(req common.ExecuteRequest) (*GasEstimate, error) {
	common.LogDebug("开始 EVM Gas 估算")
	
	// 基础gas limit: 21000 (简单转账)
	baseGasLimit := 21000.0
	
	// 根据交易类型调整
	if req.TargetAddress != "" && len(req.TxData) > 0 {
		baseGasLimit = 100000.0 // 合约调用
		common.LogDebug("检测到合约调用，调整 Gas Limit: %.0f", baseGasLimit)
	}
	
	// Gas price (gwei) - 这里应该从链上获取
	gasPriceGwei := 20.0 // 20 gwei
	gasPriceEth := gasPriceGwei / 1e9
	
	estimatedETH := (baseGasLimit * gasPriceEth)
	common.LogDebug("EVM Gas 估算: gasLimit=%.0f, gasPrice=%.1f gwei, estimated=%.9f ETH", 
		baseGasLimit, gasPriceGwei, estimatedETH)
	
	// 获取ETH价格
	common.LogDebug("获取 ETH 价格")
	ethPrice, err := getTokenPrice("ethereum")
	if err != nil {
		ethPrice = 2000.0 // 假设ETH = $2000
		common.LogWarn("获取 ETH 价格失败，使用默认价格: $%.2f", ethPrice)
	} else {
		common.LogInfo("获取 ETH 价格成功: $%.2f", ethPrice)
	}
	
	usdcAmount := estimatedETH * ethPrice
	
	result := &GasEstimate{
		Chain:          "ethereum",
		EstimatedGas:   estimatedETH,
		NativeToken:    "ETH",
		TokenPriceUSD:  ethPrice,
		USDCAmount:     usdcAmount,
		Description:    fmt.Sprintf("Ethereum transaction gas: %.6f ETH (~$%.4f)", estimatedETH, usdcAmount),
	}
	
	common.LogInfo("EVM Gas 估算完成: %s", result.Description)
	return result, nil
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
	common.LogDebug("请求代币价格: %s", url)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		common.LogError("请求代币价格失败: %v", err)
		return 0, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		common.LogError("读取价格响应失败: %v", err)
		return 0, err
	}
	
	var priceData map[string]PriceInfo
	if err := json.Unmarshal(body, &priceData); err != nil {
		common.LogError("解析价格数据失败: %v", err)
		return 0, err
	}
	
	if price, exists := priceData[tokenId]; exists {
		common.LogDebug("成功获取 %s 价格: $%.2f", tokenId, price.USD)
		return price.USD, nil
	}
	
	common.LogError("未找到代币价格: %s", tokenId)
	return 0, fmt.Errorf("未找到代币价格: %s", tokenId)
}

// 添加一些安全边际和最小费用
func addSafetyMargin(estimate *GasEstimate) {
	originalAmount := estimate.USDCAmount
	
	// 添加20%的安全边际
	estimate.USDCAmount *= 1.2
	common.LogDebug("添加 20%% 安全边际: %.6f -> %.6f USDC", originalAmount, estimate.USDCAmount)
	
	// 设置最小费用 (避免费用过低)
	minFee := 0.01 // 最少0.01 USDC
	if estimate.USDCAmount < minFee {
		common.LogDebug("应用最小费用限制: %.6f -> %.6f USDC", estimate.USDCAmount, minFee)
		estimate.USDCAmount = minFee
	}
	
	// 设置最大费用 (避免费用过高)
	maxFee := 10.0 // 最多10 USDC
	if estimate.USDCAmount > maxFee {
		common.LogWarn("应用最大费用限制: %.6f -> %.6f USDC", estimate.USDCAmount, maxFee)
		estimate.USDCAmount = maxFee
	}
	
	common.LogInfo("最终 Gas 费用: %.6f USDC (原始: %.6f)", estimate.USDCAmount, originalAmount)
}
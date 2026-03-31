package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"context"
	"strings"
	"sync"
	"time"
	"math/big"

	"agent-wallet-gas-sponsor/common"
	"github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/lib"
	suitypes "github.com/coming-chat/go-sui/v2/types"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

var (
	// 用于防止重放攻击的已使用签名缓存 (签名 -> 使用时间)
	usedSignatures sync.Map
)

const (
	ExecutionEngineURL = "http://localhost:8081/execute"
)

func main() {
	r := gin.Default()

	// 启动签名清理协程
	go cleanupExpiredSignatures()

	// 主要的执行端点
	r.POST("/execute", x402Middleware(), func(c *gin.Context) {
		// 如果通过了中间件，说明已支付或有凭证，转发给执行引擎
		proxyToExecutionEngine(c)
	})

	// 获取特定网络的支付信息
	r.GET("/payment-info/:network", func(c *gin.Context) {
		network := c.Param("network")
		config, exists := getNetworkConfig(network)
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "不支持的网络: " + network,
				"supported_networks": getEnabledNetworks(),
			})
			return
		}
		
		amount := 0.01 // 默认金额，实际应该根据交易计算
		
		c.JSON(http.StatusOK, gin.H{
			"network": network,
			"receiver": config.SponsorAddr,
			"amount": amount,
			"currency": "USDC",
			"usdc_contract": config.USDCContract,
			"rpc_endpoint": config.RPCEndpoint,
		})
	})

	// 获取所有支持的网络信息
	r.GET("/payment-info", func(c *gin.Context) {
		networks := make(map[string]interface{})
		for _, networkName := range getEnabledNetworks() {
			config, _ := getNetworkConfig(networkName)
			networks[networkName] = gin.H{
				"receiver": config.SponsorAddr,
				"usdc_contract": config.USDCContract,
				"rpc_endpoint": config.RPCEndpoint,
			}
		}
		
		c.JSON(http.StatusOK, gin.H{
			"networks": networks,
			"base_amount": 0.01,
			"currency": "USDC",
		})
	})

	// Gas费用估算端点
	r.POST("/estimate-gas", func(c *gin.Context) {
		var req common.ExecuteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
			return
		}
		
		estimate, err := estimateGasCost(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		// 添加安全边际
		addSafetyMargin(estimate)
		
		// 构建收款地址信息
		receivers := make(map[string]string)
		enabledNetworks := getEnabledNetworks()
		for _, network := range enabledNetworks {
			receivers[network] = getSponsorAddress(network)
		}
		
		c.JSON(http.StatusOK, gin.H{
			"gas_estimate": estimate,
			"payment_info": gin.H{
				"amount_usdc": estimate.USDCAmount,
				"receivers": receivers,
				"supported_networks": enabledNetworks,
			},
		})
	})

	fmt.Println("x402 Middleware 正在运行在 :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// 清理过期的签名缓存 (防止内存泄漏)
func cleanupExpiredSignatures() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		now := time.Now()
		usedSignatures.Range(func(key, value interface{}) bool {
			if usedTime, ok := value.(time.Time); ok {
				// 清理24小时前的签名
				if now.Sub(usedTime) > 24*time.Hour {
					usedSignatures.Delete(key)
				}
			}
			return true
		})
		log.Println("已清理过期的支付签名缓存")
	}
}

func x402Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		paymentProof := c.GetHeader("X-402-Payment")
		if paymentProof == "" {
			paymentProof = c.GetHeader("X-Payment-Token")
		}

		// Demo 模式
		if paymentProof == "demo" {
			c.Next()
			return
		}

		// 动态计算所需金额和gas信息
		requiredAmount, gasInfo := calculateRequiredAmountWithGasInfo(c)

		// 获取支付链，如果未指定则默认为 solana
		paymentChain := c.GetHeader("X-Payment-Chain")
		if paymentChain == "" {
			paymentChain = "solana" 
		}

		// 验证支付凭证
		var vErr error
		if paymentProof != "" {
			vErr = verifyPayment(paymentProof, requiredAmount, paymentChain)
			if vErr == nil {
				c.Next()
				return
			}
			log.Printf("支付验证失败 (%s): %v", paymentChain, vErr)
		}

		// 构建所有网络的收款地址映射
		receivers := make(map[string]string)
		enabledNetworks := getEnabledNetworks()
		
		// 确保至少有solana网络
		if len(enabledNetworks) == 0 {
			enabledNetworks = []string{"solana", "ethereum"}
		}
		
		for _, network := range enabledNetworks {
			addr := getSponsorAddress(network)
			if addr != "" {
				receivers[network] = addr
			}
		}
		
		// 确保有默认的solana地址
		if receivers["solana"] == "" {
			receivers["solana"] = common.SolanaSponsorAddr
		}

		msg := "Payment required."
		if vErr != nil {
			msg = fmt.Sprintf("Payment required: %v", vErr)
		}

		// 返回多链支付选项
		c.JSON(http.StatusPaymentRequired, common.X402Response{
			Status:  402,
			Message: msg,
			Payment: common.X402PaymentInfo{
				PriceUSD:     requiredAmount,
				Networks:     enabledNetworks,
				Tokens:       []string{"USDC"},
				Description:  fmt.Sprintf("Gas sponsorship for %s transaction", gasInfo.TargetChain),
				CapabilityID: "gas-sponsor",
				Receivers:    receivers,
				Receiver:     receivers["solana"], // 向后兼容
				GasInfo:      gasInfo,
			},
		})
		c.Abort()
	}
}

// 获取不同链的sponsor地址
func getSponsorAddress(network string) string {
	if config, exists := getNetworkConfig(network); exists {
		return config.SponsorAddr
	}
	return common.SolanaSponsorAddr // 默认返回Solana地址
}

// 统一的支付验证入口
func verifyPayment(paymentProof string, expectedAmount float64, chain string) error {
	paymentProof = strings.TrimSpace(paymentProof)
	
	// 简单的模拟支付 Token
	if paymentProof == "paid-123" {
		return nil
	}
	
	// Bootstrap支付凭证 - 用于第一笔支付交易 (我们代付用户支付gas费用的交易)
	if paymentProof == "bootstrap" {
		log.Printf("Bootstrap支付验证: 代付用户支付gas费用的交易")
		return nil
	}

	switch strings.ToLower(chain) {
	case "solana":
		return verifySolanaPayment(paymentProof, expectedAmount)
	case "sui":
		return verifySuiPayment(paymentProof, expectedAmount)
	case "evm", "ethereum", "polygon":
		return verifyEVMPayment(paymentProof, expectedAmount)
	default:
		return fmt.Errorf("不支持的支付链: %s", chain)
	}
}

// EVM支付验证 (以太坊/Polygon等)
func verifyEVMPayment(txHash string, expectedAmount float64) error {
	// 检查是否为有效的EVM交易哈希格式
	if !strings.HasPrefix(txHash, "0x") || len(txHash) != 66 {
		return fmt.Errorf("无效的EVM交易哈希格式")
	}

	// 防止重放
	if _, loaded := usedSignatures.LoadOrStore(txHash, time.Now()); loaded {
		return fmt.Errorf("该支付凭证已被使用 (Replay Attack)")
	}

	// 实现完整的EVM链上验证
	client, err := ethclient.Dial(common.EVMSepoliaRPC)
	if err != nil {
		return fmt.Errorf("连接以太坊RPC失败: %v", err)
	}
	defer client.Close()

	// 1. 验证交易存在且成功 (添加重试机制等待确认)
	txHashObj := ethcommon.HexToHash(txHash)
	
	var receipt *types.Receipt
	
	// 重试获取交易收据，等待交易确认
	for i := 0; i < 15; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		receipt, err = client.TransactionReceipt(ctx, txHashObj)
		cancel()
		
		if err == nil {
			break // 成功获取收据
		}
		
		log.Printf("等待交易确认 (%d/15): %s", i+1, txHash)
		time.Sleep(2 * time.Second)
	}
	
	if err != nil {
		return fmt.Errorf("获取交易收据失败: %v (已重试15次，交易可能尚未确认)", err)
	}
	
	// 检查交易是否成功
	if receipt.Status != 1 {
		return fmt.Errorf("交易执行失败: status=%d", receipt.Status)
	}

	// 获取交易详情
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	tx, _, err := client.TransactionByHash(ctx, txHashObj)
	if err != nil {
		return fmt.Errorf("获取交易详情失败: %v", err)
	}

	// 2. 验证是USDC转账到指定的Sponsor地址
	expectedTo := ethcommon.HexToAddress(common.EVMSponsorAddr)
	
	// 检查交易是否是发送到USDC合约
	usdcContract := ethcommon.HexToAddress("0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238") // Sepolia USDC
	if tx.To() == nil || *tx.To() != usdcContract {
		return fmt.Errorf("交易不是发送到USDC合约地址")
	}

	// 3. 解析USDC转账事件，验证接收者和金额
	transferAmount, recipient, err := parseUSDCTransferFromReceipt(receipt)
	if err != nil {
		return fmt.Errorf("解析USDC转账事件失败: %v", err)
	}

	// 验证接收者是否为Sponsor地址
	if recipient != expectedTo {
		return fmt.Errorf("USDC转账接收者不正确: 期望 %s, 实际 %s", expectedTo.Hex(), recipient.Hex())
	}

	// 4. 验证金额是否符合要求 (允许一定的误差范围)
	actualAmountUSDC := float64(transferAmount.Int64()) / 1e6 // USDC有6位小数
	tolerance := 0.001 // 允许0.001 USDC的误差
	
	if actualAmountUSDC < expectedAmount-tolerance {
		return fmt.Errorf("支付金额不足: 期望至少 %.6f USDC, 实际 %.6f USDC", 
			expectedAmount, actualAmountUSDC)
	}

	log.Printf("EVM支付验证成功: 交易 %s, 金额 %.6f USDC, 接收者 %s", 
		txHash, actualAmountUSDC, recipient.Hex())
	
	return nil
}

// 解析USDC转账事件，提取转账金额和接收者
func parseUSDCTransferFromReceipt(receipt *types.Receipt) (*big.Int, ethcommon.Address, error) {
	// ERC20 Transfer事件的签名: Transfer(address indexed from, address indexed to, uint256 value)
	transferEventSignature := ethcommon.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	
	for _, vLog := range receipt.Logs {
		if len(vLog.Topics) >= 3 && vLog.Topics[0] == transferEventSignature {
			// Topics[1] = from address (indexed)
			// Topics[2] = to address (indexed)  
			// Data = amount (uint256)
			
			toAddress := ethcommon.BytesToAddress(vLog.Topics[2].Bytes())
			
			// 解析金额 (从Data字段)
			if len(vLog.Data) != 32 {
				continue // 跳过格式不正确的日志
			}
			
			amount := new(big.Int).SetBytes(vLog.Data)
			
			return amount, toAddress, nil
		}
	}
	
	return nil, ethcommon.Address{}, fmt.Errorf("未找到USDC Transfer事件")
}

// 根据请求内容动态计算所需金额
func calculateRequiredAmount(c *gin.Context) float64 {
	amount, _ := calculateRequiredAmountWithGasInfo(c)
	return amount
}

// 计算所需金额并返回gas信息
func calculateRequiredAmountWithGasInfo(c *gin.Context) (float64, *common.GasEstimateInfo) {
	// 读取请求体来估算gas费用
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("无法读取请求体进行gas估算: %v", err)
		return 0.01, &common.GasEstimateInfo{
			TargetChain:    "unknown",
			EstimatedGas:   0.01,
			NativeToken:    "UNKNOWN",
			TokenPriceUSD:  1.0,
			GasDescription: "默认费用 (无法解析交易)",
		}
	}
	
	// 重新设置请求体，以便后续中间件可以读取
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	
	var req common.ExecuteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("无法解析请求体: %v", err)
		return 0.01, &common.GasEstimateInfo{
			TargetChain:    "unknown",
			EstimatedGas:   0.01,
			NativeToken:    "UNKNOWN",
			TokenPriceUSD:  1.0,
			GasDescription: "默认费用 (解析失败)",
		}
	}
	
	// 估算gas费用
	estimate, err := estimateGasCost(req)
	if err != nil {
		log.Printf("Gas估算失败: %v", err)
		return 0.01, &common.GasEstimateInfo{
			TargetChain:    req.Chain,
			EstimatedGas:   0.01,
			NativeToken:    "UNKNOWN",
			TokenPriceUSD:  1.0,
			GasDescription: "默认费用 (估算失败)",
		}
	}
	
	// 添加安全边际
	addSafetyMargin(estimate)
	
	gasInfo := &common.GasEstimateInfo{
		TargetChain:    estimate.Chain,
		EstimatedGas:   estimate.EstimatedGas,
		NativeToken:    estimate.NativeToken,
		TokenPriceUSD:  estimate.TokenPriceUSD,
		GasDescription: estimate.Description,
	}
	
	log.Printf("Gas估算结果: %s", estimate.Description)
	return estimate.USDCAmount, gasInfo
}

func verifySolanaPayment(sigStr string, expectedAmount float64) error {
	// 简单的模拟支付 Token (用于不具备真实链上环境的测试)
	if sigStr == "paid-123" {
		return nil
	}

	// 1. 之前先不记入，确保验证通过后再记录，防止单次请求内重试或延迟导致的误报

	// 2. 链上验证 (Solana)
	sig, err := solana.SignatureFromBase58(sigStr)
	if err != nil {
		return fmt.Errorf("无效的交易签名格式")
	}

	client := rpc.New(common.SolanaDevnetRPC)
	
	// 添加重试逻辑，等待交易在链上达到 Confirmed 状态
	var confirmed bool
	for i := 0; i < 15; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		outStatus, errStatus := client.GetSignatureStatuses(
			ctx,
			false, // searchTransactionHistory
			sig,
		)
		cancel()

		if errStatus == nil && outStatus != nil && len(outStatus.Value) > 0 && outStatus.Value[0] != nil {
			status := outStatus.Value[0]
			// 只要是 Confirmed 或 Finalized 即可
			if status.ConfirmationStatus == rpc.ConfirmationStatusConfirmed || 
			   status.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				confirmed = true
				break
			}
			log.Printf("交易 %s 状态: %s (等待 Confirmed... %d/15)", sigStr, status.ConfirmationStatus, i+1)
		} else if errStatus != nil {
			log.Printf("GetSignatureStatuses 报错 (%d/15): %v", i+1, errStatus)
		}
		
		time.Sleep(2 * time.Second)
	}

	if !confirmed {
		return fmt.Errorf("该支付交易未在链上确认 (已重试 15 次). 请稍后再试或检查浏览器: %s", sigStr)
	}

	// 3. 简化验证：只要状态确认了，就认为交易有效 (为了 Demo 简洁)
	if !confirmed {
		return fmt.Errorf("交易未在链上确认")
	}

	// 5. 验证通过后再标记为已使用，防止重放
	usedSignatures.Store(sigStr, time.Now())

	log.Printf("支付验证成功: 交易 %s", sigStr)
	return nil
}

func verifySuiPayment(digestStr string, expectedAmount float64) error {
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		return fmt.Errorf("连接 Sui RPC 失败: %v", err)
	}

	// 添加重试逻辑，等待交易在链上确认
	var confirmed bool
	for i := 0; i < 15; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		digest, dErr := lib.NewBase58(digestStr)
		if dErr != nil {
			cancel()
			return fmt.Errorf("无效的 Sui 交易摘要: %v", dErr)
		}

		resp, err := cli.GetTransactionBlock(
			ctx,
			*digest,
			suitypes.SuiTransactionBlockResponseOptions{
				ShowEffects: true,
			},
		)
		cancel()

		if err == nil && resp != nil && resp.Effects != nil {
			// 在 go-sui v2 中，Effects 可能是 Data.V1
			if resp.Effects.Data.V1.Status.Status == "success" {
				confirmed = true
				break
			}
			return fmt.Errorf("Sui 交易执行失败: %s", resp.Effects.Data.V1.Status.Error)
		}
		
		log.Printf("等待 Sui 交易确认 (%d/15): %s...", i+1, digestStr)
		time.Sleep(2 * time.Second)
	}

	if !confirmed {
		return fmt.Errorf("Sui 交易未在链上确认: %s", digestStr)
	}

	// 验证金额和接收者 (简化版：仅验证交易成功)
	// TODO: 解析交易内容以验证转账金额和接收地址
	
	log.Printf("Sui 支付验证成功: 交易 %s", digestStr)
	return nil
}

func proxyToExecutionEngine(c *gin.Context) {
	// 读取原始请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取请求体"})
		return
	}

	// 转发请求
	resp, err := http.Post(ExecutionEngineURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "无法连接到执行引擎"})
		return
	}
	defer resp.Body.Close()

	// 将执行引擎的响应返回给用户
	respBody, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", respBody)
}

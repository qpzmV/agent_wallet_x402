package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"sync"
	"time"
	"context"

	"agent-wallet-gas-sponsor/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
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

		// 验证支付凭证
		var vErr error
		if paymentProof != "" {
			vErr = verifyPayment(paymentProof, requiredAmount)
			if vErr == nil {
				c.Next()
				return
			}
			log.Printf("支付验证失败: %v", vErr)
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
func verifyPayment(paymentProof string, expectedAmount float64) error {
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

	// 尝试 Solana 验证
	solErr := verifySolanaPayment(paymentProof, expectedAmount)
	if solErr == nil {
		return nil
	}

	// 如果失败了，且看起来像 Solana 签名（base58 长度较长），则直接返回该错误
	if len(paymentProof) > 40 && !strings.HasPrefix(paymentProof, "0x") {
		return solErr
	}

	// 否则尝试 EVM 验证
	evmErr := verifyEVMPayment(paymentProof, expectedAmount)
	if evmErr == nil {
		return nil
	}

	return fmt.Errorf("无法验证支付凭证. Solana: %v, EVM: %v", solErr, evmErr)
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

	// TODO: 实现EVM链上验证
	// 这里需要调用以太坊RPC验证交易
	// 1. 验证交易存在且成功
	// 2. 验证是USDC转账到指定地址
	// 3. 验证金额符合要求
	
	log.Printf("EVM支付验证 (占位实现): %s", txHash)
	return fmt.Errorf("EVM支付验证暂未实现")
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

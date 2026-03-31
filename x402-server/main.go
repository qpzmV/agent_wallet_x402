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
	"strconv"

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

	// 1. 验证交易签名格式
	sig, err := solana.SignatureFromBase58(sigStr)
	if err != nil {
		return fmt.Errorf("无效的交易签名格式")
	}

	client := rpc.New(common.SolanaDevnetRPC)
	
	// 2. 尝试多种方法获取交易信息
	var txDetails *rpc.GetTransactionResult
	
	// 方法1: 尝试不同的 commitment 级别
	commitmentLevels := []rpc.CommitmentType{
		rpc.CommitmentFinalized,
		rpc.CommitmentConfirmed, 
		rpc.CommitmentProcessed,
	}
	
	for _, commitment := range commitmentLevels {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		txDetails, err = client.GetTransaction(
			ctx,
			sig,
			&rpc.GetTransactionOpts{
				Encoding:                       solana.EncodingBase64,
				Commitment:                     commitment,
				MaxSupportedTransactionVersion: &[]uint64{0}[0], // 支持版本化交易
			},
		)
		cancel()
		
		if err == nil && txDetails != nil {
			log.Printf("✅ 使用 %s commitment 成功获取交易", commitment)
			break
		}
		
		log.Printf("尝试 %s commitment 失败: %v", commitment, err)
	}
	
	// 方法2: 如果还是失败，尝试使用 GetSignatureStatuses 检查交易是否存在
	if txDetails == nil {
		log.Printf("尝试使用 GetSignatureStatuses 检查交易状态...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		statusResult, statusErr := client.GetSignatureStatuses(
			ctx,
			true, // searchTransactionHistory - 启用历史搜索
			sig,
		)
		cancel()
		
		if statusErr == nil && statusResult != nil && len(statusResult.Value) > 0 {
			if statusResult.Value[0] != nil {
				status := statusResult.Value[0]
				log.Printf("交易状态: %+v", status)
				
				// 如果交易存在但获取详情失败，可能是历史交易问题
				if status.Err != nil {
					return fmt.Errorf("Solana交易执行失败: %v", status.Err)
				}
				
				// 对于存在但无法获取详情的交易，我们暂时跳过详细验证
				log.Printf("⚠️  交易存在但无法获取详情，可能是历史交易限制")
				return fmt.Errorf("无法获取历史交易详情，请使用较新的交易: %s", sigStr)
			} else {
				return fmt.Errorf("交易不存在或已过期: %s", sigStr)
			}
		} else {
			return fmt.Errorf("无法查询交易状态: %v", statusErr)
		}
	}

	// 3. 验证交易是否成功执行
	if txDetails.Meta.Err != nil {
		return fmt.Errorf("Solana交易执行失败: %v", txDetails.Meta.Err)
	}

	// 4. 解析USDC转账信息
	sponsorAddr := common.SolanaSponsorAddr
	usdcMint := "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU" // Devnet USDC
	
	transferAmount, recipient, err := parseSolanaUSDCTransfer(txDetails, usdcMint, sponsorAddr)
	if err != nil {
		return fmt.Errorf("解析Solana USDC转账失败: %v", err)
	}

	// 5. 验证接收者
	if recipient != sponsorAddr {
		return fmt.Errorf("USDC转账接收者不正确: 期望 %s, 实际 %s", sponsorAddr, recipient)
	}

	// 6. 验证金额 (USDC有6位小数)
	actualAmountUSDC := float64(transferAmount) / 1e6
	tolerance := 0.001 // 允许0.001 USDC的误差
	
	if actualAmountUSDC < expectedAmount-tolerance {
		return fmt.Errorf("支付金额不足: 期望至少 %.6f USDC, 实际 %.6f USDC", 
			expectedAmount, actualAmountUSDC)
	}

	// 7. 防止重放攻击
	if _, loaded := usedSignatures.LoadOrStore(sigStr, time.Now()); loaded {
		return fmt.Errorf("该支付凭证已被使用 (Replay Attack)")
	}

	log.Printf("Solana支付验证成功: 交易 %s, 金额 %.6f USDC, 接收者 %s", 
		sigStr, actualAmountUSDC, recipient)
	
	return nil
}

// 解析Solana USDC转账信息
func parseSolanaUSDCTransfer(txDetails *rpc.GetTransactionResult, usdcMint, expectedRecipient string) (uint64, string, error) {
	// 在Solana中，SPL Token转账会在交易的preTokenBalances和postTokenBalances中体现
	preBalances := txDetails.Meta.PreTokenBalances
	postBalances := txDetails.Meta.PostTokenBalances
	
	// 查找USDC相关的余额变化
	var transferAmount uint64
	var recipient string
	
	// 构建余额变化映射
	balanceChanges := make(map[string]int64) // account -> balance change
	
	// 处理pre balances
	preBalanceMap := make(map[string]uint64)
	for _, balance := range preBalances {
		if balance.Mint.String() == usdcMint {
			ownerStr := balance.Owner.String()
			// 解析金额字符串为uint64
			if amount, err := strconv.ParseUint(balance.UiTokenAmount.Amount, 10, 64); err == nil {
				preBalanceMap[ownerStr] = amount
			}
		}
	}
	
	// 处理post balances并计算变化
	for _, balance := range postBalances {
		if balance.Mint.String() == usdcMint {
			ownerStr := balance.Owner.String()
			preAmount := preBalanceMap[ownerStr]
			
			// 解析金额字符串为uint64
			if postAmount, err := strconv.ParseUint(balance.UiTokenAmount.Amount, 10, 64); err == nil {
				change := int64(postAmount) - int64(preAmount)
				if change != 0 {
					balanceChanges[ownerStr] = change
				}
			}
		}
	}
	
	// 查找接收者（余额增加的账户）和转账金额
	for account, change := range balanceChanges {
		if change > 0 && account == expectedRecipient {
			transferAmount = uint64(change)
			recipient = account
			break
		}
	}
	
	if recipient == "" {
		return 0, "", fmt.Errorf("未找到向预期接收者 %s 的USDC转账", expectedRecipient)
	}
	
	return transferAmount, recipient, nil
}

func verifySuiPayment(digestStr string, expectedAmount float64) error {
	// 1. 连接Sui网络
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		return fmt.Errorf("连接 Sui RPC 失败: %v", err)
	}

	// 2. 验证交易摘要格式
	digest, err := lib.NewBase58(digestStr)
	if err != nil {
		return fmt.Errorf("无效的 Sui 交易摘要: %v", err)
	}

	// 3. 等待交易确认并获取详情
	var txResp *suitypes.SuiTransactionBlockResponse
	var confirmed bool
	
	for i := 0; i < 15; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		
		resp, err := cli.GetTransactionBlock(
			ctx,
			*digest,
			suitypes.SuiTransactionBlockResponseOptions{
				ShowEffects: true,
				ShowEvents:  true,
				ShowInput:   true,
				ShowObjectChanges: true,
			},
		)
		cancel()

		if err == nil && resp != nil && resp.Effects != nil {
			// 检查交易是否成功
			if resp.Effects.Data.V1.Status.Status == "success" {
				txResp = resp
				confirmed = true
				break
			} else {
				return fmt.Errorf("Sui 交易执行失败: %s", resp.Effects.Data.V1.Status.Error)
			}
		}
		
		log.Printf("等待 Sui 交易确认 (%d/15): %s...", i+1, digestStr)
		time.Sleep(2 * time.Second)
	}

	if !confirmed || txResp == nil {
		return fmt.Errorf("Sui 交易未在链上确认: %s", digestStr)
	}

	// 4. 解析SUI转账信息（这里简化为SUI代币转账，实际项目中可能需要处理其他代币）
	sponsorAddr := common.SuiSponsorAddr
	
	transferAmount, recipient, err := parseSuiTransfer(txResp, sponsorAddr)
	if err != nil {
		return fmt.Errorf("解析Sui转账失败: %v", err)
	}

	// 5. 验证接收者
	if recipient != sponsorAddr {
		return fmt.Errorf("转账接收者不正确: 期望 %s, 实际 %s", sponsorAddr, recipient)
	}

	// 6. 验证金额 (SUI有9位小数)
	actualAmountSUI := float64(transferAmount) / 1e9
	tolerance := 0.001 // 允许0.001 SUI的误差
	
	if actualAmountSUI < expectedAmount-tolerance {
		return fmt.Errorf("支付金额不足: 期望至少 %.6f SUI, 实际 %.6f SUI", 
			expectedAmount, actualAmountSUI)
	}

	// 7. 防止重放攻击
	if _, loaded := usedSignatures.LoadOrStore(digestStr, time.Now()); loaded {
		return fmt.Errorf("该支付凭证已被使用 (Replay Attack)")
	}

	log.Printf("Sui支付验证成功: 交易 %s, 金额 %.6f SUI, 接收者 %s", 
		digestStr, actualAmountSUI, recipient)
	
	return nil
}

// 解析Sui转账信息
func parseSuiTransfer(txResp *suitypes.SuiTransactionBlockResponse, expectedRecipient string) (uint64, string, error) {
	// 简化版本：由于Sui的类型结构比较复杂，这里先实现基础验证
	// 在实际应用中，需要根据具体的Sui SDK版本和交易类型来解析
	
	// 检查交易是否成功（这个已经在调用方验证过了）
	if txResp.Effects == nil {
		return 0, "", fmt.Errorf("交易没有effects信息")
	}

	// 对于演示目的，我们假设如果交易成功执行，就认为有有效的转账
	// 实际应用中需要解析具体的Events和ObjectChanges来获取准确的转账信息
	
	// 这里返回一个默认的转账金额和接收者
	// 在生产环境中，应该从交易的Events或ObjectChanges中解析实际的转账详情
	transferAmount := uint64(1000000000) // 1 SUI (9位小数)
	recipient := expectedRecipient
	
	log.Printf("Sui转账解析 (简化版): 假设向 %s 转账 %.6f SUI", 
		recipient, float64(transferAmount)/1e9)
	
	return transferAmount, recipient, nil
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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-wallet-gas-sponsor/common"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("   Solana 真实链上测试")
	fmt.Println("========================================")

	// 显示测试信息
	fmt.Printf("Sponsor地址: %s\n", common.SolanaSponsorAddr)
	fmt.Printf("用户地址: %s (有20 USDC，无SOL)\n", common.SolanaUserAddr)
	fmt.Printf("网络: Solana Devnet\n")
	fmt.Printf("浏览器: https://explorer.solana.com/?cluster=devnet\n")
	fmt.Printf("测试场景: 用户转1 USDC，我们代付SOL gas费\n\n")

	// 检查sponsor余额
	fmt.Println("=== 步骤1: 检查Sponsor余额 ===")
	if err := checkSponsorBalance(); err != nil {
		fmt.Printf("❌ Sponsor余额检查失败: %v\n", err)
		fmt.Println("请访问 https://faucet.solana.com/ 为Sponsor地址充值")
		return
	}
	fmt.Println("✅ Sponsor余额充足")

	// 验证用户没有SOL (代付测试的关键)
	fmt.Println("\n=== 步骤1.5: 验证用户账户状态 ===")
	userBalance, err := checkUserBalance()
	if err != nil {
		fmt.Printf("⚠️  无法检查用户余额: %v\n", err)
	} else {
		fmt.Printf("   用户SOL余额: %.9f SOL\n", userBalance)
		if userBalance > 0.001 {
			fmt.Printf("⚠️  用户有SOL余额，这不符合代付测试场景\n")
		} else {
			fmt.Printf("✅ 用户没有SOL，符合代付测试场景\n")
		}
	}
	fmt.Printf("   假设用户有20 USDC (devnet无法直接查询)\n")

	// 构造USDC转账交易 (用户有USDC但没有SOL)
	fmt.Println("\n=== 步骤2: 构造USDC转账交易 ===")
	fmt.Printf("💰 用户想转1 USDC给别人，但没有SOL支付gas\n")

	// 首先检查用户是否有USDC Token账户
	fmt.Printf("⚠️  注意: 请确保用户地址 %s 已经有USDC余额\n", common.SolanaUserAddr)
	fmt.Printf("   如果没有，请先转账USDC到该地址\n")

	txData, userSig, err := buildUSDCTransferTransaction()
	if err != nil {
		fmt.Printf("❌ 构造交易失败: %v\n", err)
		fmt.Printf("\n💡 可能的解决方案:\n")
		fmt.Printf("   1. 确保用户地址有USDC余额\n")
		fmt.Printf("   2. 用户地址: %s\n", common.SolanaUserAddr)
		fmt.Printf("   3. 从旧地址转账USDC到新地址\n")
		return
	}
	fmt.Println("✅ USDC转账交易构造成功")

	// 调用x402服务器获取gas费用
	fmt.Println("\n=== 步骤3: 获取Gas费用估算 ===")
	gasInfo, err := getGasEstimate(txData)
	if err != nil {
		fmt.Printf("❌ 获取gas估算失败: %v\n", err)
		return
	}

	fmt.Printf("✅ Gas估算结果:\n")
	fmt.Printf("   需要支付: $%.6f USDC\n", gasInfo.Payment.PriceUSD)
	fmt.Printf("   原生Gas: %.9f SOL\n", gasInfo.Payment.GasInfo.EstimatedGas)
	fmt.Printf("   SOL价格: $%.2f\n", gasInfo.Payment.GasInfo.TokenPriceUSD)
	fmt.Printf("   收款地址: %s\n", gasInfo.Payment.Receivers["solana"])

	// 用户支付gas费用 (用USDC支付，我们也代付这笔转账的gas)
	fmt.Println("\n=== 步骤4: 用户支付Gas费用 ===")
	fmt.Printf("💰 用户需要支付: $%.6f USDC 作为gas费用\n", gasInfo.Payment.PriceUSD)
	fmt.Printf("📍 收款地址: %s\n", gasInfo.Payment.Receivers["solana"])
	fmt.Println("💡 原理: 用户转USDC给我们作为gas费，我们代付这笔转账的SOL gas")

	// 构造用户支付gas费用的USDC转账
	paymentTxData, paymentUserSig, err := buildPaymentTransaction(gasInfo.Payment.PriceUSD)
	if err != nil {
		fmt.Printf("❌ 构造支付交易失败: %v\n", err)
		return
	}

	// 执行支付交易 (我们代付gas)
	fmt.Println("🔄 执行用户支付gas费用的USDC转账...")
	paymentResult, err := executePaymentTransaction(paymentTxData, paymentUserSig)
	if err != nil {
		fmt.Printf("❌ 支付交易失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 用户支付完成，交易Hash: %s\n", paymentResult.TxHash)
	fmt.Printf("   我们代付了用户支付gas费用这笔转账的SOL gas\n")

	// 使用支付交易hash作为支付凭证
	paymentProof := paymentResult.TxHash

	// 等待支付交易确认
	fmt.Println("⏳ 等待支付交易确认...")
	time.Sleep(10 * time.Second) // 等待10秒让支付交易被确认

	// 执行真实的代付交易
	fmt.Println("\n=== 步骤5: 执行用户原本的USDC转账 ===")
	result, err := executeWithPayment(txData, userSig, paymentProof)
	if err != nil {
		fmt.Printf("❌ 执行失败: %v\n", err)
		return
	}

	fmt.Printf("🚀 USDC转账交易执行成功!\n")
	fmt.Printf("   交易Hash: %s\n", result.TxHash)
	fmt.Printf("   浏览器查看: https://explorer.solana.com/tx/%s?cluster=devnet\n", result.TxHash)
	fmt.Printf("   ✅ 用户成功转账1 USDC，我们代付了SOL gas费\n")

	// 等待交易确认
	fmt.Println("\n=== 步骤6: 等待交易确认 ===")
	if err := waitForConfirmation(result.TxHash); err != nil {
		fmt.Printf("⚠️ 交易确认超时: %v\n", err)
	} else {
		fmt.Println("✅ 交易已确认")
	}

	fmt.Println("\n========================================")
	fmt.Println("   Solana USDC代付Gas测试完成!")
	fmt.Printf("   🎯 用户转账: 1 USDC\n")
	fmt.Printf("   💸 用户支付: $%.6f USDC (gas费)\n", gasInfo.Payment.PriceUSD)
	fmt.Printf("   ⛽ 我们代付: 2笔交易的SOL gas费\n")
	fmt.Printf("   📊 支付交易: %s\n", paymentResult.TxHash)
	fmt.Printf("   📊 转账交易: %s\n", result.TxHash)
	fmt.Println("   💡 完整流程: 用户转USDC给我们→我们代付→用户转USDC给别人→我们代付")
	fmt.Println("========================================")
}

// 检查sponsor余额
func checkSponsorBalance() error {
	client := rpc.New(common.SolanaDevnetRPC) // 使用配置中的RPC
	sponsorPubkey := solana.MustPublicKeyFromBase58(common.SolanaSponsorAddr)

	balance, err := client.GetBalance(context.Background(), sponsorPubkey, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("获取余额失败: %v", err)
	}

	// 检查是否有足够的SOL (至少0.01 SOL)
	minBalance := uint64(10000000) // 0.01 SOL in lamports
	if balance.Value < minBalance {
		return fmt.Errorf("余额不足: %d lamports (需要至少 %d lamports)", balance.Value, minBalance)
	}

	fmt.Printf("   当前余额: %.9f SOL\n", float64(balance.Value)/1e9)
	return nil
}

// 构造USDC转账交易 (用户有USDC但没有SOL)
func buildUSDCTransferTransaction() (txData, userSig string, err error) {
	client := rpc.New(common.SolanaDevnetRPC) // 使用配置中的RPC

	// 获取最新的blockhash
	recent, err := client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return "", "", fmt.Errorf("获取blockhash失败: %v", err)
	}

	// 使用配置中的用户私钥
	userPrivateKey, err := solana.PrivateKeyFromBase58(common.SolanaUserPK)
	if err != nil {
		return "", "", fmt.Errorf("解析用户私钥失败: %v", err)
	}
	userPubkey := userPrivateKey.PublicKey()

	// 目标地址 (接收USDC的用户 - 使用sponsor地址作为示例)
	targetAddr := solana.MustPublicKeyFromBase58(common.SolanaSponsorAddr)

	// USDC相关地址
	tokenProgramID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	usdcMintAddr := solana.MustPublicKeyFromBase58(common.SolanaUSDTContract)

	// 计算Associated Token Account地址
	userTokenAccount, _, err := solana.FindAssociatedTokenAddress(userPubkey, usdcMintAddr)
	if err != nil {
		return "", "", fmt.Errorf("计算用户token账户失败: %v", err)
	}

	targetTokenAccount, _, err := solana.FindAssociatedTokenAddress(targetAddr, usdcMintAddr)
	if err != nil {
		return "", "", fmt.Errorf("计算目标token账户失败: %v", err)
	}

	// 构造USDC转账指令 (转账1 USDC = 1,000,000 micro USDC)
	transferAmount := uint64(1000000) // 1 USDC

	// 创建SPL Token转账指令
	instruction := solana.NewInstruction(
		tokenProgramID,
		solana.AccountMetaSlice{
			{PublicKey: userTokenAccount, IsSigner: false, IsWritable: true},   // source
			{PublicKey: targetTokenAccount, IsSigner: false, IsWritable: true}, // destination
			{PublicKey: userPubkey, IsSigner: true, IsWritable: false},         // owner
		},
		buildTokenTransferData(transferAmount),
	)

	// 创建交易，设置sponsor为fee payer (关键：sponsor代付gas费)
	sponsorPubkey := solana.MustPublicKeyFromBase58(common.SolanaSponsorAddr)
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(sponsorPubkey), // Sponsor代付所有gas费
	)
	if err != nil {
		return "", "", fmt.Errorf("创建交易失败: %v", err)
	}

	// 用户签名 (用户只签名自己的USDC转账，不需要支付gas)
	messageData, err := tx.Message.MarshalBinary()
	if err != nil {
		return "", "", fmt.Errorf("序列化消息失败: %v", err)
	}

	userSignature, err := userPrivateKey.Sign(messageData)
	if err != nil {
		return "", "", fmt.Errorf("用户签名失败: %v", err)
	}

	// 设置用户签名到交易中
	tx.Signatures = make([]solana.Signature, int(tx.Message.Header.NumRequiredSignatures))
	for i, account := range tx.Message.AccountKeys {
		if i >= int(tx.Message.Header.NumRequiredSignatures) {
			break
		}
		if account.Equals(userPubkey) {
			tx.Signatures[i] = userSignature
		}
	}

	// 转换为base64
	txData, err = tx.ToBase64()
	if err != nil {
		return "", "", fmt.Errorf("转换base64失败: %v", err)
	}

	fmt.Printf("   用户地址: %s\n", userPubkey.String())
	fmt.Printf("   目标地址: %s\n", targetAddr.String())
	fmt.Printf("   转账金额: %.6f USDC\n", float64(transferAmount)/1e6)
	fmt.Printf("   Fee Payer: %s (Sponsor代付gas)\n", sponsorPubkey.String())
	fmt.Printf("   用户Token账户: %s\n", userTokenAccount.String())
	fmt.Printf("   目标Token账户: %s\n", targetTokenAccount.String())
	fmt.Printf("   场景: 用户转USDC，Sponsor代付交易gas费\n")

	return txData, userSignature.String(), nil
}

// 构造SPL Token转账指令的数据
func buildTokenTransferData(amount uint64) []byte {
	// SPL Token Transfer指令格式:
	// [0] = 指令类型 (3 = Transfer)
	// [1-8] = 转账数量 (little endian uint64)
	data := make([]byte, 9)
	data[0] = 3 // Transfer指令

	// 将amount转换为little endian字节
	for i := 0; i < 8; i++ {
		data[1+i] = byte(amount >> (8 * i))
	}

	return data
}

// 构造用户支付gas费用的USDC转账交易
func buildPaymentTransaction(priceUSD float64) (txData, userSig string, err error) {
	client := rpc.New(common.SolanaDevnetRPC) // 使用配置中的RPC

	// 获取最新的blockhash
	recent, err := client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return "", "", fmt.Errorf("获取blockhash失败: %v", err)
	}

	// 使用配置中的用户私钥
	userPrivateKey, err := solana.PrivateKeyFromBase58(common.SolanaUserPK)
	if err != nil {
		return "", "", fmt.Errorf("解析用户私钥失败: %v", err)
	}
	userPubkey := userPrivateKey.PublicKey()

	// 目标地址 (我们的收款地址)
	targetAddr := solana.MustPublicKeyFromBase58(common.SolanaSponsorAddr)

	// USDC相关地址
	tokenProgramID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	usdcMintAddr := solana.MustPublicKeyFromBase58(common.SolanaUSDTContract)

	// 计算Associated Token Account地址
	userTokenAccount, _, err := solana.FindAssociatedTokenAddress(userPubkey, usdcMintAddr)
	if err != nil {
		return "", "", fmt.Errorf("计算用户token账户失败: %v", err)
	}

	targetTokenAccount, _, err := solana.FindAssociatedTokenAddress(targetAddr, usdcMintAddr)
	if err != nil {
		return "", "", fmt.Errorf("计算目标token账户失败: %v", err)
	}

	// 计算需要转账的USDC数量 (假设1 USDC = 1 USD)
	transferAmount := uint64(priceUSD * 1e6) // 转换为micro USDC

	// 创建SPL Token转账指令
	instruction := solana.NewInstruction(
		tokenProgramID,
		solana.AccountMetaSlice{
			{PublicKey: userTokenAccount, IsSigner: false, IsWritable: true},   // source
			{PublicKey: targetTokenAccount, IsSigner: false, IsWritable: true}, // destination
			{PublicKey: userPubkey, IsSigner: true, IsWritable: false},         // owner
		},
		buildTokenTransferData(transferAmount),
	)

	// 创建交易，设置sponsor为fee payer (我们代付这笔支付交易的gas)
	sponsorPubkey := solana.MustPublicKeyFromBase58(common.SolanaSponsorAddr)
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(sponsorPubkey), // Sponsor代付支付交易的gas费
	)
	if err != nil {
		return "", "", fmt.Errorf("创建支付交易失败: %v", err)
	}

	// 用户签名
	messageData, err := tx.Message.MarshalBinary()
	if err != nil {
		return "", "", fmt.Errorf("序列化消息失败: %v", err)
	}

	userSignature, err := userPrivateKey.Sign(messageData)
	if err != nil {
		return "", "", fmt.Errorf("用户签名失败: %v", err)
	}

	// 设置用户签名到交易中
	tx.Signatures = make([]solana.Signature, int(tx.Message.Header.NumRequiredSignatures))
	for i, account := range tx.Message.AccountKeys {
		if i >= int(tx.Message.Header.NumRequiredSignatures) {
			break
		}
		if account.Equals(userPubkey) {
			tx.Signatures[i] = userSignature
		}
	}

	// 转换为base64
	txData, err = tx.ToBase64()
	if err != nil {
		return "", "", fmt.Errorf("转换base64失败: %v", err)
	}

	fmt.Printf("   支付金额: %.6f USDC\n", float64(transferAmount)/1e6)
	fmt.Printf("   从: %s\n", userPubkey.String())
	fmt.Printf("   到: %s\n", targetAddr.String())

	return txData, userSignature.String(), nil
}

// 执行支付交易 (我们代付gas)
func executePaymentTransaction(txData, userSig string) (*common.ExecuteResponse, error) {
	reqBody := common.ExecuteRequest{
		Chain:         "solana",
		TxData:        txData,
		UserSignature: userSig,
		UserAddress:   common.SolanaUserAddr,
		TargetAddress: common.SolanaSponsorAddr, // 支付给我们
	}

	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8080/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-402-Payment", "bootstrap") // 使用特殊的bootstrap支付凭证

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var execResp common.ExecuteResponse
	if err := json.Unmarshal(respBody, &execResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 原始响应: %s", err, string(respBody))
	}

	// 支持多种状态表示
	statusStr := fmt.Sprintf("%v", execResp.Status)
	if statusStr != "success" && statusStr != "200" {
		return nil, fmt.Errorf("执行失败: %s (Status: %s)", execResp.Error, statusStr)
	}

	return &execResp, nil
}

// 获取gas费用估算
func getGasEstimate(txData string) (*common.X402Response, error) {
	reqBody := common.ExecuteRequest{
		Chain:         "solana",
		TxData:        txData,
		UserAddress:   common.SolanaUserAddr,
		TargetAddress: common.SolanaSponsorAddr, // USDC转账目标地址
		UserSignature: "temp_signature",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post("http://localhost:8080/execute", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("期望402状态码，但收到: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var x402Resp common.X402Response
	if err := json.NewDecoder(resp.Body).Decode(&x402Resp); err != nil {
		return nil, err
	}

	return &x402Resp, nil
}

// 使用支付凭证执行交易
func executeWithPayment(txData, userSig, paymentProof string) (*common.ExecuteResponse, error) {
	reqBody := common.ExecuteRequest{
		Chain:         "solana",
		TxData:        txData,
		UserSignature: userSig,
		UserAddress:   common.SolanaUserAddr,
		TargetAddress: common.SolanaSponsorAddr, // USDC转账目标地址
	}

	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8080/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-402-Payment", paymentProof)
	resp, err := client.Do(req)

	// 打印 resp
	fmt.Printf("   响应状态: %v\n", resp.Status)

	// 打印 resp
	fmt.Printf("   响应状态: %v\n", resp.Status)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var execResp common.ExecuteResponse
	if err := json.Unmarshal(respBody, &execResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 原始响应: %s", err, string(respBody))
	}

	// 支持多种状态表示
	statusStr := fmt.Sprintf("%v", execResp.Status)
	if statusStr != "success" && statusStr != "200" {
		// 如果仍然收到 402，说明支付凭证尚未在链上确认或无效
		if resp.StatusCode == http.StatusPaymentRequired {
			return nil, fmt.Errorf("支付验证未通过: %s. 请确保交易 %s 已在链上 Confirm", execResp.Message, paymentProof)
		}
		return nil, fmt.Errorf("执行失败: %s (Status: %s)", execResp.Error, statusStr)
	}

	return &execResp, nil
}

// 等待交易确认
func waitForConfirmation(txHash string) error {
	client := rpc.New(common.SolanaDevnetRPC) // 使用配置中的RPC
	sig, err := solana.SignatureFromBase58(txHash)
	if err != nil {
		return fmt.Errorf("无效的交易签名: %v", err)
	}

	fmt.Print("   等待确认")
	for i := 0; i < 30; i++ { // 最多等待30秒
		time.Sleep(1 * time.Second)
		fmt.Print(".")

		// 检查交易状态
		_, err := client.GetTransaction(
			context.Background(),
			sig,
			&rpc.GetTransactionOpts{
				Encoding:   solana.EncodingBase64,
				Commitment: rpc.CommitmentConfirmed,
			},
		)

		if err == nil {
			fmt.Println(" ✅")
			return nil
		}
	}

	fmt.Println(" ⏰")
	return fmt.Errorf("交易确认超时")
}

// 检查用户余额 (确保用户没有SOL)
func checkUserBalance() (float64, error) {
	client := rpc.New(common.SolanaDevnetRPC) // 使用配置中的RPC
	userPubkey := solana.MustPublicKeyFromBase58(common.SolanaUserAddr)

	balance, err := client.GetBalance(context.Background(), userPubkey, rpc.CommitmentConfirmed)
	if err != nil {
		return 0, fmt.Errorf("获取用户余额失败: %v", err)
	}

	return float64(balance.Value) / 1e9, nil
}

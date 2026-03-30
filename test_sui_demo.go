package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-wallet-gas-sponsor/common"

	"github.com/coming-chat/go-sui/v2/account"
	"github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/lib"
	"github.com/coming-chat/go-sui/v2/sui_types"
	"github.com/fardream/go-bcs/bcs"
)

const (
	SuiCoinType    = "0x2::sui::SUI"
	MaxGasBudget   = uint64(100_000_000) // 0.1 SUI
)

func main() {
	fmt.Println("🚀 SUI Gas代付演示测试 (使用SUI代替USDC)")
	fmt.Println("========================================")

	fmt.Printf("Sponsor地址: %s\n", common.SuiSponsorAddr)
	fmt.Printf("用户地址: %s\n", common.SuiUserAddr)
	fmt.Printf("网络: SUI Testnet\n")
	fmt.Printf("浏览器: %s\n", common.SuiBrowser)
	fmt.Printf("测试场景: 用户转0.01 SUI，Sponsor代付gas费\n\n")

	// === 步骤1: 检查余额 ===
	fmt.Println("=== 步骤1: 检查账户余额 ===")
	if err := checkBalances(); err != nil {
		fmt.Printf("❌ 余额检查失败: %v\n", err)
		return
	}

	// === 步骤2: 构造SUI转账交易 ===
	fmt.Println("\n=== 步骤2: 构造SUI转账交易 ===")
	recipientAddr := common.SuiSponsorAddr // 转给sponsor作为演示
	transferAmount := uint64(10_000_000)   // 0.01 SUI

	txData, userSig, err := buildSUITransferTx(recipientAddr, transferAmount)
	if err != nil {
		fmt.Printf("❌ 构造SUI转账交易失败: %v\n", err)
		return
	}
	fmt.Println("✅ SUI转账交易构造成功")

	// === 步骤3: 获取Gas费用估算 ===
	fmt.Println("\n=== 步骤3: 获取Gas费用估算 ===")
	gasInfo, err := getGasEstimate(txData)
	if err != nil {
		fmt.Printf("❌ 获取gas估算失败: %v\n", err)
		return
	}

	fmt.Printf("✅ Gas估算结果:\n")
	fmt.Printf("   需要支付: $%.6f USD\n", gasInfo.Payment.PriceUSD)
	if gasInfo.Payment.GasInfo != nil {
		fmt.Printf("   原生Gas: %.6f SUI\n", gasInfo.Payment.GasInfo.EstimatedGas)
		fmt.Printf("   SUI价格: $%.4f\n", gasInfo.Payment.GasInfo.TokenPriceUSD)
	}

	// === 步骤4: 执行交易 (使用bootstrap模式) ===
	fmt.Println("\n=== 步骤4: 执行SUI转账 (bootstrap代付) ===")
	fmt.Printf("💰 转账金额: %.6f SUI\n", float64(transferAmount)/1e9)
	fmt.Println("💡 使用bootstrap模式，Sponsor直接代付gas费")

	result, err := executeTransaction(txData, userSig, "bootstrap")
	if err != nil {
		fmt.Printf("❌ 执行失败: %v\n", err)
		return
	}

	fmt.Printf("🚀 SUI转账交易执行成功!\n")
	fmt.Printf("   交易Digest: %s\n", result.TxHash)
	fmt.Printf("   浏览器查看: %s%s\n", common.SuiBrowser, result.TxHash)
	fmt.Printf("   ✅ 用户成功转账%.6f SUI，Sponsor代付了gas费\n", float64(transferAmount)/1e9)

	// === 步骤5: 等待确认 ===
	fmt.Println("\n=== 步骤5: 等待交易确认 ===")
	if err := waitForConfirmation(result.TxHash); err != nil {
		fmt.Printf("⚠️  交易确认超时: %v\n", err)
	} else {
		fmt.Println("✅ 交易已确认")
	}

	fmt.Println("\n========================================")
	fmt.Println("   SUI Gas代付演示完成!")
	fmt.Printf("   🎯 用户转账: %.6f SUI\n", float64(transferAmount)/1e9)
	fmt.Printf("   ⛽ Sponsor代付: SUI gas费\n")
	fmt.Printf("   📊 交易链接: %s%s\n", common.SuiBrowser, result.TxHash)
	fmt.Println("   💡 演示原理: 用户签名交易，Sponsor提供gas coin")
	fmt.Println("========================================")
}

// checkBalances 检查账户余额
func checkBalances() error {
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		return fmt.Errorf("连接 Sui RPC 失败: %v", err)
	}

	sponsorAddr, _ := sui_types.NewAddressFromHex(common.SuiSponsorAddr)
	userAddr, _ := sui_types.NewAddressFromHex(common.SuiUserAddr)

	// 检查 Sponsor 余额
	sponsorBalance, err := cli.GetBalance(context.Background(), *sponsorAddr, SuiCoinType)
	if err != nil {
		return fmt.Errorf("获取Sponsor余额失败: %v", err)
	}
	sponsorSUI := sponsorBalance.TotalBalance.BigInt().Uint64()
	fmt.Printf("   Sponsor SUI余额: %.6f SUI\n", float64(sponsorSUI)/1e9)

	if sponsorSUI < 100_000_000 { // 0.1 SUI
		return fmt.Errorf("Sponsor余额不足，需要至少0.1 SUI")
	}

	// 检查用户余额
	userBalance, err := cli.GetBalance(context.Background(), *userAddr, SuiCoinType)
	if err != nil {
		return fmt.Errorf("获取用户余额失败: %v", err)
	}
	userSUI := userBalance.TotalBalance.BigInt().Uint64()
	fmt.Printf("   用户 SUI余额: %.6f SUI\n", float64(userSUI)/1e9)

	if userSUI < 10_000_000 { // 0.01 SUI
		return fmt.Errorf("用户余额不足，需要至少0.01 SUI用于转账")
	}

	return nil
}

// buildSUITransferTx 构造SUI转账交易，使用GetAllCoins避免GetCoins的hex问题
func buildSUITransferTx(recipientHex string, amount uint64) (txBase64 string, userSigBase64 string, err error) {
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		return "", "", fmt.Errorf("连接 Sui RPC 失败: %v", err)
	}

	// 用户账户
	seed, err := hex.DecodeString(common.SuiUserPK)
	if err != nil {
		return "", "", fmt.Errorf("解析用户私钥失败: %v", err)
	}
	scheme, _ := sui_types.NewSignatureScheme(0)
	userAcc := account.NewAccount(scheme, seed)

	userAddr, _ := sui_types.NewAddressFromHex(userAcc.Address)
	sponsorAddr, _ := sui_types.NewAddressFromHex(common.SuiSponsorAddr)
	recipientAddr, _ := sui_types.NewAddressFromHex(recipientHex)

	// 获取 gas price
	gasPriceResp, err := cli.GetReferenceGasPrice(context.Background())
	if err != nil {
		return "", "", fmt.Errorf("获取 gas price 失败: %v", err)
	}
	gasPrice := gasPriceResp.Uint64()

	// 使用GetAllBalances来避免GetCoins的hex问题
	// 然后手动构造ObjectRef
	sponsorBalances, err := cli.GetAllBalances(context.Background(), *sponsorAddr)
	if err != nil {
		return "", "", fmt.Errorf("获取Sponsor余额失败: %v", err)
	}

	var sponsorSUIBalance uint64
	for _, bal := range sponsorBalances {
		if bal.CoinType == SuiCoinType {
			sponsorSUIBalance = bal.TotalBalance.BigInt().Uint64()
			break
		}
	}

	if sponsorSUIBalance < MaxGasBudget {
		return "", "", fmt.Errorf("Sponsor SUI余额不足支付gas")
	}

	userBalances, err := cli.GetAllBalances(context.Background(), *userAddr)
	if err != nil {
		return "", "", fmt.Errorf("获取用户余额失败: %v", err)
	}

	var userSUIBalance uint64
	for _, bal := range userBalances {
		if bal.CoinType == SuiCoinType {
			userSUIBalance = bal.TotalBalance.BigInt().Uint64()
			break
		}
	}

	if userSUIBalance < amount {
		return "", "", fmt.Errorf("用户SUI余额不足: 有%.6f SUI, 需要%.6f SUI", 
			float64(userSUIBalance)/1e9, float64(amount)/1e9)
	}

	// 由于GetCoins有问题，我们使用一个简化的方法
	// 创建一个简单的SUI转账交易，让SUI网络自动处理coin合并
	ptb := sui_types.NewProgrammableTransactionBuilder()

	// 使用TransferObjects方法进行SUI转账
	// 这需要我们先分割coins
	amountArg := ptb.Pure(amount)
	splitCoins := ptb.SplitCoins(sui_types.SUI_GAS_OBJECT_ARG, []sui_types.Argument{amountArg})
	ptb.TransferObjects([]sui_types.Argument{splitCoins}, ptb.Pure(*recipientAddr))

	pt := ptb.Finish()

	// 创建一个虚拟的gas coin reference
	// 注意：这是一个简化的实现，实际应用中需要真实的ObjectRef
	dummyObjectId := sui_types.ObjectId{0x1, 0x2, 0x3} // 占位符
	dummyVersion := sui_types.SequenceNumber(1)
	dummyDigest := sui_types.ObjectDigest{0x4, 0x5, 0x6} // 占位符

	txDataObj := sui_types.TransactionData{
		V1: &sui_types.TransactionDataV1{
			Kind: sui_types.TransactionKind{
				ProgrammableTransaction: &pt,
			},
			Sender: *userAddr,
			GasData: sui_types.GasData{
				Payment: []*sui_types.ObjectRef{
					{
						ObjectId: dummyObjectId,
						Version:  dummyVersion,
						Digest:   dummyDigest,
					},
				},
				Owner:  *sponsorAddr,
				Price:  gasPrice,
				Budget: MaxGasBudget,
			},
			Expiration: sui_types.TransactionExpiration{
				None: &lib.EmptyEnum{},
			},
		},
	}

	// BCS 编码
	bcsBytes, err := bcs.Marshal(txDataObj)
	if err != nil {
		return "", "", fmt.Errorf("BCS 编码失败: %v", err)
	}

	// 用户签名
	sig, err := userAcc.SignSecureWithoutEncode(bcsBytes, sui_types.DefaultIntent())
	if err != nil {
		return "", "", fmt.Errorf("用户签名失败: %v", err)
	}

	txBase64 = base64.StdEncoding.EncodeToString(bcsBytes)
	if sig.Ed25519SuiSignature != nil {
		userSigBase64 = base64.StdEncoding.EncodeToString(sig.Ed25519SuiSignature.Signature[:])
	}

	return txBase64, userSigBase64, nil
}

// getGasEstimate 调用 x402 服务器获取 gas 费用估算
func getGasEstimate(txData string) (*common.X402Response, error) {
	reqBody := common.ExecuteRequest{
		Chain:         "sui",
		TxData:        txData,
		UserAddress:   common.SuiUserAddr,
		TargetAddress: common.SuiSponsorAddr,
		UserSignature: "temp_signature",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post("http://localhost:8080/execute", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("请求 x402 服务器失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("期望402状态码，但收到: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var x402Resp common.X402Response
	if err := json.NewDecoder(resp.Body).Decode(&x402Resp); err != nil {
		return nil, fmt.Errorf("解析402响应失败: %v", err)
	}

	return &x402Resp, nil
}

// executeTransaction 执行交易
func executeTransaction(txData, userSig, paymentProof string) (*common.ExecuteResponse, error) {
	reqBody := common.ExecuteRequest{
		Chain:         "sui",
		TxData:        txData,
		UserSignature: userSig,
		UserAddress:   common.SuiUserAddr,
		TargetAddress: common.SuiSponsorAddr,
	}

	jsonBody, _ := json.Marshal(reqBody)

	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8080/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-402-Payment", paymentProof)
	req.Header.Set("X-Payment-Chain", "sui")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	fmt.Printf("   [响应] 状态: %d\n", resp.StatusCode)

	var execResp common.ExecuteResponse
	if err := json.Unmarshal(respBody, &execResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("执行失败: %s", execResp.Error)
	}

	return &execResp, nil
}

// waitForConfirmation 等待交易确认
func waitForConfirmation(digestStr string) error {
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		return fmt.Errorf("连接 Sui RPC 失败: %v", err)
	}

	digest, err := lib.NewBase58(digestStr)
	if err != nil {
		return fmt.Errorf("无效的交易 Digest: %v", err)
	}

	fmt.Print("   等待确认")
	for i := 0; i < 20; i++ {
		time.Sleep(2 * time.Second)
		fmt.Print(".")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := cli.GetTransactionBlock(ctx, *digest, types.SuiTransactionBlockResponseOptions{
			ShowEffects: true,
		})
		cancel()

		if err == nil && resp != nil && resp.Effects != nil {
			if resp.Effects.Data.V1.Status.Status == "success" {
				fmt.Println(" ✅")
				return nil
			}
			fmt.Println(" ❌")
			return fmt.Errorf("交易失败: %s", resp.Effects.Data.V1.Status.Error)
		}
	}

	fmt.Println(" ⏰")
	return fmt.Errorf("交易确认超时")
}
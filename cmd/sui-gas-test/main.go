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
	"github.com/coming-chat/go-sui/v2/types"
	"github.com/fardream/go-bcs/bcs"
)

// SUI Testnet USDC contract (Wormhole bridged USDC on Sui Testnet)
// 查 https://suiscan.xyz/testnet/ 获取真实地址；这里用 SUI 作为演示代币
// 真实 USDC 地址需要根据测试网部署更新
const (
	SuiUSDCType    = "0xa1ec7fc00a6f40db9693ad1415d0c193ad3906494428cf252621037bd7117e29::usdc::USDC"
	SuiCoinType    = "0x2::sui::SUI"
	MaxGasBudget   = uint64(100_000_000) // 0.1 SUI
)

func main() {
	fmt.Println("========================================")
	fmt.Println("   SUI USDC Gas代付完整测试")
	fmt.Println("========================================")

	fmt.Printf("Sponsor地址: %s\n", common.SuiSponsorAddr)
	fmt.Printf("用户地址: %s (有USDC，无SUI)\n", common.SuiUserAddr)
	fmt.Printf("网络: SUI Testnet\n")
	fmt.Printf("浏览器: %s\n", common.SuiBrowser)
	fmt.Printf("测试场景: 用户转1 USDC，Sponsor代付SUI gas费\n\n")

	// === 步骤1: 检查 Sponsor SUI 余额 ===
	fmt.Println("=== 步骤1: 检查Sponsor SUI余额 ===")
	if err := checkSponsorBalance(); err != nil {
		fmt.Printf("❌ Sponsor余额检查失败: %v\n", err)
		fmt.Println("请访问 https://faucet.testnet.sui.io/ 为Sponsor地址充值SUI")
		return
	}
	fmt.Println("✅ Sponsor SUI余额充足")

	// === 步骤2: 构造用户目标交易 (USDC 转账给别人) ===
	fmt.Println("\n=== 步骤2: 构造用户USDC转账交易 ===")
	fmt.Printf("💰 用户想转1 USDC给别人，但没有SUI支付gas\n")
	fmt.Printf("⚠️  注意: 请确保用户地址 %s 已有USDC余额\n", common.SuiUserAddr)

	// 目标接收者 (这里用 sponsor 地址演示)
	recipientAddr := common.SuiSponsorAddr
	transferAmount := uint64(1_000_000) // 1 USDC (6 decimals)

	txData, _, err := buildUSDCTransferTx(recipientAddr, transferAmount)
	if err != nil {
		fmt.Printf("❌ 构造USDC转账交易失败: %v\n", err)
		fmt.Printf("\n💡 可能的解决方案:\n")
		fmt.Printf("   1. 确保用户地址 %s 有USDC余额\n", common.SuiUserAddr)
		fmt.Printf("   2. 查询账户: https://suiscan.xyz/testnet/account/%s\n", common.SuiUserAddr)
		return
	}
	fmt.Println("✅ USDC转账交易构造成功")

	// === 步骤3: 获取 Gas 费用估算 ===
	fmt.Println("\n=== 步骤3: 获取Gas费用估算 ===")
	gasInfo, err := getGasEstimate(txData)
	if err != nil {
		fmt.Printf("❌ 获取gas估算失败: %v\n", err)
		return
	}

	fmt.Printf("✅ Gas估算结果:\n")
	fmt.Printf("   需要支付: $%.6f USDC\n", gasInfo.Payment.PriceUSD)
	if gasInfo.Payment.GasInfo != nil {
		fmt.Printf("   原生Gas: %.6f SUI\n", gasInfo.Payment.GasInfo.EstimatedGas)
		fmt.Printf("   SUI价格: $%.4f\n", gasInfo.Payment.GasInfo.TokenPriceUSD)
	}
	if addr, ok := gasInfo.Payment.Receivers["sui"]; ok {
		fmt.Printf("   SUI收款地址: %s\n", addr)
	}

	// === 步骤4: 用户支付 Gas 费用 (转USDC给我们) ===
	fmt.Println("\n=== 步骤4: 用户支付Gas费用 (USDC → Sponsor) ===")
	fmt.Printf("💰 用户需要支付: $%.6f USDC 作为gas费\n", gasInfo.Payment.PriceUSD)
	fmt.Println("💡 原理: 用户转USDC给我们，我们代付这笔转账的SUI gas")

	paymentAmount := uint64(gasInfo.Payment.PriceUSD * 1_000_000)
	if paymentAmount < 10_000 {
		paymentAmount = 10_000 // 最少 0.01 USDC
	}

	paymentTxData, paymentUserSig, err := buildUSDCTransferTx(common.SuiSponsorAddr, paymentAmount)
	if err != nil {
		fmt.Printf("❌ 构造支付交易失败: %v\n", err)
		return
	}

	fmt.Println("🔄 执行用户支付gas费的USDC转账 (bootstrap代付)...")
	paymentResult, err := executeTransaction(paymentTxData, paymentUserSig, "bootstrap")
	if err != nil {
		fmt.Printf("❌ 支付交易失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 用户支付完成!\n")
	fmt.Printf("   支付交易Digest: %s\n", paymentResult.TxHash)
	fmt.Printf("   浏览器查看: %s%s\n", common.SuiBrowser, paymentResult.TxHash)
	fmt.Printf("   我们代付了用户支付gas费的SUI gas\n")

	paymentProof := paymentResult.TxHash

	// === 步骤5: 用支付凭证执行原本的 USDC 转账 ===
	fmt.Println("\n=== 步骤5: 执行用户原本的USDC转账 ===")
	fmt.Println("🔄 等待支付交易链上确认后，执行原交易...")
	time.Sleep(3 * time.Second)

	// 重新构造交易，因为第一笔交易可能改变了 coin objects 的版本
	fmt.Println("   🔄 重新构造交易以获取最新的 coin objects...")
	freshTxData, freshUserSig, err := buildUSDCTransferTx(recipientAddr, transferAmount)
	if err != nil {
		fmt.Printf("❌ 重新构造交易失败: %v\n", err)
		return
	}

	result, err := executeTransaction(freshTxData, freshUserSig, paymentProof)
	if err != nil {
		fmt.Printf("❌ 执行失败: %v\n", err)
		return
	}

	fmt.Printf("🚀 USDC转账交易执行成功!\n")
	fmt.Printf("   交易Digest: %s\n", result.TxHash)
	fmt.Printf("   浏览器查看: %s%s\n", common.SuiBrowser, result.TxHash)
	fmt.Printf("   ✅ 用户成功转账%.6f USDC，我们代付了SUI gas费\n", float64(transferAmount)/1e6)

	// === 步骤6: 等待确认 ===
	fmt.Println("\n=== 步骤6: 等待交易确认 ===")
	if err := waitForConfirmation(result.TxHash); err != nil {
		fmt.Printf("⚠️  交易确认超时: %v\n", err)
	} else {
		fmt.Println("✅ 交易已确认")
	}

	fmt.Println("\n========================================")
	fmt.Println("   SUI USDC代付Gas测试完成!")
	fmt.Printf("   🎯 用户转账: %.6f USDC\n", float64(transferAmount)/1e6)
	fmt.Printf("   💸 用户支付: $%.6f USDC (gas费)\n", gasInfo.Payment.PriceUSD)
	fmt.Printf("   ⛽ 我们代付: 2笔交易的SUI gas费\n")
	fmt.Printf("   📊 支付交易: %s%s\n", common.SuiBrowser, paymentResult.TxHash)
	fmt.Printf("   📊 转账交易: %s%s\n", common.SuiBrowser, result.TxHash)
	fmt.Println("   💡 完整流程: 用户转USDC给我们→我们代付→用户转USDC给别人→我们代付")
	fmt.Println("========================================")
}

// checkSponsorBalance 检查 Sponsor 账户的 SUI 余额是否充足
func checkSponsorBalance() error {
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		return fmt.Errorf("连接 Sui RPC 失败: %v", err)
	}

	sponsorAddr, err := sui_types.NewAddressFromHex(common.SuiSponsorAddr)
	if err != nil {
		return fmt.Errorf("解析 Sponsor 地址失败: %v", err)
	}

	balance, err := cli.GetBalance(context.Background(), *sponsorAddr, SuiCoinType)
	if err != nil {
		return fmt.Errorf("获取余额失败: %v", err)
	}

	// TotalBalance is decimal.Decimal
	suiBalance := balance.TotalBalance.BigInt().Uint64()
	minBalance := uint64(100_000_000) // 0.1 SUI
	if suiBalance < minBalance {
		return fmt.Errorf("余额不足: %d MIST = %.4f SUI (需要至少 0.1 SUI)", suiBalance, float64(suiBalance)/1e9)
	}

	fmt.Printf("   当前SUI余额: %.6f SUI\n", float64(suiBalance)/1e9)
	return nil
}

// buildUSDCTransferTx 构造 SUI 链上 USDC 转账交易（Sponsor 代付 gas）
// 返回 base64(BCS) 交易数据 和 base64 用户签名
func buildUSDCTransferTx(recipientHex string, amount uint64) (txBase64 string, userSigBase64 string, err error) {
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		return "", "", fmt.Errorf("连接 Sui RPC 失败: %v", err)
	}

	// 用户账户
	seed, err := hex.DecodeString(common.SuiUserPK)
	if err != nil {
		return "", "", fmt.Errorf("解析用户私钥失败: %v", err)
	}
	scheme, _ := sui_types.NewSignatureScheme(0) // Ed25519
	userAcc := account.NewAccount(scheme, seed)

	userAddr, err := sui_types.NewAddressFromHex(userAcc.Address)
	if err != nil {
		return "", "", fmt.Errorf("解析用户地址失败: %v", err)
	}

	sponsorAddr, err := sui_types.NewAddressFromHex(common.SuiSponsorAddr)
	if err != nil {
		return "", "", fmt.Errorf("解析 Sponsor 地址失败: %v", err)
	}

	recipientAddr, err := sui_types.NewAddressFromHex(recipientHex)
	if err != nil {
		return "", "", fmt.Errorf("解析接收者地址失败: %v", err)
	}

	// 获取 gas price
	gasPriceResp, err := cli.GetReferenceGasPrice(context.Background())
	if err != nil {
		return "", "", fmt.Errorf("获取 gas price 失败: %v", err)
	}
	gasPrice := gasPriceResp.Uint64()

	// 获取 Sponsor 的 gas coin (SUI) - 使用 limit=2 避免 hex 解析问题
	sponsorCoins, err := cli.GetCoins(context.Background(), *sponsorAddr, nil, nil, 2)
	if err != nil {
		return "", "", fmt.Errorf("获取 Sponsor SUI coins 失败: %v", err)
	}
	
	if len(sponsorCoins.Data) == 0 {
		return "", "", fmt.Errorf("Sponsor 没有 SUI gas coin，请先充值SUI")
	}

	// 获取用户的 USDC coins
	coinType := SuiUSDCType
	userUSDCCoins, err := cli.GetCoins(context.Background(), *userAddr, &coinType, nil, 10)
	if err != nil {
		return "", "", fmt.Errorf("获取用户 USDC coins 失败: %v", err)
	}
	
	if len(userUSDCCoins.Data) == 0 {
		// 尝试获取用户的所有余额来检查是否有USDC
		allBalances, balErr := cli.GetAllBalances(context.Background(), *userAddr)
		if balErr != nil {
			return "", "", fmt.Errorf("用户没有 USDC coins，且无法查询余额: %v", balErr)
		}
		
		var hasUSDC bool
		var usdcBalance uint64
		for _, bal := range allBalances {
			if bal.CoinType == SuiUSDCType {
				hasUSDC = true
				usdcBalance = bal.TotalBalance.BigInt().Uint64()
				break
			}
		}
		
		if !hasUSDC || usdcBalance == 0 {
			return "", "", fmt.Errorf("用户没有 USDC coins，请先充值USDC到 %s", userAcc.Address)
		}
		
		return "", "", fmt.Errorf("用户有USDC余额(%.6f USDC)但找不到USDC coins", float64(usdcBalance)/1e6)
	}

	// 检查用户 USDC 余额
	totalUSDC := uint64(0)
	for _, coin := range userUSDCCoins.Data {
		totalUSDC += coin.Balance.Uint64()
	}
	if totalUSDC < amount {
		return "", "", fmt.Errorf("用户 USDC 余额不足: 有 %d (%.6f USDC), 需要 %d (%.6f USDC)",
			totalUSDC, float64(totalUSDC)/1e6, amount, float64(amount)/1e6)
	}

	fmt.Printf("   用户USDC余额: %.6f USDC\n", float64(totalUSDC)/1e6)
	fmt.Printf("   转账金额: %.6f USDC\n", float64(amount)/1e6)
	fmt.Printf("   接收者: %s\n", recipientHex)

	// 使用 ptb.Pay() 高层方法进行 USDC 转账
	ptb := sui_types.NewProgrammableTransactionBuilder()

	// 收集 USDC coin refs
	var usdcCoinRefs []*sui_types.ObjectRef
	for i := range userUSDCCoins.Data {
		ref := userUSDCCoins.Data[i].Reference()
		usdcCoinRefs = append(usdcCoinRefs, ref)
	}

	if err := ptb.Pay(
		usdcCoinRefs,
		[]sui_types.SuiAddress{*recipientAddr},
		[]uint64{amount},
	); err != nil {
		return "", "", fmt.Errorf("构造USDC转账指令失败: %v", err)
	}

	pt := ptb.Finish()

	txDataObj := sui_types.TransactionData{
		V1: &sui_types.TransactionDataV1{
			Kind: sui_types.TransactionKind{
				ProgrammableTransaction: &pt,
			},
			Sender: *userAddr,
			GasData: sui_types.GasData{
				Payment: []*sui_types.ObjectRef{
					sponsorCoins.Data[0].Reference(),
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
		return nil, fmt.Errorf("请求 x402 服务器失败 (确保服务已启动): %v", err)
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

// executeTransaction 通用交易执行函数，支持 bootstrap 和 SUI digest 支付凭证
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
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("   [响应内容] %s\n", string(respBody))
	}

	var execResp common.ExecuteResponse
	if err := json.Unmarshal(respBody, &execResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 原始: %s", err, string(respBody))
	}

	statusStr := fmt.Sprintf("%v", execResp.Status)
	if statusStr != "success" && statusStr != "200" {
		if resp.StatusCode == http.StatusPaymentRequired {
			return nil, fmt.Errorf("支付验证未通过: %s. 凭证: %s", execResp.Message, paymentProof)
		}
		return nil, fmt.Errorf("执行失败: %s (Status: %s)", execResp.Error, statusStr)
	}

	return &execResp, nil
}

// waitForConfirmation 等待 SUI 交易确认
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
	return fmt.Errorf("交易确认超时 (40秒)")
}

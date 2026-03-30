package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"agent-wallet-gas-sponsor/common"

	"github.com/coming-chat/go-sui/v2/account"
	"github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/lib"
	"github.com/coming-chat/go-sui/v2/sui_types"
	"github.com/fardream/go-bcs/bcs"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("========================================")
	fmt.Println("   x402 Gas 代付系统 - 端到端测试")
	fmt.Println("========================================")

	// 显示固定用户地址和充值提示
	showUserAddressesAndUSDTInfo()

	fmt.Print("\n确认已充值USDT？输入 'continue' 继续: ")
	waitForContinue(reader)

	// 测试流程选择
	fmt.Println("\n请选择要测试的链:")
	fmt.Println("1. Solana (Devnet)")
	fmt.Println("2. Ethereum (Sepolia)")
	fmt.Println("3. Sui (Testnet)")
	fmt.Print("请输入选择 (1-3): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		runSolanaTest(reader)
	case "2":
		runEthereumTest(reader)
	case "3":
		runSuiTest(reader)
	default:
		fmt.Println("无效选择，退出程序")
		return
	}

	fmt.Println("\n[测试完成] 端到端测试流程已完成。")
}

func showUserAddressesAndUSDTInfo() {
	fmt.Println("\n[用户地址信息] 请向以下地址充值USDT进行测试:")
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ Ethereum (Sepolia): %s │\n", common.EVMUserAddr)
	fmt.Printf("│ USDT合约: %s │\n", common.EVMUSDTContract)
	fmt.Println("│ 充值方式: 使用Sepolia测试网水龙头获取ETH，然后兑换USDT        │")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ Solana (Devnet): %s   │\n", common.SolanaUserAddr)
	fmt.Printf("│ USDT合约: %s │\n", common.SolanaUSDTContract)
	fmt.Println("│ 充值方式: 使用Solana Devnet水龙头获取SOL，然后兑换USDT       │")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ Sui (Testnet): %s │\n", common.SuiUserAddr)
	fmt.Printf("│ USDT合约: %s        │\n", common.SuiUSDTContract)
	fmt.Println("│ 充值方式: 使用Sui Testnet水龙头获取SUI，然后兑换USDT        │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	
	fmt.Println("\n[Sponsor地址信息] 以下地址用于接收USDT并代付gas:")
	fmt.Printf("• Ethereum Sponsor: %s\n", common.EVMSponsorAddr)
	fmt.Printf("• Solana Sponsor: %s\n", common.SolanaSponsorAddr)
	fmt.Printf("• Sui Sponsor: %s\n", common.SuiSponsorAddr)
}

func waitForContinue(reader *bufio.Reader) {
	for {
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(input) == "continue" {
			break
		}
		fmt.Print("无效输入。请输入 'continue': ")
	}
}

func runSolanaTest(reader *bufio.Reader) {
	fmt.Println("\n--- Solana 测试流程 ---")
	
	// 1. 构造目标交易 (用户想要执行的交易)
	fmt.Println("[步骤1] 构造Solana交易 - 用户想要转账1000 lamports")
	txData, userSig, err := buildSolanaTransaction()
	if err != nil {
		fmt.Printf("❌ 构造交易失败: %v\n", err)
		return
	}
	
	// 2. 调用x402服务器获取gas费用
	fmt.Println("[步骤2] 调用x402服务器获取gas费用估算...")
	gasInfo, err := getGasEstimate("solana", txData, common.SolanaUserAddr)
	if err != nil {
		fmt.Printf("❌ 获取gas估算失败: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Gas估算结果: 需要支付 $%.4f USDT\n", gasInfo.Payment.PriceUSD)
	fmt.Printf("   原因: %s\n", gasInfo.Payment.Description)
	
	// 安全地获取收款地址
	var receiverAddr string
	if gasInfo.Payment.Receivers != nil && gasInfo.Payment.Receivers["solana"] != "" {
		receiverAddr = gasInfo.Payment.Receivers["solana"]
	} else if gasInfo.Payment.Receiver != "" {
		receiverAddr = gasInfo.Payment.Receiver
	} else {
		receiverAddr = common.SolanaSponsorAddr // 使用默认地址
	}
	fmt.Printf("   收款地址: %s\n", receiverAddr)
	
	// 3. 模拟用户支付USDT
	fmt.Print("\n[步骤3] 模拟用户支付USDT到Sponsor地址...")
	fmt.Print("按回车继续模拟支付完成: ")
	reader.ReadString('\n')
	
	// 4. 使用支付凭证调用执行
	fmt.Println("[步骤4] 使用支付凭证调用x402服务器执行代付交易...")
	result, err := executeWithPayment("solana", txData, userSig, common.SolanaUserAddr, "paid-123")
	if err != nil {
		fmt.Printf("❌ 执行失败: %v\n", err)
		return
	}
	
	fmt.Printf("🚀 代付交易执行成功!\n")
	fmt.Printf("   交易Hash: %s\n", result.TxHash)
	fmt.Printf("   浏览器查看: %s\n", fmt.Sprintf(common.SolanaBrowser, result.TxHash))
}

func runEthereumTest(reader *bufio.Reader) {
	fmt.Println("\n--- Ethereum 测试流程 ---")
	
	// 1. 构造目标交易
	fmt.Println("[步骤1] 构造Ethereum交易 - 用户想要转账0.01 ETH")
	txData := "0x1234567890abcdef" // 模拟交易数据
	userSig := "0xabcdef1234567890" // 模拟用户签名
	
	// 2. 调用x402服务器获取gas费用
	fmt.Println("[步骤2] 调用x402服务器获取gas费用估算...")
	gasInfo, err := getGasEstimate("ethereum", txData, common.EVMUserAddr)
	if err != nil {
		fmt.Printf("❌ 获取gas估算失败: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Gas估算结果: 需要支付 $%.4f USDT\n", gasInfo.Payment.PriceUSD)
	fmt.Printf("   原因: %s\n", gasInfo.Payment.Description)
	
	// 安全地获取收款地址
	var receiverAddr string
	if gasInfo.Payment.Receivers != nil && gasInfo.Payment.Receivers["ethereum"] != "" {
		receiverAddr = gasInfo.Payment.Receivers["ethereum"]
	} else if gasInfo.Payment.Receiver != "" {
		receiverAddr = gasInfo.Payment.Receiver
	} else {
		receiverAddr = common.EVMSponsorAddr // 使用默认地址
	}
	fmt.Printf("   收款地址: %s\n", receiverAddr)
	
	// 3. 模拟用户支付USDT
	fmt.Print("\n[步骤3] 模拟用户支付USDT到Sponsor地址...")
	fmt.Print("按回车继续模拟支付完成: ")
	reader.ReadString('\n')
	
	// 4. 使用支付凭证调用执行
	fmt.Println("[步骤4] 使用支付凭证调用x402服务器执行代付交易...")
	result, err := executeWithPayment("ethereum", txData, userSig, common.EVMUserAddr, "paid-123")
	if err != nil {
		fmt.Printf("❌ 执行失败: %v\n", err)
		return
	}
	
	fmt.Printf("🚀 代付交易执行成功!\n")
	fmt.Printf("   交易Hash: %s\n", result.TxHash)
	fmt.Printf("   浏览器查看: %s%s\n", common.EVMBrowser, result.TxHash)
}

func runSuiTest(reader *bufio.Reader) {
	fmt.Println("\n--- Sui 测试流程 ---")
	
	// 1. 构造目标交易
	fmt.Println("[步骤1] 构造Sui交易 - 用户想要转账100 MIST")
	txData, userSig, err := buildSuiSponsoredTx()
	if err != nil {
		fmt.Printf("❌ 构造交易失败: %v\n", err)
		return
	}
	
	// 2. 调用x402服务器获取gas费用
	fmt.Println("[步骤2] 调用x402服务器获取gas费用估算...")
	gasInfo, err := getGasEstimate("sui", txData, common.SuiUserAddr)
	if err != nil {
		fmt.Printf("❌ 获取gas估算失败: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Gas估算结果: 需要支付 $%.4f USDT\n", gasInfo.Payment.PriceUSD)
	fmt.Printf("   原因: %s\n", gasInfo.Payment.Description)
	
	// 安全地获取收款地址 (用户可以选择任意支持的网络支付)
	var receiverAddr string
	if gasInfo.Payment.Receivers != nil && gasInfo.Payment.Receivers["solana"] != "" {
		receiverAddr = gasInfo.Payment.Receivers["solana"]
	} else if gasInfo.Payment.Receiver != "" {
		receiverAddr = gasInfo.Payment.Receiver
	} else {
		receiverAddr = common.SolanaSponsorAddr // 使用默认地址
	}
	fmt.Printf("   收款地址: %s (可选择任意支持的网络)\n", receiverAddr)
	
	// 3. 模拟用户支付USDT
	fmt.Print("\n[步骤3] 模拟用户支付USDT到Sponsor地址...")
	fmt.Print("按回车继续模拟支付完成: ")
	reader.ReadString('\n')
	
	// 4. 使用支付凭证调用执行
	fmt.Println("[步骤4] 使用支付凭证调用x402服务器执行代付交易...")
	result, err := executeWithPayment("sui", txData, userSig, common.SuiUserAddr, "paid-123")
	if err != nil {
		fmt.Printf("❌ 执行失败: %v\n", err)
		return
	}
	
	fmt.Printf("🚀 代付交易执行成功!\n")
	fmt.Printf("   交易Hash: %s\n", result.TxHash)
	fmt.Printf("   浏览器查看: %s%s\n", common.SuiBrowser, result.TxHash)
}

// 获取gas费用估算
func getGasEstimate(chain, txData, userAddr string) (*common.X402Response, error) {
	reqBody := common.ExecuteRequest{
		Chain:         chain,
		TxData:        txData,
		UserAddress:   userAddr,
		TargetAddress: "11111111111111111111111111111112", // 使用有效的Solana地址
		UserSignature: "temp_signature",
	}
	
	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post("http://localhost:8080/execute", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusPaymentRequired {
		return nil, fmt.Errorf("期望402状态码，但收到: %d", resp.StatusCode)
	}
	
	var x402Resp common.X402Response
	if err := json.NewDecoder(resp.Body).Decode(&x402Resp); err != nil {
		return nil, err
	}
	
	return &x402Resp, nil
}

// 使用支付凭证执行交易
func executeWithPayment(chain, txData, userSig, userAddr, paymentProof string) (*common.ExecuteResponse, error) {
	var targetAddr string
	switch chain {
	case "solana":
		targetAddr = "11111111111111111111111111111112"
	case "ethereum":
		targetAddr = "0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e"
	case "sui":
		targetAddr = common.SuiSponsorAddr
	default:
		targetAddr = "0xTARGET_ADDR"
	}
	
	reqBody := common.ExecuteRequest{
		Chain:         chain,
		TxData:        txData,
		UserSignature: userSig,
		UserAddress:   userAddr,
		TargetAddress: targetAddr,
	}
	
	jsonBody, _ := json.Marshal(reqBody)
	
	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8080/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-402-Payment", paymentProof)
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// 读取原始响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	
	// 打印原始响应用于调试
	fmt.Printf("   [调试] 执行引擎响应: %s\n", string(respBody))
	
	// 尝试解析为ExecuteResponse
	var execResp common.ExecuteResponse
	if err := json.Unmarshal(respBody, &execResp); err != nil {
		// 如果解析失败，尝试解析为通用响应
		var genericResp map[string]interface{}
		if jsonErr := json.Unmarshal(respBody, &genericResp); jsonErr == nil {
			// 手动构造ExecuteResponse
			execResp = common.ExecuteResponse{
				Status: "unknown",
				Error:  fmt.Sprintf("响应格式异常: %s", string(respBody)),
			}
			
			// 尝试提取字段
			if status, ok := genericResp["status"]; ok {
				if statusStr, ok := status.(string); ok {
					execResp.Status = statusStr
				}
			}
			if txHash, ok := genericResp["tx_hash"]; ok {
				if txHashStr, ok := txHash.(string); ok {
					execResp.TxHash = txHashStr
				}
			}
			if errorMsg, ok := genericResp["error"]; ok {
				if errorStr, ok := errorMsg.(string); ok {
					execResp.Error = errorStr
				}
			}
		} else {
			return nil, fmt.Errorf("JSON解析失败: %v, 原始响应: %s", err, string(respBody))
		}
	}
	
	// 注意：这里不要检查status是否为success，因为可能是failed但仍然是有效响应
	return &execResp, nil
}

// 构造Solana交易
func buildSolanaTransaction() (txData, userSig string, err error) {
	c := rpc.New(rpc.DevNet_RPC)
	recent, err := c.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return "", "", err
	}
	
	// 使用固定的用户私钥 (在实际应用中应该安全管理)
	user := solana.NewWallet()
	target := solana.MustPublicKeyFromBase58("11111111111111111111111111111112")
	
	tx, _ := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(1000, user.PublicKey(), target).Build(),
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(solana.MustPublicKeyFromBase58(common.SolanaSponsorAddr)),
	)
	
	tx.Signatures = make([]solana.Signature, int(tx.Message.Header.NumRequiredSignatures))
	data, _ := tx.Message.MarshalBinary()
	sig, _ := user.PrivateKey.Sign(data)
	
	for i := 0; i < int(tx.Message.Header.NumRequiredSignatures); i++ {
		if tx.Message.AccountKeys[i].Equals(user.PublicKey()) {
			tx.Signatures[i] = sig
		}
	}
	
	txData, _ = tx.ToBase64()
	userSig = sig.String()
	
	return txData, userSig, nil
}

func buildSuiSponsoredTx() (txBase64 string, userSigBase64 string, err error) {
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		return "", "", err
	}

	// 使用固定的用户账户
	seed, _ := hex.DecodeString(common.SuiUserPK)
	scheme, _ := sui_types.NewSignatureScheme(0)
	userAcc := account.NewAccount(scheme, seed)

	// sponsor 地址
	sponsorAddr, _ := sui_types.NewAddressFromHex(common.SuiSponsorAddr)

	// 查询 gas price
	gasPriceResp, err := cli.GetReferenceGasPrice(context.Background())
	if err != nil {
		return "", "", err
	}
	gasPrice := gasPriceResp.Uint64()

	// sponsor gas coin
	sponsorCoins, err := cli.GetCoins(
		context.Background(),
		*sponsorAddr,
		nil,
		nil,
		1,
	)
	if err != nil || len(sponsorCoins.Data) == 0 {
		return "", "", fmt.Errorf("sponsor 没有 gas coin")
	}

	// 构造交易
	recipient, _ := sui_types.NewAddressFromHex(common.SuiSponsorAddr)
	amount := uint64(100)

	ptb := sui_types.NewProgrammableTransactionBuilder()
	ptb.TransferSui(*recipient, &amount)
	pt := ptb.Finish()

	userSuiAddr, _ := sui_types.NewAddressFromHex(userAcc.Address)

	txDataObj := sui_types.TransactionData{
		V1: &sui_types.TransactionDataV1{
			Kind: sui_types.TransactionKind{
				ProgrammableTransaction: &pt,
			},
			Sender: *userSuiAddr,
			GasData: sui_types.GasData{
				Payment: []*sui_types.ObjectRef{
					sponsorCoins.Data[0].Reference(),
				},
				Owner:  *sponsorAddr,
				Price:  gasPrice,
				Budget: 100000000,
			},
			Expiration: sui_types.TransactionExpiration{
				None: &lib.EmptyEnum{},
			},
		},
	}

	// BCS 编码
	bcsBytes, err := bcs.Marshal(txDataObj)
	if err != nil {
		return "", "", err
	}

	// 用户签名
	sig, err := userAcc.SignSecureWithoutEncode(
		bcsBytes,
		sui_types.DefaultIntent(),
	)
	if err != nil {
		return "", "", err
	}

	txBase64 = base64.StdEncoding.EncodeToString(bcsBytes)
	if sig.Ed25519SuiSignature != nil {
		userSigBase64 = base64.StdEncoding.EncodeToString(sig.Ed25519SuiSignature.Signature[:])
	}

	return txBase64, userSigBase64, nil
}
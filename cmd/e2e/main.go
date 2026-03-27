package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	fmt.Println("   x402 Gas 代付系统 - 端到端交互测试")
	fmt.Println("========================================")

	// 1. 显示固定账户并提示领币
	fmt.Println("\n[准备阶段] 请确保以下 Sponsor 账户在对应测试链上有余额：")
	fmt.Printf("EVM (Sepolia): %s\n", common.EVMSponsorAddr)
	fmt.Printf("Solana (Devnet): %s\n", common.SolanaSponsorAddr)
	fmt.Printf("Sui (Testnet): %s\n", common.SuiSponsorAddr)

	fmt.Print("\n确认已领好测试币？输入 'continue' 继续: ")
	waitForContinue(reader)

	// 2. 测试 EVM (Sepolia)
	if promptChoice(reader, "EVM (Sepolia)") {
		runTest(reader, "evm", "Sepolia", common.EVMBrowser)
	}

	// 3. 测试 Solana (Devnet)
	if promptChoice(reader, "Solana (Devnet)") {
		runTest(reader, "solana", "Devnet", common.SolanaBrowser)
	}

	// 4. 测试 Sui (Testnet)
	if promptChoice(reader, "Sui (Testnet)") {
		runTest(reader, "sui", "Testnet", common.SuiBrowser)
	}

	fmt.Println("\n[测试完成] 选定的流程已模拟/验证完毕。")
}

func promptChoice(reader *bufio.Reader, target string) bool {
	fmt.Printf("\n>>> 准备测试 %s. 输入 'continue' 开始, 或输入 'skip' 跳过: ", target)
	for {
		input, _ := reader.ReadString('\n')
		cmd := strings.TrimSpace(input)
		if cmd == "continue" {
			return true
		}
		if cmd == "skip" {
			fmt.Printf("已跳过 %s。\n", target)
			return false
		}
		fmt.Print("无效输入。请输入 'continue' 或 'skip': ")
	}
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

func runTest(reader *bufio.Reader, chain, network, explorerBase string) {
	fmt.Printf("\n--- 正在启动 %s (%s) 测试 ---\n", chain, network)

	serverURL := "http://localhost:8080/execute"
	txData := "bW9ja2VkX3R4X2RhdGE="
	userSig := ""
	userAddr := "0xUSER_ADDR"

	if chain == "solana" {
		fmt.Println("[准备] 正在从 Devnet 获取最新 Blockhash 以构造真实交易...")
		c := rpc.New(rpc.DevNet_RPC)
		recent, err := c.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
		if err == nil {
			user := solana.NewWallet()
			target := solana.MustPublicKeyFromBase58("6R1A6p2P9ph7Y6T7m7h7n7b7v7n7b7v7n7b7v7n7b7v7")
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
		}
	} else if chain == "sui" {
		fmt.Println("[准备] 正在通过 buildSuiSponsoredTx 构造真实代付交易...")
		var err error
		txData, userSig, userAddr, err = buildSuiSponsoredTx()
		if err != nil {
			fmt.Printf("⚠️ 构造 Sui 交易失败: %v\n", err)
			return
		}
	}

	reqBody := common.ExecuteRequest{
		Chain:         chain,
		TxData:        txData,
		UserSignature: userSig,
		UserAddress:   userAddr,
		TargetAddress: "0xTARGET_ADDR",
	}
	jsonBody, _ := json.Marshal(reqBody)

	fmt.Println("[A] 发送无 Token 请求，验证 x402 拦截...")
	resp1, err := http.Post(serverURL, "application/json", bytes.NewBuffer(jsonBody))
	if err == nil {
		defer resp1.Body.Close()
		if resp1.StatusCode == http.StatusPaymentRequired {
			fmt.Printf("✅ 成功拦截 (402)! 需要支付至: %s\n", resp1.Header.Get("X-Payment-Address"))
		}
	}

	fmt.Print("\n[B] 模拟支付完成。按回车开始代付流程...")
	reader.ReadString('\n')

	httpClient := &http.Client{}
	req2, _ := http.NewRequest("POST", serverURL, bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Payment-Token", "paid-123")
	resp2, err := httpClient.Do(req2)
	if err == nil {
		defer resp2.Body.Close()
		var execResp common.ExecuteResponse
		json.NewDecoder(resp2.Body).Decode(&execResp)
		if execResp.Status == "success" {
			fmt.Printf("🚀 代付并执行成功!\n🔗 交易 Hash: %s\n🌐 浏览器查看: %s%s\n", execResp.TxHash, explorerBase, execResp.TxHash)
		} else {
			fmt.Printf("❌ 执行失败: %s\n", execResp.Error)
		}
	}
}

func buildSuiSponsoredTx() (txBase64 string, userSigBase64 string, userAddr string, err error) {
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		return "", "", "", err
	}

	// ========== 用户账户 ==========
	seed, _ := hex.DecodeString(common.SuiUserPK)
	scheme, _ := sui_types.NewSignatureScheme(0)
	userAcc := account.NewAccount(scheme, seed)
	userAddr = userAcc.Address

	// ========== sponsor 地址 ==========
	sponsorAddr, _ := sui_types.NewAddressFromHex(common.SuiSponsorAddr)

	// ========== 查询 gas price ==========
	gasPriceResp, err := cli.GetReferenceGasPrice(context.Background())
	if err != nil {
		return "", "", "", err
	}
	gasPrice := gasPriceResp.Uint64()

	// ========== sponsor gas coin ==========
	sponsorCoins, err := cli.GetCoins(
		context.Background(),
		*sponsorAddr,
		nil,
		nil,
		1,
	)
	if err != nil || len(sponsorCoins.Data) == 0 {
		return "", "", "", fmt.Errorf("sponsor 没有 gas coin")
	}

	// ========== 构造交易 ==========
	recipient, _ := sui_types.NewAddressFromHex(common.SuiSponsorAddr)
	amount := uint64(1000)

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

	// ========== BCS 编码 ==========
	bcsBytes, err := bcs.Marshal(txDataObj)
	if err != nil {
		return "", "", "", err
	}

	// ========== 用户签名 ==========
	sig, err := userAcc.SignSecureWithoutEncode(
		bcsBytes,
		sui_types.DefaultIntent(),
	)
	if err != nil {
		return "", "", "", err
	}

	txBase64 = base64.StdEncoding.EncodeToString(bcsBytes)
	// 在 v2.0.1 中，Signature 是一个包含指针的结构体
	if sig.Ed25519SuiSignature != nil {
		userSigBase64 = base64.StdEncoding.EncodeToString(sig.Ed25519SuiSignature.Signature[:])
	}

	return
}


package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"agent-wallet-gas-sponsor/common"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Sepolia USDC contract address (Circle USDC)
const (
	SepoliaUSDCContract = "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238" // Sepolia USDC
	MaxGasLimit         = uint64(100000)                                // 100k gas limit
)

// ERC20 ABI for USDC operations
const erc20ABI = `[
	{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "balance", "type": "uint256"}],
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_to", "type": "address"},
			{"name": "_value", "type": "uint256"}
		],
		"name": "transfer",
		"outputs": [{"name": "", "type": "bool"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "decimals",
		"outputs": [{"name": "", "type": "uint8"}],
		"type": "function"
	}
]`

func main() {
	fmt.Println("========================================")
	fmt.Println("   ETH USDC Gas代付完整测试")
	fmt.Println("========================================")

	fmt.Printf("Sponsor地址: %s\n", common.EVMSponsorAddr)
	fmt.Printf("用户地址: %s (有USDC，无ETH)\n", common.EVMUserAddr)
	fmt.Printf("网络: Ethereum Sepolia Testnet\n")
	fmt.Printf("浏览器: %s\n", common.EVMBrowser)
	fmt.Printf("测试场景: 用户转1 USDC，Sponsor代付ETH gas费\n\n")

	// === 步骤1: 检查 Sponsor ETH 余额 ===
	fmt.Println("=== 步骤1: 检查Sponsor ETH余额 ===")
	if err := checkSponsorBalance(); err != nil {
		fmt.Printf("❌ Sponsor余额检查失败: %v\n", err)
		fmt.Println("请访问 https://faucet.sepolia.dev/ 为Sponsor地址充值ETH")
		return
	}
	fmt.Println("✅ Sponsor ETH余额充足")

	// === 步骤2: 构造用户目标交易 (USDC 转账给别人) ===
	fmt.Println("\n=== 步骤2: 构造用户USDC转账交易 ===")
	fmt.Printf("💰 用户想转1 USDC给别人，但没有ETH支付gas\n")
	fmt.Printf("⚠️  注意: 请确保用户地址 %s 已有USDC余额\n", common.EVMUserAddr)

	// 目标接收者 (这里用 sponsor 地址演示)
	recipientAddr := common.EVMSponsorAddr
	transferAmount := big.NewInt(1000000) // 1 USDC (6 decimals)

	txData, err := buildUSDCTransferTx(recipientAddr, transferAmount)
	if err != nil {
		fmt.Printf("❌ 构造USDC转账交易失败: %v\n", err)
		fmt.Printf("\n💡 可能的解决方案:\n")
		fmt.Printf("   1. 确保用户地址 %s 有USDC余额\n", common.EVMUserAddr)
		fmt.Printf("   2. 查询账户: https://sepolia.etherscan.io/address/%s\n", common.EVMUserAddr)
		fmt.Printf("   3. 获取测试USDC: 使用Sepolia水龙头\n")
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
		fmt.Printf("   原生Gas: %.6f ETH\n", gasInfo.Payment.GasInfo.EstimatedGas)
		fmt.Printf("   ETH价格: $%.4f\n", gasInfo.Payment.GasInfo.TokenPriceUSD)
	}
	if addr, ok := gasInfo.Payment.Receivers["ethereum"]; ok {
		fmt.Printf("   ETH收款地址: %s\n", addr)
	}

	// === 步骤4: 用户支付 Gas 费用 (转USDC给我们) ===
	fmt.Println("\n=== 步骤4: 用户支付Gas费用 (USDC → Sponsor) ===")
	fmt.Printf("💰 用户需要支付: $%.6f USDC 作为gas费\n", gasInfo.Payment.PriceUSD)
	fmt.Println("💡 原理: 用户转USDC给我们，我们代付这笔转账的ETH gas")

	paymentAmount := big.NewInt(int64(gasInfo.Payment.PriceUSD * 1_000_000))
	if paymentAmount.Cmp(big.NewInt(10_000)) < 0 {
		paymentAmount = big.NewInt(10_000) // 最少 0.01 USDC
	}

	paymentTxData, err := buildUSDCTransferTx(common.EVMSponsorAddr, paymentAmount)
	if err != nil {
		fmt.Printf("❌ 构造支付交易失败: %v\n", err)
		return
	}

	fmt.Println("🔄 执行用户支付gas费的USDC转账 (bootstrap代付)...")
	paymentResult, err := executeTransaction(paymentTxData, "bootstrap")
	if err != nil {
		fmt.Printf("❌ 支付交易失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 用户支付完成!\n")
	fmt.Printf("   支付交易Hash: %s\n", paymentResult.TxHash)
	fmt.Printf("   浏览器查看: %s%s\n", common.EVMBrowser, paymentResult.TxHash)
	fmt.Printf("   我们代付了用户支付gas费的ETH gas\n")

	paymentProof := paymentResult.TxHash

	// === 步骤5: 用支付凭证执行原本的 USDC 转账 ===
	fmt.Println("\n=== 步骤5: 执行用户原本的USDC转账 ===")
	fmt.Println("🔄 等待支付交易链上确认后，执行原交易...")
	time.Sleep(5 * time.Second) // ETH 确认时间较长

	// 重新构造交易，获取最新的 nonce
	fmt.Println("   🔄 重新构造交易以获取最新的 nonce...")
	freshTxData, err := buildUSDCTransferTx(recipientAddr, transferAmount)
	if err != nil {
		fmt.Printf("❌ 重新构造交易失败: %v\n", err)
		return
	}

	result, err := executeTransaction(freshTxData, paymentProof)
	if err != nil {
		fmt.Printf("❌ 执行失败: %v\n", err)
		return
	}

	fmt.Printf("🚀 USDC转账交易执行成功!\n")
	fmt.Printf("   交易Hash: %s\n", result.TxHash)
	fmt.Printf("   浏览器查看: %s%s\n", common.EVMBrowser, result.TxHash)
	fmt.Printf("   ✅ 用户成功转账%.6f USDC，我们代付了ETH gas费\n", float64(transferAmount.Int64())/1e6)

	// === 步骤6: 等待确认 ===
	fmt.Println("\n=== 步骤6: 等待交易确认 ===")
	if err := waitForConfirmation(result.TxHash); err != nil {
		fmt.Printf("⚠️  交易确认超时: %v\n", err)
	} else {
		fmt.Println("✅ 交易已确认")
	}

	fmt.Println("\n========================================")
	fmt.Println("   ETH USDC代付Gas测试完成!")
	fmt.Printf("   🎯 用户转账: %.6f USDC\n", float64(transferAmount.Int64())/1e6)
	fmt.Printf("   💸 用户支付: $%.6f USDC (gas费)\n", gasInfo.Payment.PriceUSD)
	fmt.Printf("   ⛽ 我们代付: 2笔交易的ETH gas费\n")
	fmt.Printf("   📊 支付交易: %s%s\n", common.EVMBrowser, paymentResult.TxHash)
	fmt.Printf("   📊 转账交易: %s%s\n", common.EVMBrowser, result.TxHash)
	fmt.Println("   💡 完整流程: 用户转USDC给我们→我们代付→用户转USDC给别人→我们代付")
	fmt.Println("========================================")
}

// checkSponsorBalance 检查 Sponsor 账户的 ETH 余额是否充足
func checkSponsorBalance() error {
	client, err := ethclient.Dial(common.EVMSepoliaRPC)
	if err != nil {
		return fmt.Errorf("连接 Ethereum RPC 失败: %v", err)
	}
	defer client.Close()

	sponsorAddr := ethcommon.HexToAddress(common.EVMSponsorAddr)
	balance, err := client.BalanceAt(context.Background(), sponsorAddr, nil)
	if err != nil {
		return fmt.Errorf("获取余额失败: %v", err)
	}

	// Convert wei to ETH
	ethBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))
	minBalance := big.NewFloat(0.01) // 0.01 ETH

	if ethBalance.Cmp(minBalance) < 0 {
		return fmt.Errorf("余额不足: %.6f ETH (需要至少 0.01 ETH)", ethBalance)
	}

	fmt.Printf("   当前ETH余额: %.6f ETH\n", ethBalance)
	return nil
}

// buildUSDCTransferTx 构造 ETH 链上 USDC 转账交易（Sponsor 代付 gas）
func buildUSDCTransferTx(recipientHex string, amount *big.Int) (string, error) {
	client, err := ethclient.Dial(common.EVMSepoliaRPC)
	if err != nil {
		return "", fmt.Errorf("连接 Ethereum RPC 失败: %v", err)
	}
	defer client.Close()

	// 解析用户私钥
	userPK := strings.TrimPrefix(common.EVMUserPK, "0x")
	privateKey, err := crypto.HexToECDSA(userPK)
	if err != nil {
		return "", fmt.Errorf("解析用户私钥失败: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("获取公钥失败")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	
	// 检查用户 USDC 余额
	if err := checkUserUSDCBalance(client, fromAddress, amount); err != nil {
		return "", err
	}

	// 获取 nonce
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("获取 nonce 失败: %v", err)
	}

	// 获取 gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("获取 gas price 失败: %v", err)
	}

	// 构造 USDC 转账交易
	contractAddr := ethcommon.HexToAddress(SepoliaUSDCContract)
	recipientAddr := ethcommon.HexToAddress(recipientHex)

	// 解析 ERC20 ABI
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return "", fmt.Errorf("解析 ERC20 ABI 失败: %v", err)
	}

	// 编码 transfer 函数调用
	data, err := parsedABI.Pack("transfer", recipientAddr, amount)
	if err != nil {
		return "", fmt.Errorf("编码 transfer 调用失败: %v", err)
	}

	// 估算 gas limit
	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: fromAddress,
		To:   &contractAddr,
		Data: data,
	})
	if err != nil {
		gasLimit = MaxGasLimit // 使用默认值
	}

	// 创建交易
	tx := types.NewTransaction(nonce, contractAddr, big.NewInt(0), gasLimit, gasPrice, data)

	// 签名交易
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return "", fmt.Errorf("获取 chain ID 失败: %v", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %v", err)
	}

	// 编码交易为 hex
	txBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("编码交易失败: %v", err)
	}

	fmt.Printf("   用户USDC转账: %.6f USDC\n", float64(amount.Int64())/1e6)
	fmt.Printf("   接收者: %s\n", recipientHex)
	fmt.Printf("   Gas Limit: %d\n", gasLimit)
	fmt.Printf("   Gas Price: %s Gwei\n", new(big.Float).Quo(new(big.Float).SetInt(gasPrice), big.NewFloat(1e9)))

	return "0x" + hex.EncodeToString(txBytes), nil
}

// checkUserUSDCBalance 检查用户 USDC 余额
func checkUserUSDCBalance(client *ethclient.Client, userAddr ethcommon.Address, requiredAmount *big.Int) error {
	contractAddr := ethcommon.HexToAddress(SepoliaUSDCContract)
	
	// 解析 ERC20 ABI
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return fmt.Errorf("解析 ERC20 ABI 失败: %v", err)
	}

	// 调用 balanceOf
	data, err := parsedABI.Pack("balanceOf", userAddr)
	if err != nil {
		return fmt.Errorf("编码 balanceOf 调用失败: %v", err)
	}

	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}, nil)
	if err != nil {
		return fmt.Errorf("调用 balanceOf 失败: %v", err)
	}

	// 解析结果
	balance := new(big.Int).SetBytes(result)
	
	if balance.Cmp(requiredAmount) < 0 {
		return fmt.Errorf("用户 USDC 余额不足: 有 %.6f USDC, 需要 %.6f USDC",
			float64(balance.Int64())/1e6, float64(requiredAmount.Int64())/1e6)
	}

	fmt.Printf("   用户USDC余额: %.6f USDC\n", float64(balance.Int64())/1e6)
	return nil
}

// getGasEstimate 调用 x402 服务器获取 gas 费用估算
func getGasEstimate(txData string) (*common.X402Response, error) {
	reqBody := common.ExecuteRequest{
		Chain:         "evm",
		TxData:        txData,
		UserAddress:   common.EVMUserAddr,
		TargetAddress: common.EVMSponsorAddr,
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

// executeTransaction 通用交易执行函数，支持 bootstrap 和 ETH hash 支付凭证
func executeTransaction(txData, paymentProof string) (*common.ExecuteResponse, error) {
	reqBody := common.ExecuteRequest{
		Chain:         "evm",
		TxData:        txData,
		UserSignature: "signed_in_txdata", // ETH 交易已包含签名
		UserAddress:   common.EVMUserAddr,
		TargetAddress: common.EVMSponsorAddr,
	}

	jsonBody, _ := json.Marshal(reqBody)

	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8080/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-402-Payment", paymentProof)
	req.Header.Set("X-Payment-Chain", "evm")

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

// waitForConfirmation 等待 ETH 交易确认
func waitForConfirmation(txHashStr string) error {
	client, err := ethclient.Dial(common.EVMSepoliaRPC)
	if err != nil {
		return fmt.Errorf("连接 Ethereum RPC 失败: %v", err)
	}
	defer client.Close()

	txHash := ethcommon.HexToHash(txHashStr)

	fmt.Print("   等待确认")
	for i := 0; i < 30; i++ { // ETH 确认时间较长
		time.Sleep(3 * time.Second)
		fmt.Print(".")

		receipt, err := client.TransactionReceipt(context.Background(), txHash)
		if err == nil && receipt != nil {
			if receipt.Status == 1 {
				fmt.Println(" ✅")
				return nil
			}
			fmt.Println(" ❌")
			return fmt.Errorf("交易失败: status=%d", receipt.Status)
		}
	}

	fmt.Println(" ⏰")
	return fmt.Errorf("交易确认超时 (90秒)")
}
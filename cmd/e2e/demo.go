package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"agent-wallet-gas-sponsor/common"
)

// 演示完整的x402支付流程
func runDemo() {
	fmt.Println("========================================")
	fmt.Println("   x402 Gas 代付系统 - 快速演示")
	fmt.Println("========================================")

	// 演示Solana交易
	demoSolanaFlow()
	
	// 演示Ethereum交易
	demoEthereumFlow()
	
	// 演示Sui交易
	demoSuiFlow()
}

func demoSolanaFlow() {
	fmt.Println("\n--- Solana 演示流程 ---")
	
	// 1. 构造交易请求
	reqBody := common.ExecuteRequest{
		Chain:         "solana",
		TxData:        "base64_encoded_solana_tx_data",
		UserAddress:   common.SolanaUserAddr,
		TargetAddress: "11111111111111111111111111111112",
		UserSignature: "solana_user_signature",
	}
	
	// 2. 第一次请求 - 获取gas费用
	fmt.Println("[步骤1] 发送交易请求，获取gas费用估算...")
	gasInfo := getGasEstimateDemo("solana", reqBody)
	if gasInfo != nil {
		fmt.Printf("✅ 需要支付: $%.4f USDT\n", gasInfo.Payment.PriceUSD)
		fmt.Printf("   Gas详情: %s\n", gasInfo.Payment.Description)
		if gasInfo.Payment.GasInfo != nil {
			fmt.Printf("   目标链: %s, 预估gas: %.6f %s\n", 
				gasInfo.Payment.GasInfo.TargetChain,
				gasInfo.Payment.GasInfo.EstimatedGas,
				gasInfo.Payment.GasInfo.NativeToken)
		}
		fmt.Printf("   收款地址: %s\n", gasInfo.Payment.Receivers["solana"])
	}
	
	// 3. 模拟支付并执行
	fmt.Println("[步骤2] 模拟用户支付USDT并执行代付交易...")
	result := executeWithPaymentDemo("solana", reqBody, "demo-solana-payment")
	if result != nil {
		fmt.Printf("🚀 执行成功! 交易Hash: %s\n", result.TxHash)
	}
}

func demoEthereumFlow() {
	fmt.Println("\n--- Ethereum 演示流程 ---")
	
	reqBody := common.ExecuteRequest{
		Chain:         "ethereum",
		TxData:        "0x1234567890abcdef",
		UserAddress:   common.EVMUserAddr,
		TargetAddress: "0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e",
		UserSignature: "ethereum_user_signature",
	}
	
	fmt.Println("[步骤1] 发送交易请求，获取gas费用估算...")
	gasInfo := getGasEstimateDemo("ethereum", reqBody)
	if gasInfo != nil {
		fmt.Printf("✅ 需要支付: $%.4f USDT\n", gasInfo.Payment.PriceUSD)
		fmt.Printf("   Gas详情: %s\n", gasInfo.Payment.Description)
		fmt.Printf("   收款地址: %s\n", gasInfo.Payment.Receivers["ethereum"])
	}
	
	fmt.Println("[步骤2] 模拟用户支付USDT并执行代付交易...")
	result := executeWithPaymentDemo("ethereum", reqBody, "demo-ethereum-payment")
	if result != nil {
		fmt.Printf("🚀 执行成功! 交易Hash: %s\n", result.TxHash)
	}
}

func demoSuiFlow() {
	fmt.Println("\n--- Sui 演示流程 ---")
	
	reqBody := common.ExecuteRequest{
		Chain:         "sui",
		TxData:        "sui_transaction_data",
		UserAddress:   common.SuiUserAddr,
		TargetAddress: common.SuiSponsorAddr,
		UserSignature: "sui_user_signature",
	}
	
	fmt.Println("[步骤1] 发送交易请求，获取gas费用估算...")
	gasInfo := getGasEstimateDemo("sui", reqBody)
	if gasInfo != nil {
		fmt.Printf("✅ 需要支付: $%.4f USDT\n", gasInfo.Payment.PriceUSD)
		fmt.Printf("   Gas详情: %s\n", gasInfo.Payment.Description)
		fmt.Printf("   收款地址: %s\n", gasInfo.Payment.Receivers["solana"]) // 用户可选择任意网络支付
	}
	
	fmt.Println("[步骤2] 模拟用户支付USDT并执行代付交易...")
	result := executeWithPaymentDemo("sui", reqBody, "demo-sui-payment")
	if result != nil {
		fmt.Printf("🚀 执行成功! 交易Hash: %s\n", result.TxHash)
	}
}

func getGasEstimateDemo(chain string, reqBody common.ExecuteRequest) *common.X402Response {
	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post("http://localhost:8080/execute", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusPaymentRequired {
		fmt.Printf("❌ 期望402状态码，但收到: %d\n", resp.StatusCode)
		return nil
	}
	
	var x402Resp common.X402Response
	if err := json.NewDecoder(resp.Body).Decode(&x402Resp); err != nil {
		fmt.Printf("❌ 解析响应失败: %v\n", err)
		return nil
	}
	
	return &x402Resp
}

func executeWithPaymentDemo(chain string, reqBody common.ExecuteRequest, paymentProof string) *common.ExecuteResponse {
	jsonBody, _ := json.Marshal(reqBody)
	
	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8080/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-402-Payment", paymentProof)
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 执行请求失败: %v\n", err)
		return nil
	}
	defer resp.Body.Close()
	
	var execResp common.ExecuteResponse
	if err := json.NewDecoder(resp.Body).Decode(&execResp); err != nil {
		fmt.Printf("❌ 解析执行响应失败: %v\n", err)
		return nil
	}
	
	if execResp.Status != "success" {
		fmt.Printf("❌ 执行失败: %s\n", execResp.Error)
		return nil
	}
	
	return &execResp
}

// 如果想要运行演示，取消注释下面的main函数
/*
func main() {
	runDemo()
}
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	// 测试1: Solana交易gas估算
	fmt.Println("=== 测试1: Solana交易gas估算 ===")
	solanaRequest := map[string]interface{}{
		"chain":          "solana",
		"tx_data":        "base64_encoded_solana_transaction_data",
		"user_address":   "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
		"target_address": "11111111111111111111111111111112",
		"user_signature": "solana_signature_here",
	}
	
	testGasEstimate(solanaRequest)
	
	// 测试2: Ethereum交易gas估算
	fmt.Println("\n=== 测试2: Ethereum交易gas估算 ===")
	ethRequest := map[string]interface{}{
		"chain":          "ethereum",
		"tx_data":        "0x1234567890abcdef",
		"user_address":   "0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e",
		"target_address": "0xA0b86a33E6417c4c2f1C6C5b2c5c5c5c5c5c5c5c",
		"user_signature": "ethereum_signature_here",
	}
	
	testGasEstimate(ethRequest)
	
	// 测试3: Sui交易gas估算
	fmt.Println("\n=== 测试3: Sui交易gas估算 ===")
	suiRequest := map[string]interface{}{
		"chain":          "sui",
		"tx_data":        "sui_transaction_data",
		"user_address":   "0x5f31097cd2bd9957c0de799c088de591ece9747302a49e072528409014ed24dc",
		"target_address": "0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc",
		"user_signature": "sui_signature_here",
	}
	
	testGasEstimate(suiRequest)
	
	// 测试4: 实际的402流程
	fmt.Println("\n=== 测试4: 完整的402支付流程 ===")
	testFullPaymentFlow(solanaRequest)
}

func testGasEstimate(request map[string]interface{}) {
	jsonData, _ := json.Marshal(request)
	
	// 调用gas估算端点
	resp, err := http.Post("http://localhost:8080/estimate-gas", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("状态码: %d\n", resp.StatusCode)
	
	// 格式化JSON输出
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	prettyJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("响应: %s\n", string(prettyJSON))
}

func testFullPaymentFlow(request map[string]interface{}) {
	jsonData, _ := json.Marshal(request)
	
	// 第一次请求 - 应该返回402
	fmt.Println("第一次请求 (无支付凭证):")
	resp1, err := http.Post("http://localhost:8080/execute", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp1.Body.Close()
	
	body1, _ := io.ReadAll(resp1.Body)
	fmt.Printf("状态码: %d\n", resp1.StatusCode)
	
	var result1 map[string]interface{}
	json.Unmarshal(body1, &result1)
	prettyJSON1, _ := json.MarshalIndent(result1, "", "  ")
	fmt.Printf("402响应: %s\n\n", string(prettyJSON1))
	
	// 第二次请求 - 使用demo支付凭证
	fmt.Println("第二次请求 (demo支付凭证):")
	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8080/execute", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-402-Payment", "demo")
	
	resp2, err := client.Do(req)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp2.Body.Close()
	
	body2, _ := io.ReadAll(resp2.Body)
	fmt.Printf("状态码: %d\n", resp2.StatusCode)
	fmt.Printf("响应: %s\n", string(body2))
}
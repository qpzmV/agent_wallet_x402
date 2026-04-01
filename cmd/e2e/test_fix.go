package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"agent-wallet-gas-sponsor/common"
)

// 测试修复后的功能
func testFix() {
	fmt.Println("========================================")
	fmt.Println("   测试修复后的x402功能")
	fmt.Println("========================================")

	// 测试1: 检查402响应中的receivers字段
	fmt.Println("\n[测试1] 检查402响应中的receivers字段...")
	testReceiversField()

	// 测试2: 测试支付验证和执行
	fmt.Println("\n[测试2] 测试支付验证和执行...")
	testPaymentExecution()
}

func testReceiversField() {
	reqBody := common.ExecuteRequest{
		Chain:         "solana",
		TxData:        "test_tx_data",
		UserAddress:   common.SolanaUserAddr,
		TargetAddress: "test_target",
		UserSignature: "test_signature",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post("http://localhost:8080/execute", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		fmt.Printf("❌ 期望402状态码，但收到: %d\n", resp.StatusCode)
		return
	}

	var x402Resp common.X402Response
	if err := json.NewDecoder(resp.Body).Decode(&x402Resp); err != nil {
		fmt.Printf("❌ 解析响应失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 402响应解析成功\n")
	fmt.Printf("   支付金额: $%.4f USDT\n", x402Resp.Payment.PriceUSD)
	fmt.Printf("   描述: %s\n", x402Resp.Payment.Description)
	
	if x402Resp.Payment.Receivers != nil {
		fmt.Printf("   收款地址映射:\n")
		for network, addr := range x402Resp.Payment.Receivers {
			fmt.Printf("     %s: %s\n", network, addr)
		}
	} else {
		fmt.Printf("   ⚠️ Receivers字段为空\n")
	}
	
	if x402Resp.Payment.Receiver != "" {
		fmt.Printf("   默认收款地址: %s\n", x402Resp.Payment.Receiver)
	}
}

func testPaymentExecution() {
	reqBody := common.ExecuteRequest{
		Chain:         "solana",
		TxData:        "test_tx_data",
		UserAddress:   common.SolanaUserAddr,
		TargetAddress: "test_target",
		UserSignature: "test_signature",
	}

	jsonBody, _ := json.Marshal(reqBody)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8080/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 执行请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 应该返回402
	if resp.StatusCode == 402 {
		fmt.Printf("✅ 正确返回402支付要求\n")
	} else {
		fmt.Printf("❌ 期望402但得到: %d\n", resp.StatusCode)
	}

	// 读取原始响应
	respBody := make([]byte, 0)
	buffer := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			respBody = append(respBody, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}

	fmt.Printf("✅ 执行响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("   原始响应: %s\n", string(respBody))

	// 尝试解析响应
	var execResp common.ExecuteResponse
	if err := json.Unmarshal(respBody, &execResp); err != nil {
		fmt.Printf("   ⚠️ JSON解析失败: %v\n", err)
		
		// 尝试解析为通用响应
		var genericResp map[string]interface{}
		if jsonErr := json.Unmarshal(respBody, &genericResp); jsonErr == nil {
			fmt.Printf("   通用解析成功: %+v\n", genericResp)
		}
	} else {
		fmt.Printf("   ✅ ExecuteResponse解析成功\n")
		fmt.Printf("     状态: %s\n", execResp.Status)
		fmt.Printf("     交易Hash: %s\n", execResp.TxHash)
		if execResp.Error != "" {
			fmt.Printf("     错误: %s\n", execResp.Error)
		}
	}
}

// 如果想要运行测试，取消注释下面的main函数
/*
func main() {
	testFix()
}
*/
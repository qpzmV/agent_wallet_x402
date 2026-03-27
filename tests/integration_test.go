package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"agent-wallet-gas-sponsor/common"
)

func TestX402Flow(t *testing.T) {
	// 启动服务器通常需要 background 运行，这里我们假设已手动启动或在流水线中启动。
	// 为了演示，我们执行模拟 HTTP 请求。

	serverURL := "http://localhost:8080/execute"
	
	reqBody := common.ExecuteRequest{
		Chain:         "evm",
		TxData:        "0x...USER_SIGNED_TX...",
		UserAddress:   "0xUSER_ADDRESS",
		TargetAddress: "0xTARGET_ADDRESS",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// 1. 发送无 Token 请求 -> 期望 402
	t.Log("步骤 1: 发送无 Token 请求...")
	resp1, err := http.Post(serverURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusPaymentRequired {
		t.Errorf("期望 402 Payment Required，但得到 %d", resp1.StatusCode)
	} else {
		t.Log("成功得到 402 响应，包含支付地址: ", resp1.Header.Get("X-Payment-Address"))
	}

	// 2. 发送带有效 Token 的请求 -> 期望 200
	t.Log("步骤 2: 发送带有效支付 Token 的请求...")
	client := &http.Client{}
	req2, _ := http.NewRequest("POST", serverURL, bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Payment-Token", "paid-123") // 模拟已支付的凭证

	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("期望 200 OK，但得到 %d", resp2.StatusCode)
		respBody, _ := io.ReadAll(resp2.Body)
		t.Logf("错误响应: %s", string(respBody))
	} else {
		var execResp common.ExecuteResponse
		json.NewDecoder(resp2.Body).Decode(&execResp)
		t.Logf("交易执行成功! Hash: %s", execResp.TxHash)
	}
}

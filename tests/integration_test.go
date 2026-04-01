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

	// 2. 测试无支付凭证的情况
	t.Log("步骤 2: 测试无支付凭证...")
	client := &http.Client{}
	req2, _ := http.NewRequest("POST", serverURL, bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusPaymentRequired {
		t.Errorf("期望 402 Payment Required，但得到 %d", resp2.StatusCode)
		respBody, _ := io.ReadAll(resp2.Body)
		t.Logf("响应: %s", string(respBody))
	} else {
		t.Log("正确返回 402，需要支付")
	}
}

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"agent-wallet-gas-sponsor/common"
	"github.com/gin-gonic/gin"
)

const (
	ExecutionEngineURL = "http://localhost:8081/execute"
	SponsorUSDCAddress = "0x546A... (Sponsor USDC Receiver)"
)

func main() {
	r := gin.Default()

	r.POST("/execute", x402Middleware(), func(c *gin.Context) {
		// 如果通过了中间件，说明已支付或有凭证，转发给执行引擎
		proxyToExecutionEngine(c)
	})

	fmt.Println("x402 Middleware 正在运行在 :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func x402Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		paymentToken := c.GetHeader("X-Payment-Token")

		// 简单的模拟验证：如果 token 是 "paid-123"，则认为已支付
		if paymentToken == "paid-123" {
			c.Next()
			return
		}

		// 否则，计算费用并返回 402
		// 在实际场景中，我们会根据交易内容估算 Gas
		amount := 1.5 // 假设 1.5 USDC (含 Gas + 手续费)

		c.Header("X-Payment-Address", SponsorUSDCAddress)
		c.Header("X-Payment-Amount", fmt.Sprintf("%.2f", amount))
		c.Header("X-Payment-Currency", "USDC")

		c.JSON(http.StatusPaymentRequired, gin.H{
			"message": "需要支付以代付 Gas",
			"payment_info": common.PaymentRequest{
				Amount:   amount,
				Receiver: SponsorUSDCAddress,
				Currency: "USDC",
			},
		})
		c.Abort()
	}
}

func proxyToExecutionEngine(c *gin.Context) {
	// 读取原始请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取请求体"})
		return
	}

	// 转发请求
	resp, err := http.Post(ExecutionEngineURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "无法连接到执行引擎"})
		return
	}
	defer resp.Body.Close()

	// 将执行引擎的响应返回给用户
	respBody, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", respBody)
}

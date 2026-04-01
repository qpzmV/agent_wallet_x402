package main

import (
	"log"
	"net/http"

	"agent-wallet-gas-sponsor/common"
	"agent-wallet-gas-sponsor/execution-engine/sponsor"
	"github.com/gin-gonic/gin"
)

func main() {
	common.InitLogger("execution-engine")
	common.LogInfo("执行引擎正在启动...")

	r := gin.Default()

	r.POST("/execute", handleExecute)

	common.LogInfo("执行引擎正在运行在 :8081")
	if err := r.Run(":8081"); err != nil {
		common.LogError("启动服务失败: %v", err)
		log.Fatal(err)
	}
}

func handleExecute(c *gin.Context) {
	var req common.ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.LogError("请求格式错误: %v", err)
		c.JSON(http.StatusBadRequest, common.ExecuteResponse{Status: "failed", Error: "无效格式请求"})
		return
	}

	common.LogInfo("收到执行请求: chain=%s, user=%s, target=%s", req.Chain, req.UserAddress, req.TargetAddress)
	common.LogDebug("交易数据长度: %d bytes", len(req.TxData))

	var resp common.ExecuteResponse
	var err error

	switch req.Chain {
	case "evm":
		common.LogDebug("执行 EVM 交易")
		resp, err = sponsor.EVMExecute(req)
	case "solana":
		common.LogDebug("执行 Solana 交易")
		resp, err = sponsor.SolanaExecute(req)
	case "sui":
		common.LogDebug("执行 Sui 交易")
		resp, err = sponsor.SuiExecute(req)
	default:
		common.LogError("不支持的区块链: %s", req.Chain)
		c.JSON(http.StatusBadRequest, common.ExecuteResponse{Status: "failed", Error: "不支持的区块链: " + req.Chain})
		return
	}

	if err != nil {
		common.LogError("执行交易失败: %v", err)
		c.JSON(http.StatusInternalServerError, common.ExecuteResponse{Status: "failed", Error: err.Error()})
		return
	}

	common.LogInfo("交易执行成功: %s", resp.TxHash)
	c.JSON(http.StatusOK, resp)
}

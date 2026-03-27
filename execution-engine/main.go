package main

import (
	"log"
	"net/http"

	"agent-wallet-gas-sponsor/common"
	"agent-wallet-gas-sponsor/execution-engine/sponsor"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.POST("/execute", handleExecute)

	log.Println("执行引擎正在运行在 :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal(err)
	}
}

func handleExecute(c *gin.Context) {
	var req common.ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ExecuteResponse{Status: "failed", Error: "无效格式请求"})
		return
	}

	var resp common.ExecuteResponse
	var err error

	switch req.Chain {
	case "evm":
		resp, err = sponsor.EVMExecute(req)
	case "solana":
		resp, err = sponsor.SolanaExecute(req)
	case "sui":
		resp, err = sponsor.SuiExecute(req)
	default:
		c.JSON(http.StatusBadRequest, common.ExecuteResponse{Status: "failed", Error: "不支持的区块链: " + req.Chain})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ExecuteResponse{Status: "failed", Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

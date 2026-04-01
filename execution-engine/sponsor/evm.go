package sponsor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"agent-wallet-gas-sponsor/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func EVMExecute(req common.ExecuteRequest) (common.ExecuteResponse, error) {
	common.LogInfo("开始执行 EVM 交易: user=%s, target=%s", req.UserAddress, req.TargetAddress)
	common.LogDebug("连接 EVM RPC: %s", common.EVMSepoliaRPC)
	
	client, err := ethclient.Dial(common.EVMSepoliaRPC)
	if err != nil {
		common.LogError("无法连接到 Sepolia RPC: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("无法连接到 Sepolia: %v", err)
	}
	defer client.Close()

	common.LogDebug("解析交易数据，长度: %d bytes", len(req.TxData))
	
	// 1. 解析用户提交的交易数据 (假设是已签名的 hex)
	rawTx, err := hexutil.Decode(req.TxData)
	if err != nil {
		common.LogError("无效的交易数据格式: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("无效的交易数据: %v", err)
	}

	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(rawTx); err != nil {
		common.LogError("解析交易失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("解析交易失败: %v", err)
	}

	common.LogInfo("解析交易成功: hash=%s, to=%s, value=%s ETH", 
		tx.Hash().Hex(), tx.To().Hex(), formatWei(tx.Value()))

	// 2. 这里实现 Gas 代付的一种方式：由 Sponsor 提交交易并支付 Gas
	// 注意：在 EVM 中，只有 Meta-Transaction 或 Account Abstraction 才能让第三方付 Gas。
	// 这里演示由交易引擎直接广播已签名的交易，或者演示向用户地址转入 Gas 币。
	
	common.LogInfo("尝试广播交易到网络")
	
	// 演示：广播交易 (假设用户已经构造好了，只是我们需要协助提交)
	err = client.SendTransaction(context.Background(), tx)
	if err != nil {
		// 如果是因为没钱付 Gas 而失败，我们可以尝试给用户转一点 Gas
		common.LogWarn("广播失败，尝试为用户充值 Gas: %v", err)
		common.LogInfo("开始为用户 %s 转入 Gas 币", req.UserAddress)
		
		if err := transferGasETH(client, req.UserAddress); err != nil {
			common.LogError("Gas 充值失败: %v", err)
			return common.ExecuteResponse{}, fmt.Errorf("Gas 充值失败: %v", err)
		}
		
		// 等待gas转账完成
		common.LogInfo("等待 Gas 转账确认...")
		time.Sleep(10 * time.Second) // 等待10秒让转账被挖矿
		
		// 再次尝试
		common.LogInfo("重新尝试广播交易")
		err = client.SendTransaction(context.Background(), tx)
		if err != nil {
			common.LogError("再次广播失败: %v", err)
			return common.ExecuteResponse{}, fmt.Errorf("再次广播失败: %v", err)
		}
	}

	common.LogInfo("EVM 交易广播成功: %s", tx.Hash().Hex())

	return common.ExecuteResponse{
		TxHash: tx.Hash().Hex(),
		Status: "success",
	}, nil
}

func transferGasETH(client *ethclient.Client, toAddress string) error {
	common.LogInfo("开始 Gas ETH 转账: to=%s", toAddress)
	
	// 实现简单的转账逻辑，从 Sponsor 账户给用户转 0.01 ETH
	privateKey, err := crypto.ToECDSA(hexutil.MustDecode(common.EVMSponsorPK))
	if err != nil {
		common.LogError("解析 Sponsor 私钥失败: %v", err)
		return err
	}
	
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKey)
	
	common.LogDebug("Sponsor 地址: %s", fromAddress.Hex())

	ctx := context.Background()
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		common.LogError("获取 nonce 失败: %v", err)
		return err
	}
	
	value := big.NewInt(10000000000000000) // 0.01 ETH
	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		common.LogError("获取 gas price 失败: %v", err)
		return err
	}

	common.LogInfo("构造 Gas 转账交易: nonce=%d, value=0.01 ETH, gasPrice=%s gwei", 
		nonce, formatGwei(gasPrice))

	tx := types.NewTransaction(nonce, ethcommon.HexToAddress(toAddress), value, gasLimit, gasPrice, nil)
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		common.LogError("获取 chain ID 失败: %v", err)
		return err
	}
	
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		common.LogError("签名交易失败: %v", err)
		return err
	}

	common.LogDebug("发送 Gas 转账交易: %s", signedTx.Hash().Hex())
	
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		common.LogError("发送 Gas 转账失败: %v", err)
		return err
	}

	common.LogInfo("Gas 转账交易已发送: %s", signedTx.Hash().Hex())
	return nil
}

// 辅助函数：格式化 Wei 为 ETH
func formatWei(wei *big.Int) string {
	eth := new(big.Float).SetInt(wei)
	eth.Quo(eth, big.NewFloat(1e18))
	return eth.Text('f', 6)
}

// 辅助函数：格式化 Wei 为 Gwei
func formatGwei(wei *big.Int) string {
	gwei := new(big.Float).SetInt(wei)
	gwei.Quo(gwei, big.NewFloat(1e9))
	return gwei.Text('f', 2)
}

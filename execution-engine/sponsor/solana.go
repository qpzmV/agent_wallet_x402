package sponsor

import (
	"context"
	"fmt"

	"agent-wallet-gas-sponsor/common"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func SolanaExecute(req common.ExecuteRequest) (common.ExecuteResponse, error) {
	common.LogInfo("开始执行 Solana 交易: user=%s, target=%s", req.UserAddress, req.TargetAddress)
	common.LogDebug("连接 Solana RPC: %s", common.SolanaDevnetRPC)
	
	client := rpc.New(common.SolanaDevnetRPC)

	common.LogDebug("解析 Solana 交易数据，长度: %d bytes", len(req.TxData))
	
	// 1. 反序列化
	tx, err := solana.TransactionFromBase64(req.TxData)
	if err != nil {
		common.LogError("无效的 Solana 交易数据: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("无效的 Solana 交易数据: %v", err)
	}

	common.LogInfo("解析交易成功: 账户数=%d, 指令数=%d", 
		len(tx.Message.AccountKeys), len(tx.Message.Instructions))

	// 2. 准备 Sponsor (Fee Payer)
	sponsorKey, err := solana.PrivateKeyFromBase58(common.SolanaSponsorPK)
	if err != nil {
		common.LogError("解析 Sponsor 私钥失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("解析 Sponsor 私钥失败: %v", err)
	}
	
	common.LogDebug("Sponsor 地址: %s", sponsorKey.PublicKey().String())

	// 【关键检查】：确保交易的 FeePayer 确实是我们的 Sponsor 地址
	// 如果客户端没设对，这里必须修正，否则钱扣不到 Sponsor 账上
	if !tx.Message.AccountKeys[0].Equals(sponsorKey.PublicKey()) {
		common.LogError("交易 FeePayer 不匹配: 期望=%s, 实际=%s", 
			sponsorKey.PublicKey().String(), tx.Message.AccountKeys[0].String())
		return common.ExecuteResponse{}, fmt.Errorf("交易 FeePayer 不匹配，期望: %s", sponsorKey.PublicKey())
	}

	common.LogInfo("FeePayer 验证通过: %s", sponsorKey.PublicKey().String())

	// 3. 重新计算并签名
	common.LogDebug("开始 Sponsor 签名")
	data, err := tx.Message.MarshalBinary()
	if err != nil {
		common.LogError("序列化消息失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("序列化消息失败: %v", err)
	}
	
	sig, err := sponsorKey.Sign(data)
	if err != nil {
		common.LogError("Sponsor 签名失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("Sponsor 签名失败: %v", err)
	}

	// 填充签名到正确位置（通常 Payer 是第 0 个）
	found := false
	for i, key := range tx.Message.AccountKeys {
		if i >= int(tx.Message.Header.NumRequiredSignatures) {
			break
		}
		if key.Equals(sponsorKey.PublicKey()) {
			tx.Signatures[i] = sig
			found = true
			common.LogDebug("在位置 %d 填充 Sponsor 签名", i)
		}
	}

	if !found {
		common.LogError("在交易账户列表中未找到 Sponsor 账户")
		return common.ExecuteResponse{}, fmt.Errorf("在交易账户列表中未找到 Sponsor 账户")
	}

	// 4. 发送前预检 (避免 0x1 错误再次发生)
	common.LogDebug("检查 Sponsor 账户余额")
	balance, err := client.GetBalance(context.Background(), sponsorKey.PublicKey(), rpc.CommitmentProcessed)
	if err != nil {
		common.LogWarn("无法获取 Sponsor 余额: %v", err)
	} else {
		balanceSOL := float64(balance.Value) / 1e9
		common.LogInfo("Sponsor 账户余额: %.9f SOL", balanceSOL)
		
		if balance.Value < 10000 { // 假设至少需要 0.00001 SOL
			common.LogError("Sponsor 账户余额不足: %.9f SOL", balanceSOL)
			return common.ExecuteResponse{}, fmt.Errorf("Sponsor 账户余额不足，请充值 Devnet SOL")
		}
	}

	// 5. 广播
	common.LogInfo("广播 Solana 交易到网络")
	sigVal, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		common.LogError("Solana 交易广播失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("广播失败: %v", err)
	}

	common.LogInfo("Solana 交易广播成功: %s", sigVal.String())

	return common.ExecuteResponse{
		TxHash: sigVal.String(),
		Status: "success",
	}, nil
}

package sponsor

import (
	"context"
	"fmt"

	"agent-wallet-gas-sponsor/common"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func SolanaExecute(req common.ExecuteRequest) (common.ExecuteResponse, error) {
	client := rpc.New(common.SolanaDevnetRPC)

	// 1. 反序列化
	tx, err := solana.TransactionFromBase64(req.TxData)
	if err != nil {
		return common.ExecuteResponse{}, fmt.Errorf("无效的 Solana 交易数据: %v", err)
	}

	// 2. 准备 Sponsor (Fee Payer)
	sponsorKey, _ := solana.PrivateKeyFromBase58(common.SolanaSponsorPK)

	// 【关键检查】：确保交易的 FeePayer 确实是我们的 Sponsor 地址
	// 如果客户端没设对，这里必须修正，否则钱扣不到 Sponsor 账上
	if !tx.Message.AccountKeys[0].Equals(sponsorKey.PublicKey()) {
		return common.ExecuteResponse{}, fmt.Errorf("交易 FeePayer 不匹配，期望: %s", sponsorKey.PublicKey())
	}

	// 3. 重新计算并签名
	data, err := tx.Message.MarshalBinary()
	if err != nil {
		return common.ExecuteResponse{}, fmt.Errorf("序列化消息失败: %v", err)
	}
	sig, err := sponsorKey.Sign(data)
	if err != nil {
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
		}
	}

	if !found {
		return common.ExecuteResponse{}, fmt.Errorf("在交易账户列表中未找到 Sponsor 账户")
	}

	// 4. 发送前预检 (避免 0x1 错误再次发生)
	balance, err := client.GetBalance(context.Background(), sponsorKey.PublicKey(), rpc.CommitmentProcessed)
	if err == nil && balance.Value < 10000 { // 假设至少需要 0.00001 SOL
		return common.ExecuteResponse{}, fmt.Errorf("Sponsor 账户余额不足，请充值 Devnet SOL")
	}

	// 5. 广播
	sigVal, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		return common.ExecuteResponse{}, fmt.Errorf("广播失败: %v", err)
	}

	return common.ExecuteResponse{
		TxHash: sigVal.String(),
		Status: "success",
	}, nil
}

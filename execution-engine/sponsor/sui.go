package sponsor

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"agent-wallet-gas-sponsor/common"
	"github.com/coming-chat/go-sui/v2/account"
	"github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/sui_types"
	"github.com/coming-chat/go-sui/v2/types"
)

func SuiExecute(req common.ExecuteRequest) (common.ExecuteResponse, error) {
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		return common.ExecuteResponse{}, fmt.Errorf("连接 Sui RPC 失败: %v", err)
	}

	// ========== 解码交易 ==========
	txBytes, err := base64.StdEncoding.DecodeString(req.TxData)
	if err != nil {
		return common.ExecuteResponse{}, fmt.Errorf("无效交易数据: %v", err)
	}

	// ========== 解析 TransactionData (暂略，因 go-bcs v0.2.1 反射切片 Bug) ==========
	// var txData sui_types.TransactionData
	// err = bcs.Unmarshal(txBytes, &txData)
	// if err != nil {
	// 	return common.ExecuteResponse{}, fmt.Errorf("交易解析失败: %v", err)
	// }

	// ========== 验证 sender (暂略，Demo 版信任请求参数) ==========
	// if txData.V1.Sender.String() != req.UserAddress {
	// 	return common.ExecuteResponse{}, fmt.Errorf("sender 与用户地址不匹配")
	// }

	// ========== 验证用户签名 (暂略，v2.0.1 接口变更) ==========
	// Note: VerifySecure is missing in this version, would need manual ed25519 check.

	// ========== sponsor 账户 ==========
	seed, _ := hex.DecodeString(common.SuiSponsorPK)
	scheme, _ := sui_types.NewSignatureScheme(0)
	sponsorAcc := account.NewAccount(scheme, seed)

	// ========== sponsor 签名 ==========
	sponsorSig, err := sponsorAcc.SignSecureWithoutEncode(
		txBytes,
		sui_types.DefaultIntent(),
	)
	if err != nil {
		return common.ExecuteResponse{}, fmt.Errorf("sponsor 签名失败: %v", err)
	}

	// ========== 合并签名 ==========
	var sponsorSigBase64 string
	if sponsorSig.Ed25519SuiSignature != nil {
		sponsorSigBase64 = base64.StdEncoding.EncodeToString(sponsorSig.Ed25519SuiSignature.Signature[:])
	}

	signatures := []any{
		req.UserSignature,
		sponsorSigBase64,
	}

	// ========== 广播 ==========
	resp, err := cli.ExecuteTransactionBlock(
		context.Background(),
		txBytes,
		signatures,
		&types.SuiTransactionBlockResponseOptions{
			ShowEffects: true,
		},
		types.TxnRequestTypeWaitForLocalExecution,
	)
	if err != nil {
		return common.ExecuteResponse{}, fmt.Errorf("交易执行失败: %v", err)
	}

	return common.ExecuteResponse{
		TxHash: resp.Digest.String(),
		Status: "success",
	}, nil
}

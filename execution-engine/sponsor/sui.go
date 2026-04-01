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
	common.LogInfo("开始执行 Sui 交易: user=%s, target=%s", req.UserAddress, req.TargetAddress)
	common.LogDebug("连接 Sui RPC: %s", common.SuiTestnetRPC)
	
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		common.LogError("连接 Sui RPC 失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("连接 Sui RPC 失败: %v", err)
	}

	common.LogDebug("解析 Sui 交易数据，长度: %d bytes", len(req.TxData))

	// ========== 解码交易 ==========
	txBytes, err := base64.StdEncoding.DecodeString(req.TxData)
	if err != nil {
		common.LogError("无效交易数据格式: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("无效交易数据: %v", err)
	}

	common.LogInfo("交易数据解码成功，二进制长度: %d bytes", len(txBytes))

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

	common.LogDebug("跳过交易数据详细解析 (SDK 限制)")

	// ========== sponsor 账户 ==========
	common.LogDebug("初始化 Sponsor 账户")
	seed, err := hex.DecodeString(common.SuiSponsorPK)
	if err != nil {
		common.LogError("解析 Sponsor 私钥失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("解析 Sponsor 私钥失败: %v", err)
	}
	
	scheme, err := sui_types.NewSignatureScheme(0)
	if err != nil {
		common.LogError("创建签名方案失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("创建签名方案失败: %v", err)
	}
	
	sponsorAcc := account.NewAccount(scheme, seed)
	common.LogInfo("Sponsor 账户地址: %s", sponsorAcc.Address)

	// ========== sponsor 签名 ==========
	common.LogDebug("开始 Sponsor 签名")
	sponsorSig, err := sponsorAcc.SignSecureWithoutEncode(
		txBytes,
		sui_types.DefaultIntent(),
	)
	if err != nil {
		common.LogError("Sponsor 签名失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("sponsor 签名失败: %v", err)
	}

	common.LogInfo("Sponsor 签名完成")

	// ========== 合并签名 ==========
	var sponsorSigBase64 string
	if sponsorSig.Ed25519SuiSignature != nil {
		sponsorSigBase64 = base64.StdEncoding.EncodeToString(sponsorSig.Ed25519SuiSignature.Signature[:])
		common.LogDebug("Sponsor 签名编码完成，长度: %d", len(sponsorSigBase64))
	} else {
		common.LogWarn("Sponsor 签名格式异常")
	}

	signatures := []any{
		req.UserSignature,
		sponsorSigBase64,
	}

	common.LogInfo("合并签名完成: 用户签名 + Sponsor 签名")
	common.LogDebug("用户签名长度: %d, Sponsor 签名长度: %d", 
		len(req.UserSignature), len(sponsorSigBase64))

	// ========== 广播 ==========
	common.LogInfo("广播 Sui 交易到网络")
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
		common.LogError("Sui 交易执行失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("交易执行失败: %v", err)
	}

	// 检查交易执行结果
	if resp.Effects != nil && resp.Effects.Data.V1.Status.Status != "success" {
		common.LogError("Sui 交易执行状态异常: %s", resp.Effects.Data.V1.Status.Status)
		if resp.Effects.Data.V1.Status.Error != "" {
			common.LogError("错误详情: %s", resp.Effects.Data.V1.Status.Error)
		}
		return common.ExecuteResponse{}, fmt.Errorf("交易执行失败: %s", resp.Effects.Data.V1.Status.Status)
	}

	common.LogInfo("Sui 交易执行成功: %s", resp.Digest.String())

	return common.ExecuteResponse{
		TxHash: resp.Digest.String(),
		Status: "success",
	}, nil
}

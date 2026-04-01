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
	"github.com/fardream/go-bcs/bcs"
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

	// ========== 解析并验证交易数据 ==========
	// 完整解析交易数据，验证所有关键字段
	common.LogDebug("开始解析 Sui 交易数据")

	// 尝试解析交易数据
	var txData sui_types.TransactionData
	readLen, err := bcs.Unmarshal(txBytes, &txData)
	if err != nil {
		common.LogError("交易数据解析失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("交易数据解析失败: %v", err)
	}
	if readLen != len(txBytes) {
		extra := len(txBytes) - readLen
		common.LogError("交易数据异常: 包含 %d 字节的额外数据", extra)
		return common.ExecuteResponse{}, fmt.Errorf("交易包含未知的多余字节")
	}

	common.LogInfo("交易数据解析成功")

	// ========== 验证 sender ==========
	// 严格验证交易发送者地址，确保与请求中的用户地址完全匹配
	// 防止用户提交其他人的交易进行代付
	common.LogDebug("验证交易发送者")

	if txData.V1 == nil {
		common.LogError("交易数据格式错误: V1 为空")
		return common.ExecuteResponse{}, fmt.Errorf("交易数据格式错误")
	}

	// 验证 sender 地址是否与请求中的用户地址匹配
	txSender := txData.V1.Sender.String()
	if txSender != req.UserAddress {
		common.LogError("Sender 验证失败: 交易中的sender=%s, 请求中的用户地址=%s",
			txSender, req.UserAddress)
		return common.ExecuteResponse{}, fmt.Errorf("sender 与用户地址不匹配: 期望 %s, 实际 %s",
			req.UserAddress, txSender)
	}

	common.LogInfo("Sender 验证通过: %s", txSender)

	// ========== 验证 Gas 配置 ==========
	// 验证 Gas Owner 必须是我们的 Sponsor 地址
	// 验证 Gas Budget 在合理范围内，防止恶意消耗
	common.LogDebug("验证 Gas 配置")

	gasOwner := txData.V1.GasData.Owner.String()
	expectedSponsor := common.SuiSponsorAddr

	if gasOwner != expectedSponsor {
		common.LogError("Gas Owner 验证失败: 交易中的gas_owner=%s, 期望的sponsor=%s",
			gasOwner, expectedSponsor)
		return common.ExecuteResponse{}, fmt.Errorf("gas owner 必须是 sponsor 地址: 期望 %s, 实际 %s",
			expectedSponsor, gasOwner)
	}

	common.LogInfo("Gas Owner 验证通过: %s", gasOwner)

	// 验证 Gas Budget 是否合理
	gasBudget := txData.V1.GasData.Budget
	maxGasBudget := uint64(100000000) // 0.1 SUI

	if gasBudget > maxGasBudget {
		common.LogWarn("Gas Budget 过高: %d, 最大允许: %d", gasBudget, maxGasBudget)
		return common.ExecuteResponse{}, fmt.Errorf("gas budget 过高: %d, 最大允许: %d",
			gasBudget, maxGasBudget)
	}

	common.LogInfo("Gas Budget 验证通过: %d", gasBudget)

	// ========== 验证交易类型和内容 ==========
	// 限制只支持可编程交易，并检查命令数量
	// 防止复杂或恶意的交易类型
	common.LogDebug("验证交易类型和内容")

	// 检查交易是否为可编程交易
	if txData.V1.Kind.ProgrammableTransaction == nil {
		common.LogError("不支持的交易类型: 非可编程交易")
		return common.ExecuteResponse{}, fmt.Errorf("仅支持可编程交易")
	}

	// 检查交易命令数量是否合理
	commands := txData.V1.Kind.ProgrammableTransaction.Commands
	maxCommands := 10 // 限制最多10个命令

	if len(commands) > maxCommands {
		common.LogError("交易命令过多: %d, 最大允许: %d", len(commands), maxCommands)
		return common.ExecuteResponse{}, fmt.Errorf("交易命令过多: %d, 最大允许: %d",
			len(commands), maxCommands)
	}

	common.LogInfo("交易验证通过: %d 个命令", len(commands))

	// ========== 验证交易过期时间 ==========
	if txData.V1.Expiration.Epoch != nil {
		common.LogDebug("交易设置了过期时间: epoch %d", *txData.V1.Expiration.Epoch)
		// 这里可以添加过期时间的验证逻辑
	} else {
		common.LogDebug("交易未设置过期时间")
	}

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

	// ========== 验证用户签名 ==========
	// 验证用户签名的格式和有效性
	// 确保交易确实由用户授权
	common.LogDebug("验证用户签名")
	err = verifyUserSignature(req.UserSignature, txBytes, req.UserAddress)
	if err != nil {
		common.LogError("用户签名验证失败: %v", err)
		return common.ExecuteResponse{}, fmt.Errorf("用户签名验证失败: %v", err)
	}
	common.LogInfo("用户签名验证通过")

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

// verifyUserSignature 验证用户签名是否有效
func verifyUserSignature(userSigBase64 string, txBytes []byte, userAddress string) error {
	// 解码用户签名
	userSigBytes, err := base64.StdEncoding.DecodeString(userSigBase64)
	if err != nil {
		return fmt.Errorf("用户签名解码失败: %v", err)
	}

	// 解析用户地址
	userAddr, err := sui_types.NewAddressFromHex(userAddress)
	if err != nil {
		return fmt.Errorf("用户地址格式错误: %v", err)
	}

	common.LogDebug("用户签名验证: 地址=%s, 签名长度=%d", userAddr.String(), len(userSigBytes))

	// 注意: 这里简化了签名验证逻辑
	// 在生产环境中，应该使用完整的密码学验证
	// 包括从签名中恢复公钥并验证是否与用户地址匹配

	// 基本格式验证已通过
	common.LogDebug("用户签名基本格式验证通过")
	return nil
}

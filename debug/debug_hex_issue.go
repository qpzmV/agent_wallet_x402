package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"agent-wallet-gas-sponsor/common"

	"github.com/coming-chat/go-sui/v2/account"
	"github.com/coming-chat/go-sui/v2/sui_types"
)

func main() {
	fmt.Println("🔍 调试 Hex 编码问题")
	fmt.Println("========================================")

	// 测试用户私钥解码
	fmt.Printf("用户私钥: %s\n", common.SuiUserPK)
	fmt.Printf("私钥长度: %d 字符\n", len(common.SuiUserPK))
	
	seed, err := hex.DecodeString(common.SuiUserPK)
	if err != nil {
		fmt.Printf("❌ 解析用户私钥失败: %v\n", err)
		fmt.Printf("   私钥内容: %q\n", common.SuiUserPK)
		
		// 检查每个字符
		for i, char := range common.SuiUserPK {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				fmt.Printf("   无效字符在位置 %d: %c (U+%04X)\n", i, char, char)
			}
		}
		return
	}
	fmt.Printf("✅ 用户私钥解码成功，长度: %d 字节\n", len(seed))

	// 测试创建账户
	fmt.Println("\n=== 测试创建用户账户 ===")
	scheme, err := sui_types.NewSignatureScheme(0) // Ed25519
	if err != nil {
		log.Fatalf("❌ 创建签名方案失败: %v", err)
	}
	
	userAcc := account.NewAccount(scheme, seed)
	fmt.Printf("✅ 用户账户创建成功\n")
	fmt.Printf("   地址: %s\n", userAcc.Address)
	fmt.Printf("   期望地址: %s\n", common.SuiUserAddr)
	
	if userAcc.Address != common.SuiUserAddr {
		fmt.Printf("⚠️  地址不匹配!\n")
	}

	// 测试 Sponsor 私钥
	fmt.Printf("\nSponsor 私钥: %s\n", common.SuiSponsorPK)
	fmt.Printf("Sponsor 私钥长度: %d 字符\n", len(common.SuiSponsorPK))
	
	sponsorSeed, err := hex.DecodeString(common.SuiSponsorPK)
	if err != nil {
		fmt.Printf("❌ 解析 Sponsor 私钥失败: %v\n", err)
		return
	}
	fmt.Printf("✅ Sponsor 私钥解码成功，长度: %d 字节\n", len(sponsorSeed))

	sponsorAcc := account.NewAccount(scheme, sponsorSeed)
	fmt.Printf("✅ Sponsor 账户创建成功\n")
	fmt.Printf("   地址: %s\n", sponsorAcc.Address)
	fmt.Printf("   期望地址: %s\n", common.SuiSponsorAddr)
	
	if sponsorAcc.Address != common.SuiSponsorAddr {
		fmt.Printf("⚠️  Sponsor 地址不匹配!\n")
	}

	fmt.Println("\n========================================")
	fmt.Println("🔍 Hex 调试完成")
}
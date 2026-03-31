package main

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	"agent-wallet-gas-sponsor/common"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	fmt.Println("🔍 验证 ETH 私钥和地址匹配")
	fmt.Println("========================================")

	// 验证 Sponsor 账户
	fmt.Println("=== Sponsor 账户验证 ===")
	fmt.Printf("配置的地址: %s\n", common.EVMSponsorAddr)
	fmt.Printf("配置的私钥: %s\n", common.EVMSponsorPK)
	
	sponsorMatch, sponsorDerived := verifyKeyPair(common.EVMSponsorPK, common.EVMSponsorAddr)
	if sponsorMatch {
		fmt.Printf("✅ Sponsor 私钥和地址匹配\n")
	} else {
		fmt.Printf("❌ Sponsor 私钥和地址不匹配\n")
		fmt.Printf("   私钥派生的地址: %s\n", sponsorDerived)
		fmt.Printf("   配置的地址:     %s\n", common.EVMSponsorAddr)
	}

	fmt.Println()

	// 验证用户账户
	fmt.Println("=== 用户账户验证 ===")
	fmt.Printf("配置的地址: %s\n", common.EVMUserAddr)
	fmt.Printf("配置的私钥: %s\n", common.EVMUserPK)
	
	userMatch, userDerived := verifyKeyPair(common.EVMUserPK, common.EVMUserAddr)
	if userMatch {
		fmt.Printf("✅ 用户私钥和地址匹配\n")
	} else {
		fmt.Printf("❌ 用户私钥和地址不匹配\n")
		fmt.Printf("   私钥派生的地址: %s\n", userDerived)
		fmt.Printf("   配置的地址:     %s\n", common.EVMUserAddr)
	}

	fmt.Println()

	// 总结
	fmt.Println("=== 验证总结 ===")
	if sponsorMatch && userMatch {
		fmt.Println("🎉 所有私钥和地址都匹配，可以进行充值")
		fmt.Println()
		fmt.Println("💰 充值指南:")
		fmt.Printf("   1. Sponsor ETH 充值地址: %s\n", common.EVMSponsorAddr)
		fmt.Printf("      获取 ETH: https://faucet.sepolia.dev/\n")
		fmt.Printf("   2. 用户 USDC 充值地址: %s\n", common.EVMUserAddr)
		fmt.Printf("      获取 USDC: 需要通过 DEX 或其他方式获取 Sepolia USDC\n")
		fmt.Printf("      USDC 合约: 0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238\n")
	} else {
		fmt.Println("❌ 存在不匹配的私钥和地址，需要修复配置")
		if !sponsorMatch {
			fmt.Printf("   建议: 将 EVMSponsorAddr 改为 %s\n", sponsorDerived)
		}
		if !userMatch {
			fmt.Printf("   建议: 将 EVMUserAddr 改为 %s\n", userDerived)
		}
	}

	fmt.Println("\n========================================")
}

// verifyKeyPair 验证私钥和地址是否匹配
func verifyKeyPair(privateKeyHex, expectedAddr string) (bool, string) {
	// 移除 0x 前缀
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	
	// 解析私钥
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		fmt.Printf("❌ 解析私钥失败: %v\n", err)
		return false, ""
	}

	// 获取公钥
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		fmt.Printf("❌ 获取公钥失败\n")
		return false, ""
	}

	// 派生地址
	derivedAddr := crypto.PubkeyToAddress(*publicKeyECDSA)
	derivedAddrStr := derivedAddr.Hex()

	// 比较地址 (忽略大小写)
	expectedAddrLower := strings.ToLower(expectedAddr)
	derivedAddrLower := strings.ToLower(derivedAddrStr)

	return expectedAddrLower == derivedAddrLower, derivedAddrStr
}
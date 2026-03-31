package main

import (
	"fmt"
	"agent-wallet-gas-sponsor/common"
	"github.com/gagliardetto/solana-go"
)

func main() {
	fmt.Println("=== 验证Solana密钥对 ===")
	
	// 验证Sponsor密钥对
	fmt.Println("\n1. 验证Sponsor密钥对:")
	fmt.Printf("   配置的地址: %s\n", common.SolanaSponsorAddr)
	fmt.Printf("   配置的私钥: %s\n", common.SolanaSponsorPK)
	
	sponsorKey, err := solana.PrivateKeyFromBase58(common.SolanaSponsorPK)
	if err != nil {
		fmt.Printf("   ❌ Sponsor私钥格式错误: %v\n", err)
	} else {
		derivedAddr := sponsorKey.PublicKey().String()
		fmt.Printf("   从私钥推导的地址: %s\n", derivedAddr)
		if derivedAddr == common.SolanaSponsorAddr {
			fmt.Printf("   ✅ Sponsor密钥对匹配\n")
		} else {
			fmt.Printf("   ❌ Sponsor密钥对不匹配\n")
		}
	}
	
	// 验证User密钥对
	fmt.Println("\n2. 验证User密钥对:")
	fmt.Printf("   配置的地址: %s\n", common.SolanaUserAddr)
	fmt.Printf("   配置的私钥: %s\n", common.SolanaUserPK)
	
	userKey, err := solana.PrivateKeyFromBase58(common.SolanaUserPK)
	if err != nil {
		fmt.Printf("   ❌ User私钥格式错误: %v\n", err)
	} else {
		derivedAddr := userKey.PublicKey().String()
		fmt.Printf("   从私钥推导的地址: %s\n", derivedAddr)
		if derivedAddr == common.SolanaUserAddr {
			fmt.Printf("   ✅ User密钥对匹配\n")
		} else {
			fmt.Printf("   ❌ User密钥对不匹配\n")
		}
	}
	
	// 生成一个新的正确的密钥对
	fmt.Println("\n3. 生成新的正确密钥对:")
	newWallet := solana.NewWallet()
	fmt.Printf("   新地址: %s\n", newWallet.PublicKey().String())
	fmt.Printf("   新私钥: %s\n", newWallet.PrivateKey.String())
	fmt.Printf("   私钥长度: %d 字符\n", len(newWallet.PrivateKey.String()))
	
	// 验证新生成的密钥对
	testKey, err := solana.PrivateKeyFromBase58(newWallet.PrivateKey.String())
	if err != nil {
		fmt.Printf("   ❌ 新私钥格式错误: %v\n", err)
	} else {
		testAddr := testKey.PublicKey().String()
		if testAddr == newWallet.PublicKey().String() {
			fmt.Printf("   ✅ 新密钥对验证成功\n")
		} else {
			fmt.Printf("   ❌ 新密钥对验证失败\n")
		}
	}
}
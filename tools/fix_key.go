package main

import (
	"fmt"
	"github.com/gagliardetto/solana-go"
)

func main() {
	// 生成一个有效的Solana密钥对用于测试
	wallet := solana.NewWallet()
	
	fmt.Printf("// 有效的Solana测试密钥对\n")
	fmt.Printf("SolanaUserAddr = \"%s\"\n", wallet.PublicKey().String())
	fmt.Printf("SolanaUserPK   = \"%s\"\n", wallet.PrivateKey.String())
	
	// 验证私钥格式
	_, err := solana.PrivateKeyFromBase58(wallet.PrivateKey.String())
	if err != nil {
		fmt.Printf("❌ 私钥格式错误: %v\n", err)
	} else {
		fmt.Printf("✅ 私钥格式正确\n")
	}
}
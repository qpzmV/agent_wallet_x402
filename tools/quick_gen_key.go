package main

import (
	"fmt"
	"github.com/gagliardetto/solana-go"
)

func main() {
	// 生成新的测试密钥对
	wallet := solana.NewWallet()
	
	fmt.Println("=== 新的Solana测试密钥对 ===")
	fmt.Printf("地址: %s\n", wallet.PublicKey().String())
	fmt.Printf("私钥: %s\n", wallet.PrivateKey.String())
	fmt.Println("")
	fmt.Println("请将你的20 USDC转到新地址，然后更新config.go")
}
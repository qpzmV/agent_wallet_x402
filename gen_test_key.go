package main

import (
	"fmt"
	"github.com/gagliardetto/solana-go"
)

func main() {
	// 生成新的Solana测试密钥对
	wallet := solana.NewWallet()
	
	fmt.Printf("Solana测试用户地址: %s\n", wallet.PublicKey().String())
	fmt.Printf("Solana测试用户私钥: %s\n", wallet.PrivateKey.String())
}
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
)

func main() {
	fmt.Println("正在生成固定测试私钥...")

	// 1. EVM (Sepolia)
	evmPriv, _ := crypto.GenerateKey()
	evmAddr := crypto.PubkeyToAddress(evmPriv.PublicKey).Hex()
	evmPK := hexutil.Encode(crypto.FromECDSA(evmPriv))
	fmt.Printf("EVM Address: %s\n", evmAddr)
	fmt.Printf("EVM Private Key: %s\n", evmPK)

	// 2. Solana (Devnet)
	solanaKey := solana.NewWallet()
	solanaAddr := solanaKey.PublicKey().String()
	solanaPK := solanaKey.PrivateKey.String()
	fmt.Printf("Solana Address: %s\n", solanaAddr)
	fmt.Printf("Solana Private Key: %s\n", solanaPK)

	// 3. Sui (Testnet)
	// Sui 使用 ed25519, 32字节 secret
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	suiPK := hex.EncodeToString(priv.Seed())
	suiAddr := hex.EncodeToString(pub) // 简化模型
	fmt.Printf("Sui Address (Seed): %s\n", suiAddr)
	fmt.Printf("Sui Private Key (Seed Hex): %s\n", suiPK)

	fmt.Println("\n请手动将以上信息更新到 common/config.go 中。")
}

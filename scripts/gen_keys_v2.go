package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/blake2b"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
)

func main() {
	fmt.Println("正在重新生成准确的固定测试私钥...")

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

	// 3. Sui (Testnet) - 正确派生
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	
	// Sui Address = Blake2b256(flag || pubkey)
	// Flag for Ed25519 is 0x00
	tmp := append([]byte{0x00}, pub...)
	hash := blake2b.Sum256(tmp)
	suiAddr := "0x" + hex.EncodeToString(hash[:])
	
	suiPK := hex.EncodeToString(priv.Seed())
	
	fmt.Printf("Sui Address: %s\n", suiAddr)
	fmt.Printf("Sui Private Key (Seed Hex): %s\n", suiPK)

	fmt.Println("\n请手动将以上信息更新到 common/config.go 中。")
}

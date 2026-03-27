package main

import (
	"fmt"
	"github.com/gagliardetto/solana-go"
)

func main() {
	pkStr := "3V2PTXSb2sMR6uyKZZHaouDV8AWQfsgaqRfpgK6NAGBDM8r4oU3kdHAfSFwR948FGtvJaX94d1NLcfMmyatbKdMq"
	key, err := solana.PrivateKeyFromBase58(pkStr)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Private Key: %s\n", pkStr)
	fmt.Printf("Derived Public Key: %s\n", key.PublicKey().String())
}

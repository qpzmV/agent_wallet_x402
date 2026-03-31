package main

import (
	"context"
	"fmt"
	"agent-wallet-gas-sponsor/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func main() {
	fmt.Println("=== 检查USDC合约地址 ===")
	
	client := rpc.New(rpc.DevNet_RPC)
	userPubkey := solana.MustPublicKeyFromBase58(common.SolanaUserAddr)
	sponsorPubkey := solana.MustPublicKeyFromBase58(common.SolanaSponsorAddr)
	
	fmt.Printf("用户地址: %s\n", common.SolanaUserAddr)
	fmt.Printf("Sponsor地址: %s\n", common.SolanaSponsorAddr)
	fmt.Printf("当前配置的USDC合约: %s\n", common.SolanaUSDTContract)
	
	// 检查用户的USDC Token账户
	usdcMint := solana.MustPublicKeyFromBase58(common.SolanaUSDTContract)
	userTokenAccount, _, err := solana.FindAssociatedTokenAddress(userPubkey, usdcMint)
	if err != nil {
		fmt.Printf("❌ 计算用户Token账户地址失败: %v\n", err)
		return
	}
	
	fmt.Printf("\n=== 用户USDC Token账户 ===\n")
	fmt.Printf("计算的Token账户: %s\n", userTokenAccount.String())
	
	// 检查用户账户是否存在
	userAccountInfo, err := client.GetAccountInfo(context.Background(), userTokenAccount)
	if err != nil {
		fmt.Printf("❌ 用户Token账户不存在: %v\n", err)
	} else if userAccountInfo.Value == nil {
		fmt.Printf("❌ 用户Token账户不存在\n")
	} else {
		fmt.Printf("✅ 用户Token账户存在\n")
	}
	
	// 检查Sponsor的USDC Token账户
	sponsorTokenAccount, _, err := solana.FindAssociatedTokenAddress(sponsorPubkey, usdcMint)
	if err != nil {
		fmt.Printf("❌ 计算Sponsor Token账户地址失败: %v\n", err)
		return
	}
	
	fmt.Printf("\n=== Sponsor USDC Token账户 ===\n")
	fmt.Printf("计算的Token账户: %s\n", sponsorTokenAccount.String())
	
	// 检查Sponsor账户是否存在
	sponsorAccountInfo, err := client.GetAccountInfo(context.Background(), sponsorTokenAccount)
	if err != nil {
		fmt.Printf("❌ Sponsor Token账户不存在: %v\n", err)
	} else if sponsorAccountInfo.Value == nil {
		fmt.Printf("❌ Sponsor Token账户不存在 - 需要创建!\n")
		fmt.Printf("💡 解决方案: 先给Sponsor地址转一点USDC来创建Token账户\n")
	} else {
		fmt.Printf("✅ Sponsor Token账户存在\n")
	}
}
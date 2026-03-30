package main

import (
	"context"
	"fmt"
	"log"

	"agent-wallet-gas-sponsor/common"

	"github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/sui_types"
)

func main() {
	fmt.Println("🔍 调试简单余额查询")
	fmt.Println("========================================")

	// 连接 RPC
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		log.Fatalf("❌ 连接 Sui RPC 失败: %v", err)
	}
	fmt.Println("✅ RPC 连接成功")

	// 解析地址
	sponsorAddr, err := sui_types.NewAddressFromHex(common.SuiSponsorAddr)
	if err != nil {
		log.Fatalf("❌ 解析 Sponsor 地址失败: %v", err)
	}

	// 使用 GetBalance 而不是 GetCoins
	fmt.Println("\n=== 使用 GetBalance 查询 ===")
	suiCoinType := "0x2::sui::SUI"
	balance, err := cli.GetBalance(context.Background(), *sponsorAddr, suiCoinType)
	if err != nil {
		fmt.Printf("❌ 获取余额失败: %v\n", err)
		fmt.Printf("   错误类型: %T\n", err)
		return
	}
	
	suiBalance := balance.TotalBalance.BigInt().Uint64()
	fmt.Printf("✅ Sponsor SUI 余额: %d MIST = %.6f SUI\n", suiBalance, float64(suiBalance)/1e9)

	// 尝试获取所有余额
	fmt.Println("\n=== 获取所有余额 ===")
	allBalances, err := cli.GetAllBalances(context.Background(), *sponsorAddr)
	if err != nil {
		fmt.Printf("❌ 获取所有余额失败: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Sponsor 有 %d 种币\n", len(allBalances))
	for i, bal := range allBalances {
		amount := bal.TotalBalance.BigInt().Uint64()
		fmt.Printf("   币种 %d: %s (余额: %d)\n", i+1, bal.CoinType, amount)
	}

	fmt.Println("\n========================================")
	fmt.Println("🔍 简单余额查询完成")
}
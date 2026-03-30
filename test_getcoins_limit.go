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
	fmt.Println("🔍 测试 GetCoins 的 limit 参数影响")
	fmt.Println("========================================")

	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		log.Fatalf("❌ 连接 Sui RPC 失败: %v", err)
	}
	fmt.Println("✅ RPC 连接成功")

	sponsorAddr, err := sui_types.NewAddressFromHex(common.SuiSponsorAddr)
	if err != nil {
		log.Fatalf("❌ 解析 Sponsor 地址失败: %v", err)
	}

	userAddr, err := sui_types.NewAddressFromHex(common.SuiUserAddr)
	if err != nil {
		log.Fatalf("❌ 解析用户地址失败: %v", err)
	}

	// 测试不同的 limit 值对 Sponsor SUI coins 的影响
	fmt.Println("\n=== 测试 Sponsor SUI coins ===")
	limits := []uint{1, 2, 5, 10, 20}
	
	for _, limit := range limits {
		fmt.Printf("测试 limit=%d: ", limit)
		coins, err := cli.GetCoins(context.Background(), *sponsorAddr, nil, nil, limit)
		if err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
		} else {
			fmt.Printf("✅ 成功，获得 %d 个 coins\n", len(coins.Data))
		}
	}

	// 测试不同的 limit 值对用户 USDC coins 的影响
	fmt.Println("\n=== 测试用户 USDC coins ===")
	usdcType := "0xa1ec7fc00a6f40db9693ad1415d0c193ad3906494428cf252621037bd7117e29::usdc::USDC"
	
	for _, limit := range limits {
		fmt.Printf("测试 limit=%d: ", limit)
		coins, err := cli.GetCoins(context.Background(), *userAddr, &usdcType, nil, limit)
		if err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
		} else {
			fmt.Printf("✅ 成功，获得 %d 个 coins\n", len(coins.Data))
		}
	}

	// 测试用户 SUI coins
	fmt.Println("\n=== 测试用户 SUI coins ===")
	
	for _, limit := range limits {
		fmt.Printf("测试 limit=%d: ", limit)
		coins, err := cli.GetCoins(context.Background(), *userAddr, nil, nil, limit)
		if err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
		} else {
			fmt.Printf("✅ 成功，获得 %d 个 coins\n", len(coins.Data))
		}
	}

	// 测试特殊情况：limit=0 和 limit=nil
	fmt.Println("\n=== 测试特殊 limit 值 ===")
	
	fmt.Printf("测试 limit=0: ")
	coins, err := cli.GetCoins(context.Background(), *sponsorAddr, nil, nil, 0)
	if err != nil {
		fmt.Printf("❌ 失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功，获得 %d 个 coins\n", len(coins.Data))
	}

	fmt.Println("\n========================================")
	fmt.Println("🔍 limit 参数测试完成")
}
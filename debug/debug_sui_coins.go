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
	fmt.Println("🔍 调试 SUI Coins 查询问题")
	fmt.Println("========================================")

	// 连接到 SUI RPC
	fmt.Printf("连接到 SUI RPC: %s\n", common.SuiTestnetRPC)
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		log.Fatalf("❌ 连接 Sui RPC 失败: %v", err)
	}
	fmt.Println("✅ RPC 连接成功")

	// 解析 Sponsor 地址
	fmt.Printf("解析 Sponsor 地址: %s\n", common.SuiSponsorAddr)
	sponsorAddr, err := sui_types.NewAddressFromHex(common.SuiSponsorAddr)
	if err != nil {
		log.Fatalf("❌ 解析 Sponsor 地址失败: %v", err)
	}
	fmt.Printf("✅ Sponsor 地址解析成功: %s\n", sponsorAddr.String())

	// 解析用户地址
	fmt.Printf("解析用户地址: %s\n", common.SuiUserAddr)
	userAddr, err := sui_types.NewAddressFromHex(common.SuiUserAddr)
	if err != nil {
		log.Fatalf("❌ 解析用户地址失败: %v", err)
	}
	fmt.Printf("✅ 用户地址解析成功: %s\n", userAddr.String())

	// 测试获取 Sponsor 的 SUI coins
	fmt.Println("\n=== 测试 Sponsor SUI Coins ===")
	sponsorCoins, err := cli.GetCoins(context.Background(), *sponsorAddr, nil, nil, 5)
	if err != nil {
		fmt.Printf("❌ 获取 Sponsor SUI coins 失败: %v\n", err)
		fmt.Printf("   地址: %s\n", sponsorAddr.String())
		fmt.Printf("   错误详情: %T - %v\n", err, err)
	} else {
		fmt.Printf("✅ 获取 Sponsor SUI coins 成功，数量: %d\n", len(sponsorCoins.Data))
		for i, coin := range sponsorCoins.Data {
			fmt.Printf("   Coin %d: %s (余额: %d)\n", i+1, coin.CoinObjectId.String(), coin.Balance.Uint64())
		}
	}

	// 测试获取用户的 SUI coins
	fmt.Println("\n=== 测试用户 SUI Coins ===")
	userCoins, err := cli.GetCoins(context.Background(), *userAddr, nil, nil, 5)
	if err != nil {
		fmt.Printf("❌ 获取用户 SUI coins 失败: %v\n", err)
	} else {
		fmt.Printf("✅ 获取用户 SUI coins 成功，数量: %d\n", len(userCoins.Data))
		for i, coin := range userCoins.Data {
			fmt.Printf("   Coin %d: %s (余额: %d)\n", i+1, coin.CoinObjectId.String(), coin.Balance.Uint64())
		}
	}

	// 测试获取用户的 USDC coins
	fmt.Println("\n=== 测试用户 USDC Coins ===")
	usdcType := "0xa1ec7fc00a6f40db9d37cd5e4b452b3c7b61f87c56e6a82b6ec23a0fdf29c0e4::usdc::USDC"
	fmt.Printf("USDC 类型: %s\n", usdcType)
	
	userUSDCCoins, err := cli.GetCoins(context.Background(), *userAddr, &usdcType, nil, 10)
	if err != nil {
		fmt.Printf("❌ 获取用户 USDC coins 失败: %v\n", err)
		fmt.Printf("   地址: %s\n", userAddr.String())
		fmt.Printf("   币种类型: %s\n", usdcType)
		fmt.Printf("   错误详情: %T - %v\n", err, err)
	} else {
		fmt.Printf("✅ 获取用户 USDC coins 成功，数量: %d\n", len(userUSDCCoins.Data))
		totalUSDC := uint64(0)
		for i, coin := range userUSDCCoins.Data {
			balance := coin.Balance.Uint64()
			totalUSDC += balance
			fmt.Printf("   USDC Coin %d: %s (余额: %d = %.6f USDC)\n", 
				i+1, coin.CoinObjectId.String(), balance, float64(balance)/1e6)
		}
		fmt.Printf("   总 USDC 余额: %.6f USDC\n", float64(totalUSDC)/1e6)
	}

	// 测试获取 Sponsor 余额
	fmt.Println("\n=== 测试 Sponsor SUI 余额 ===")
	suiCoinType := "0x2::sui::SUI"
	balance, err := cli.GetBalance(context.Background(), *sponsorAddr, suiCoinType)
	if err != nil {
		fmt.Printf("❌ 获取 Sponsor SUI 余额失败: %v\n", err)
	} else {
		suiBalance := balance.TotalBalance.BigInt().Uint64()
		fmt.Printf("✅ Sponsor SUI 余额: %d MIST = %.6f SUI\n", suiBalance, float64(suiBalance)/1e9)
	}

	fmt.Println("\n========================================")
	fmt.Println("🔍 调试完成")
}
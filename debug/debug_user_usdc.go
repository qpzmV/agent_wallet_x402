package main

import (
	"context"
	"fmt"
	"log"

	"agent-wallet-gas-sponsor/common"

	"github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/sui_types"
)

const (
	SuiUSDCType = "0xa1ec7fc00a6f40db9693ad1415d0c193ad3906494428cf252621037bd7117e29::usdc::USDC"
)

func main() {
	fmt.Println("🔍 调试用户 USDC Coins 获取问题")
	fmt.Println("========================================")

	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		log.Fatalf("❌ 连接 Sui RPC 失败: %v", err)
	}
	fmt.Println("✅ RPC 连接成功")

	userAddr, err := sui_types.NewAddressFromHex(common.SuiUserAddr)
	if err != nil {
		log.Fatalf("❌ 解析用户地址失败: %v", err)
	}
	fmt.Printf("✅ 用户地址: %s\n", userAddr.String())

	// 测试1: 获取用户所有余额
	fmt.Println("\n=== 测试1: 获取用户所有余额 ===")
	allBalances, err := cli.GetAllBalances(context.Background(), *userAddr)
	if err != nil {
		fmt.Printf("❌ GetAllBalances 失败: %v\n", err)
	} else {
		fmt.Printf("✅ 用户有 %d 种币\n", len(allBalances))
		for i, bal := range allBalances {
			amount := bal.TotalBalance.BigInt().Uint64()
			fmt.Printf("   币种 %d: %s (余额: %d)\n", i+1, bal.CoinType, amount)
			
			if bal.CoinType == SuiUSDCType {
				fmt.Printf("     ⭐ 这是 USDC! 余额: %.6f USDC\n", float64(amount)/1e6)
			}
		}
	}

	// 测试2: 尝试获取 USDC coins (预期失败)
	fmt.Println("\n=== 测试2: 获取 USDC coins (预期失败) ===")
	fmt.Printf("查询币种: %s\n", SuiUSDCType)
	
	coinType := SuiUSDCType
	userUSDCCoins, err := cli.GetCoins(context.Background(), *userAddr, &coinType, nil, 10)
	if err != nil {
		fmt.Printf("❌ GetCoins 失败: %v\n", err)
		fmt.Printf("   错误类型: %T\n", err)
		
		// 分析错误
		if hexErr, ok := err.(interface{ Error() string }); ok {
			errMsg := hexErr.Error()
			if len(errMsg) > 50 {
				fmt.Printf("   错误详情: %s...\n", errMsg[:50])
			} else {
				fmt.Printf("   错误详情: %s\n", errMsg)
			}
		}
	} else {
		fmt.Printf("✅ GetCoins 成功，获得 %d 个 USDC coins\n", len(userUSDCCoins.Data))
		for i, coin := range userUSDCCoins.Data {
			balance := coin.Balance.Uint64()
			fmt.Printf("   USDC Coin %d: %s (余额: %.6f USDC)\n", 
				i+1, coin.CoinObjectId.String(), float64(balance)/1e6)
		}
	}

	// 测试3: 尝试获取用户的 SUI coins
	fmt.Println("\n=== 测试3: 获取用户 SUI coins ===")
	userSUICoins, err := cli.GetCoins(context.Background(), *userAddr, nil, nil, 5)
	if err != nil {
		fmt.Printf("❌ 获取用户 SUI coins 失败: %v\n", err)
	} else {
		fmt.Printf("✅ 获取用户 SUI coins 成功，数量: %d\n", len(userSUICoins.Data))
	}

	// 测试4: 尝试不同的 USDC 币种格式
	fmt.Println("\n=== 测试4: 尝试不同的币种格式 ===")
	
	alternativeTypes := []string{
		"0xa1ec7fc00a6f40db9d37cd5e4b452b3c7b61f87c56e6a82b6ec23a0fdf29c0e4::usdc::USDC",
		"0x2::coin::Coin<0xa1ec7fc00a6f40db9d37cd5e4b452b3c7b61f87c56e6a82b6ec23a0fdf29c0e4::usdc::USDC>",
	}
	
	for i, altType := range alternativeTypes {
		fmt.Printf("   尝试格式 %d: %s\n", i+1, altType)
		coins, err := cli.GetCoins(context.Background(), *userAddr, &altType, nil, 5)
		if err != nil {
			fmt.Printf("     ❌ 失败: %v\n", err)
		} else {
			fmt.Printf("     ✅ 成功，找到 %d 个 coins\n", len(coins.Data))
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("🔍 用户 USDC 调试完成")
}
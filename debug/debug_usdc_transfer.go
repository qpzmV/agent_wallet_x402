package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"

	"agent-wallet-gas-sponsor/common"

	"github.com/coming-chat/go-sui/v2/account"
	"github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/sui_types"
)

const (
	SuiUSDCType    = "0xa1ec7fc00a6f40db9d37cd5e4b452b3c7b61f87c56e6a82b6ec23a0fdf29c0e4::usdc::USDC"
	SuiCoinType    = "0x2::sui::SUI"
)

func main() {
	fmt.Println("🔍 调试 USDC 转账构造问题")
	fmt.Println("========================================")

	// 连接 RPC
	cli, err := client.Dial(common.SuiTestnetRPC)
	if err != nil {
		log.Fatalf("❌ 连接 Sui RPC 失败: %v", err)
	}
	fmt.Println("✅ RPC 连接成功")

	// 创建用户账户
	seed, err := hex.DecodeString(common.SuiUserPK)
	if err != nil {
		log.Fatalf("❌ 解析用户私钥失败: %v", err)
	}
	
	scheme, _ := sui_types.NewSignatureScheme(0) // Ed25519
	userAcc := account.NewAccount(scheme, seed)
	fmt.Printf("✅ 用户账户: %s\n", userAcc.Address)

	userAddr, err := sui_types.NewAddressFromHex(userAcc.Address)
	if err != nil {
		log.Fatalf("❌ 解析用户地址失败: %v", err)
	}

	sponsorAddr, err := sui_types.NewAddressFromHex(common.SuiSponsorAddr)
	if err != nil {
		log.Fatalf("❌ 解析 Sponsor 地址失败: %v", err)
	}

	// 步骤1: 获取 Sponsor 的 gas coin
	fmt.Println("\n=== 步骤1: 获取 Sponsor SUI coins ===")
	sponsorCoins, err := cli.GetCoins(context.Background(), *sponsorAddr, nil, nil, 1)
	if err != nil {
		fmt.Printf("❌ 获取 Sponsor SUI coins 失败: %v\n", err)
		fmt.Printf("   错误类型: %T\n", err)
		fmt.Printf("   错误详情: %+v\n", err)
		return
	}
	
	if len(sponsorCoins.Data) == 0 {
		fmt.Printf("❌ Sponsor 没有 SUI coins\n")
		return
	}
	
	fmt.Printf("✅ Sponsor 有 %d 个 SUI coins\n", len(sponsorCoins.Data))
	for i, coin := range sponsorCoins.Data {
		fmt.Printf("   Coin %d: %s (余额: %d MIST)\n", i+1, coin.CoinObjectId.String(), coin.Balance.Uint64())
	}

	// 步骤2: 获取用户的 USDC coins
	fmt.Println("\n=== 步骤2: 获取用户 USDC coins ===")
	fmt.Printf("查询地址: %s\n", userAddr.String())
	fmt.Printf("币种类型: %s\n", SuiUSDCType)
	
	coinType := SuiUSDCType
	userUSDCCoins, err := cli.GetCoins(context.Background(), *userAddr, &coinType, nil, 10)
	if err != nil {
		fmt.Printf("❌ 获取用户 USDC coins 失败: %v\n", err)
		fmt.Printf("   错误类型: %T\n", err)
		fmt.Printf("   错误详情: %+v\n", err)
		
		// 尝试不指定币种类型
		fmt.Println("\n--- 尝试获取所有币种 ---")
		allCoins, err2 := cli.GetCoins(context.Background(), *userAddr, nil, nil, 10)
		if err2 != nil {
			fmt.Printf("❌ 获取所有币种也失败: %v\n", err2)
		} else {
			fmt.Printf("✅ 用户有 %d 个 coins (所有类型)\n", len(allCoins.Data))
			for i, coin := range allCoins.Data {
				fmt.Printf("   Coin %d: %s (类型: %s, 余额: %d)\n", 
					i+1, coin.CoinObjectId.String(), coin.CoinType, coin.Balance.Uint64())
			}
		}
		return
	}
	
	fmt.Printf("✅ 用户有 %d 个 USDC coins\n", len(userUSDCCoins.Data))
	totalUSDC := uint64(0)
	for i, coin := range userUSDCCoins.Data {
		balance := coin.Balance.Uint64()
		totalUSDC += balance
		fmt.Printf("   USDC Coin %d: %s (余额: %d = %.6f USDC)\n", 
			i+1, coin.CoinObjectId.String(), balance, float64(balance)/1e6)
	}
	
	if totalUSDC == 0 {
		fmt.Printf("⚠️  用户没有 USDC 余额\n")
		fmt.Printf("   请访问 SUI 测试网水龙头获取 USDC\n")
		fmt.Printf("   或者修改代码使用 SUI 代替 USDC 进行演示\n")
		return
	}

	fmt.Printf("✅ 用户 USDC 总余额: %.6f USDC\n", float64(totalUSDC)/1e6)

	fmt.Println("\n========================================")
	fmt.Println("🔍 USDC 转账调试完成")
}
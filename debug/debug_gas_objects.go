package main

import (
	"context"
	"fmt"
	"log"

	"agent-wallet-gas-sponsor/common"

	"github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/sui_types"
	"github.com/coming-chat/go-sui/v2/types"
)

func main() {
	fmt.Println("🔍 调试 Gas Objects 获取问题")
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

	// 方法1: 直接调用 GetCoins (会失败)
	fmt.Println("\n=== 方法1: GetCoins (预期失败) ===")
	coins, err := cli.GetCoins(context.Background(), *sponsorAddr, nil, nil, 5)
	if err != nil {
		fmt.Printf("❌ GetCoins 失败: %v\n", err)
		fmt.Printf("   错误类型: %T\n", err)
	} else {
		fmt.Printf("✅ GetCoins 成功，获得 %d 个 coins\n", len(coins.Data))
	}

	// 方法2: 使用 GetOwnedObjects 获取所有对象
	fmt.Println("\n=== 方法2: GetOwnedObjects ===")
	
	// 构造查询 - 查找所有 Coin<SUI> 对象
	query := types.SuiObjectResponseQuery{
		Filter: &types.SuiObjectDataFilter{
			StructType: "0x2::coin::Coin<0x2::sui::SUI>",
		},
		Options: &types.SuiObjectDataOptions{
			ShowType:                true,
			ShowContent:             true,
			ShowBcs:                 false,
			ShowOwner:               true,
			ShowPreviousTransaction: false,
			ShowStorageRebate:       false,
			ShowDisplay:             false,
		},
	}

	objects, err := cli.GetOwnedObjects(context.Background(), *sponsorAddr, &query, nil, nil)
	if err != nil {
		fmt.Printf("❌ GetOwnedObjects 失败: %v\n", err)
		fmt.Printf("   错误类型: %T\n", err)
	} else {
		fmt.Printf("✅ GetOwnedObjects 成功，找到 %d 个对象\n", len(objects.Data))
		
		for i, obj := range objects.Data {
			if obj.Data != nil {
				fmt.Printf("   对象 %d:\n", i+1)
				fmt.Printf("     ObjectId: %s\n", obj.Data.ObjectId.String())
				fmt.Printf("     Version: %d\n", obj.Data.Version.Uint64())
				fmt.Printf("     Type: %s\n", obj.Data.Type)
				
				// 尝试提取余额信息
				if obj.Data.Content != nil {
					fmt.Printf("     Content: %+v\n", obj.Data.Content)
				}
			}
		}
	}

	// 方法3: 使用不同的查询方式
	fmt.Println("\n=== 方法3: 简化查询 ===")
	
	simpleQuery := types.SuiObjectResponseQuery{
		Filter: &types.SuiObjectDataFilter{
			StructType: "0x2::coin::Coin",
		},
		Options: &types.SuiObjectDataOptions{
			ShowType:    true,
			ShowContent: true,
			ShowOwner:   true,
		},
	}

	simpleObjects, err := cli.GetOwnedObjects(context.Background(), *sponsorAddr, &simpleQuery, nil, nil)
	if err != nil {
		fmt.Printf("❌ 简化查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 简化查询成功，找到 %d 个 coin 对象\n", len(simpleObjects.Data))
	}

	// 方法4: 不使用过滤器，获取所有对象
	fmt.Println("\n=== 方法4: 获取所有对象 ===")
	
	allQuery := types.SuiObjectResponseQuery{
		Options: &types.SuiObjectDataOptions{
			ShowType:    true,
			ShowContent: false,
			ShowOwner:   true,
		},
	}

	allObjects, err := cli.GetOwnedObjects(context.Background(), *sponsorAddr, &allQuery, nil, nil)
	if err != nil {
		fmt.Printf("❌ 获取所有对象失败: %v\n", err)
	} else {
		fmt.Printf("✅ 获取所有对象成功，找到 %d 个对象\n", len(allObjects.Data))
		
		// 查找 SUI coin 对象
		suiCoins := 0
		for _, obj := range allObjects.Data {
			if obj.Data != nil && obj.Data.Type != nil {
				if *obj.Data.Type == "0x2::coin::Coin<0x2::sui::SUI>" {
					suiCoins++
					fmt.Printf("   SUI Coin: %s (版本: %d)\n", 
						obj.Data.ObjectId.String(), obj.Data.Version.Uint64())
				}
			}
		}
		fmt.Printf("   其中 SUI coins: %d 个\n", suiCoins)
	}

	fmt.Println("\n========================================")
	fmt.Println("🔍 Gas Objects 调试完成")
}
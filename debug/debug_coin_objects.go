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
	fmt.Println("🔍 调试 Coin Objects 获取")
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

	// 方法1: 尝试使用 GetOwnedObjects 获取对象
	fmt.Println("\n=== 方法1: GetOwnedObjects ===")
	
	// 构造查询选项
	query := types.SuiObjectResponseQuery{
		Filter: &types.SuiObjectDataFilter{
			StructType: &types.StructTag{
				Address: sui_types.SuiAddress{0x2},
				Module:  "coin",
				Name:    "Coin",
				TypeParams: []types.TypeTag{
					{
						Struct: &types.StructTag{
							Address: sui_types.SuiAddress{0x2},
							Module:  "sui",
							Name:    "SUI",
						},
					},
				},
			},
		},
		Options: &types.SuiObjectDataOptions{
			ShowType:    true,
			ShowContent: true,
			ShowOwner:   true,
		},
	}

	objects, err := cli.GetOwnedObjects(context.Background(), *sponsorAddr, &query, nil, nil)
	if err != nil {
		fmt.Printf("❌ GetOwnedObjects 失败: %v\n", err)
		fmt.Printf("   错误类型: %T\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个 SUI coin 对象\n", len(objects.Data))
		for i, obj := range objects.Data {
			if obj.Data != nil {
				fmt.Printf("   对象 %d: %s\n", i+1, obj.Data.ObjectId.String())
				if obj.Data.Content != nil && obj.Data.Content.MoveObject != nil {
					if fields, ok := obj.Data.Content.MoveObject.Fields.(map[string]interface{}); ok {
						if balance, exists := fields["balance"]; exists {
							fmt.Printf("     余额: %v\n", balance)
						}
					}
				}
			}
		}
	}

	// 方法2: 尝试直接调用 RPC
	fmt.Println("\n=== 方法2: 直接 RPC 调用 ===")
	
	// 这里我们可以尝试使用更底层的方法
	// 但首先让我们检查是否是 go-sui 库的版本问题
	
	fmt.Println("检查当前使用的 go-sui 库版本...")
	
	fmt.Println("\n========================================")
	fmt.Println("🔍 Coin Objects 调试完成")
}
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"agent-wallet-gas-sponsor/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run debug_solana_payment.go <transaction_signature>")
		return
	}
	
	sigStr := os.Args[1]
	fmt.Printf("调试Solana交易: %s\n", sigStr)
	fmt.Printf("使用RPC: %s\n", common.SolanaDevnetRPC)
	
	// 验证交易签名格式
	sig, err := solana.SignatureFromBase58(sigStr)
	if err != nil {
		fmt.Printf("❌ 无效的交易签名格式: %v\n", err)
		return
	}
	fmt.Println("✅ 交易签名格式正确")

	client := rpc.New(common.SolanaDevnetRPC)
	
	// 尝试获取交易状态
	for i := 0; i < 5; i++ {
		fmt.Printf("\n--- 尝试 %d/5 ---\n", i+1)
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		// 获取交易状态
		outStatus, errStatus := client.GetSignatureStatuses(
			ctx,
			true, // searchTransactionHistory - 搜索历史交易
			sig,
		)
		
		fmt.Printf("   GetSignatureStatuses 错误: %v\n", errStatus)
		fmt.Printf("   GetSignatureStatuses 结果: %v\n", outStatus)
		
		if errStatus != nil {
			fmt.Printf("❌ GetSignatureStatuses 错误: %v\n", errStatus)
		} else if outStatus == nil {
			fmt.Printf("❌ GetSignatureStatuses 返回 nil\n")
		} else {
			fmt.Printf("   返回数组长度: %d\n", len(outStatus.Value))
			if len(outStatus.Value) == 0 {
				fmt.Printf("❌ GetSignatureStatuses 返回空数组\n")
			} else if outStatus.Value[0] == nil {
				fmt.Printf("❌ GetSignatureStatuses 返回 nil 状态\n")
			} else {
			status := outStatus.Value[0]
			fmt.Printf("✅ 交易状态: %s\n", status.ConfirmationStatus)
			fmt.Printf("   Slot: %d\n", status.Slot)
			if status.Err != nil {
				fmt.Printf("   错误: %v\n", status.Err)
			}
			
			// 如果已确认，尝试获取详情
			if status.ConfirmationStatus == rpc.ConfirmationStatusConfirmed || 
			   status.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				
				fmt.Println("🔍 尝试获取交易详情...")
				txDetails, err := client.GetTransaction(
					ctx,
					sig,
					&rpc.GetTransactionOpts{
						Encoding:   solana.EncodingBase64,
						Commitment: rpc.CommitmentConfirmed,
					},
				)
				
				if err != nil {
					fmt.Printf("❌ 获取交易详情失败: %v\n", err)
				} else if txDetails == nil {
					fmt.Printf("❌ 交易详情为空\n")
				} else {
					fmt.Printf("✅ 交易详情获取成功\n")
					fmt.Printf("   Slot: %d\n", txDetails.Slot)
					if txDetails.Meta != nil {
						fmt.Printf("   执行状态: %v\n", txDetails.Meta.Err)
						fmt.Printf("   PreTokenBalances: %d\n", len(txDetails.Meta.PreTokenBalances))
						fmt.Printf("   PostTokenBalances: %d\n", len(txDetails.Meta.PostTokenBalances))
					}
					cancel()
					return
				}
			}
		}
		
		cancel()
		time.Sleep(3 * time.Second)
	}
	
	fmt.Println("\n❌ 无法获取交易信息")
}
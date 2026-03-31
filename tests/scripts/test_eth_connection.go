package main

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"agent-wallet-gas-sponsor/common"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	fmt.Println("🔍 测试 ETH 连接和配置")
	fmt.Println("========================================")

	// 连接到 Sepolia
	client, err := ethclient.Dial(common.EVMSepoliaRPC)
	if err != nil {
		log.Fatalf("❌ 连接 Ethereum RPC 失败: %v", err)
	}
	defer client.Close()
	fmt.Println("✅ Sepolia RPC 连接成功")

	// 获取网络信息
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatalf("❌ 获取 chain ID 失败: %v", err)
	}
	fmt.Printf("✅ Chain ID: %s\n", chainID.String())

	// 检查 Sponsor 余额
	fmt.Println("\n=== Sponsor 账户信息 ===")
	sponsorAddr := ethcommon.HexToAddress(common.EVMSponsorAddr)
	fmt.Printf("Sponsor 地址: %s\n", sponsorAddr.Hex())
	
	sponsorBalance, err := client.BalanceAt(context.Background(), sponsorAddr, nil)
	if err != nil {
		fmt.Printf("❌ 获取 Sponsor 余额失败: %v\n", err)
	} else {
		ethBalance := new(big.Float).Quo(new(big.Float).SetInt(sponsorBalance), big.NewFloat(1e18))
		fmt.Printf("Sponsor ETH 余额: %.6f ETH\n", ethBalance)
	}

	// 检查用户余额
	fmt.Println("\n=== 用户账户信息 ===")
	userAddr := ethcommon.HexToAddress(common.EVMUserAddr)
	fmt.Printf("用户地址: %s\n", userAddr.Hex())
	
	userBalance, err := client.BalanceAt(context.Background(), userAddr, nil)
	if err != nil {
		fmt.Printf("❌ 获取用户余额失败: %v\n", err)
	} else {
		ethBalance := new(big.Float).Quo(new(big.Float).SetInt(userBalance), big.NewFloat(1e18))
		fmt.Printf("用户 ETH 余额: %.6f ETH\n", ethBalance)
	}

	// 获取当前 gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		fmt.Printf("❌ 获取 gas price 失败: %v\n", err)
	} else {
		gasPriceGwei := new(big.Float).Quo(new(big.Float).SetInt(gasPrice), big.NewFloat(1e9))
		fmt.Printf("当前 Gas Price: %.2f Gwei\n", gasPriceGwei)
	}

	fmt.Println("\n========================================")
	fmt.Println("🔍 ETH 连接测试完成")
	fmt.Println("")
	fmt.Println("💡 下一步:")
	fmt.Printf("   1. 为 Sponsor 充值 ETH: https://faucet.sepolia.dev/ -> %s\n", common.EVMSponsorAddr)
	fmt.Printf("   2. 为用户获取 USDC (需要通过 DEX 或其他方式)\n")
	fmt.Printf("   3. 运行完整测试: ./test_usdc_gas_sponsor_on_eth.sh\n")
}
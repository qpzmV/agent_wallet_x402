package main

import (
	"agent-wallet-gas-sponsor/common"
)

// 支持的网络配置
type NetworkConfig struct {
	Name         string
	SponsorAddr  string
	RPCEndpoint  string
	USDCContract string // USDC合约地址
	Enabled      bool
}

var SupportedNetworks = map[string]NetworkConfig{
	"solana": {
		Name:         "Solana",
		SponsorAddr:  common.SolanaSponsorAddr,
		RPCEndpoint:  common.SolanaDevnetRPC,
		USDCContract: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", // Devnet USDC
		Enabled:      true,
	},
	"ethereum": {
		Name:         "Ethereum",
		SponsorAddr:  common.EVMSponsorAddr,
		RPCEndpoint:  common.EVMSepoliaRPC,
		USDCContract: "0xA0b86a33E6417c4c2f1C6C5b2c5c5c5c5c5c5c5c", // Sepolia USDC (示例)
		Enabled:      true,
	},
	"polygon": {
		Name:         "Polygon",
		SponsorAddr:  common.EVMSponsorAddr,
		RPCEndpoint:  "https://rpc-mumbai.maticvigil.com", // Mumbai testnet
		USDCContract: "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", // Polygon USDC
		Enabled:      false, // 暂时禁用
	},
}

// 获取启用的网络列表
func getEnabledNetworks() []string {
	var networks []string
	for key, config := range SupportedNetworks {
		if config.Enabled {
			networks = append(networks, key)
		}
	}
	return networks
}

// 获取网络配置
func getNetworkConfig(network string) (NetworkConfig, bool) {
	config, exists := SupportedNetworks[network]
	return config, exists && config.Enabled
}
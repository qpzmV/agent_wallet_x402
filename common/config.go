package common

// 为测试链预留的固定 Sponsor 账户
const (
	// EVM (Sepolia)
	EVMSponsorPK   = "0xa4385ca0cf7fc1614e093334d8228d26c39dd65a3f6a49cd21001b6762240b22"
	EVMSponsorAddr = "0x125a63a553f5494313565F3baa099DD73dA500Bc"
	EVMSepoliaRPC  = "https://rpc.sepolia.org"
	EVMBrowser     = "https://sepolia.etherscan.io/tx/"

	// Solana (Devnet)
	SolanaSponsorPK   = "3V2PTXSb2sMR6uyKZZHaouDV8AWQfsgaqRfpgK6NAGBDM8r4oU3kdHAfSFwR948FGtvJaX94d1NLcfMmyatbKdMq"
	SolanaSponsorAddr = "CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK"
	SolanaDevnetRPC   = "https://api.devnet.solana.com"
	SolanaBrowser     = "https://explorer.solana.com/tx/%s?cluster=devnet"

	// Sui (Testnet) - 使用真实生成的助记词派生地址
	// Sponsor: 用户已领币
	SuiSponsorPK   = "ee53b0e3505da82c5f73ed2dc26d368e73468987ec9e014798604724a7374026"
	SuiSponsorAddr = "0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc"

	// User: 用来充当签名的主体，需要领一点币来作为交易对象 (Gas 由 Sponsor 出)
	SuiUserPK   = "678e7c10b240837d9e15f4007b89e34c22b938a0f928e341c2b9a384f923e3c1"
	SuiUserAddr = "0x5f31097cd2bd9957c0de799c088de591ece9747302a49e072528409014ed24dc"

	SuiTestnetRPC = "https://fullnode.testnet.sui.io:443"
	SuiBrowser    = "https://suiscan.xyz/testnet/tx/"
)

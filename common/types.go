package common

type ExecuteRequest struct {
	Chain         string `json:"chain"`          // evm, solana, sui
	TxData        string `json:"tx_data"`        // 原始交易数据 (hex 或 base64)
	UserAddress   string `json:"user_address"`   // 用户钱包地址
	TargetAddress string `json:"target_address"` // 目标合约/收款地址
	UserSignature string `json:"user_signature"` // 用户原始签名
}

type ExecuteResponse struct {
	TxHash  string `json:"tx_hash,omitempty"`
	Status  any    `json:"status"`            // 可以是 string ("success") 或 int (402)
	Message string `json:"message,omitempty"` // 用于 402 等响应的详细说明
	Error   string `json:"error,omitempty"`
}

type PaymentRequest struct {
	Amount   float64 `json:"amount"`   // USDC 金额
	Receiver string  `json:"receiver"` // 收款地址
	Currency string  `json:"currency"` // USDC
}

type X402Response struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Payment X402PaymentInfo `json:"payment"`
}

type X402PaymentInfo struct {
	PriceUSD     float64           `json:"priceUsd"`
	Networks     []string          `json:"networks"`
	Tokens       []string          `json:"tokens"`
	Description  string            `json:"description"`
	CapabilityID string            `json:"capabilityId"`
	Receivers    map[string]string `json:"receivers,omitempty"` // 网络 -> 收款地址的映射
	Receiver     string            `json:"receiver,omitempty"`  // 保持向后兼容
	GasInfo      *GasEstimateInfo  `json:"gas_info,omitempty"`  // Gas估算详情
}

type GasEstimateInfo struct {
	TargetChain    string  `json:"target_chain"`     // 目标执行链
	EstimatedGas   float64 `json:"estimated_gas"`    // 预估gas数量
	NativeToken    string  `json:"native_token"`     // 原生代币
	TokenPriceUSD  float64 `json:"token_price_usd"`  // 代币USD价格
	GasDescription string  `json:"gas_description"`  // Gas费用说明
}

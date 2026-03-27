package common

type ExecuteRequest struct {
	Chain         string `json:"chain"`          // evm, solana, sui
	TxData        string `json:"tx_data"`        // 原始交易数据 (hex 或 base64)
	UserAddress   string `json:"user_address"`   // 用户钱包地址
	TargetAddress string `json:"target_address"` // 目标合约/收款地址
	UserSignature string `json:"user_signature"` // 用户原始签名
}

type ExecuteResponse struct {
	TxHash string `json:"tx_hash,omitempty"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type PaymentRequest struct {
	Amount   float64 `json:"amount"`   // USDC 金额
	Receiver string  `json:"receiver"` // 收款地址
	Currency string  `json:"currency"` // USDC
}

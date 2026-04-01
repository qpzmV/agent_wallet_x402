# x402 端到端测试

## 概述

这个端到端测试演示了完整的x402 gas代付流程：
1. 用户构造目标链交易
2. x402服务器估算gas费用并返回USDT支付金额
3. 用户支付USDT到sponsor地址
4. x402服务器验证支付并代付原生代币gas执行交易

## 测试前准备

### 1. 启动服务

```bash
# 启动执行引擎
cd execution-engine
go run main.go

# 启动x402服务器
cd x402-server
go run main.go config.go gas_calculator.go
```

### 2. 用户地址充值

测试使用以下固定用户地址，需要充值USDT：

**Ethereum (Sepolia):**
- 用户地址: `0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e`
- USDT合约: `0xdAC17F958D2ee523a2206206994597C13D831ec7`
- 充值方式: Sepolia水龙头 → 兑换USDT

**Solana (Devnet):**
- 用户地址: `9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM`
- USDT合约: `Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB`
- 充值方式: Solana水龙头 → 兑换USDT

**Sui (Testnet):**
- 用户地址: `0x5f31097cd2bd9957c0de799c088de591ece9747302a49e072528409014ed24dc`
- USDT合约: `0x2::coin::Coin<0x2::sui::SUI>`
- 充值方式: Sui水龙头 → 兑换USDT

## 运行测试

### 交互式测试
```bash
cd cmd/e2e
go run main.go
```

## 测试流程

### 步骤1: 构造交易
- Solana: 转账1000 lamports
- Ethereum: 转账0.01 ETH  
- Sui: 转账100 MIST

### 步骤2: 获取Gas估算
```
POST /execute (无支付凭证)
↓
402 Payment Required
{
  "payment": {
    "priceUsd": 0.0012,
    "description": "Gas sponsorship for solana transaction",
    "receivers": {
      "solana": "CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK",
      "ethereum": "0x125a63a553f5494313565F3baa099DD73dA500Bc"
    },
    "gas_info": {
      "target_chain": "solana",
      "estimated_gas": 0.000005,
      "native_token": "SOL",
      "token_price_usd": 100.0
    }
  }
}
```

### 步骤3: 用户支付USDT
用户向任意支持的网络的sponsor地址支付相应USDT金额

### 步骤4: 执行代付交易
```
POST /execute (带支付凭证)
X-402-Payment: <payment_proof>
↓
200 OK
{
  "tx_hash": "transaction_hash",
  "status": "success"
}
```

## 支付凭证格式

- **Solana**: Base58编码的交易签名
- **Ethereum**: 0x开头的交易哈希

## 费用计算示例

```
Sui交易需要0.003 SUI gas
SUI价格: $1.50
安全边际: 20%
最终USDT费用: 0.003 × $1.50 × 1.2 = $0.0054
```

## 错误处理

- **Gas估算失败**: 返回默认$0.01费用
- **支付验证失败**: 重新返回402要求支付
- **执行失败**: 返回具体错误信息

## 注意事项

1. 测试环境使用固定的私钥和地址，生产环境需要安全管理
2. 价格数据从CoinGecko API获取，可能有延迟
3. 支付验证目前使用模拟逻辑，生产环境需要完整的链上验证
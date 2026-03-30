# X402 动态Gas费用支付服务器

## 概述

这是一个支持多区块链的x402支付中间件，用于agent web3钱包的代付功能。系统会根据实际的gas费用动态计算USDC支付金额。

## 核心功能

- **动态Gas估算**: 根据交易数据估算实际gas费用
- **实时价格转换**: 将原生代币(SOL/ETH/SUI)转换为USDC等值金额
- **多链支持**: 支持Solana、Ethereum、Sui等多条链
- **安全边际**: 自动添加20%安全边际，避免gas不足

## 支持的网络

- **Solana** ✅ (完整支持)
- **Ethereum** 🚧 (开发中)
- **Sui** 🚧 (开发中)

## API 端点

### 1. 执行交易 (需要支付)
```
POST /execute
```

**请求体:**
```json
{
  "chain": "solana",
  "tx_data": "交易数据",
  "user_address": "用户地址",
  "target_address": "目标地址",
  "user_signature": "用户签名"
}
```

**无支付时返回 402 (包含动态gas信息):**
```json
{
  "status": 402,
  "message": "Payment required.",
  "payment": {
    "priceUsd": 0.0012,
    "networks": ["solana", "ethereum"],
    "tokens": ["USDC"],
    "description": "Gas sponsorship for solana transaction",
    "capabilityId": "gas-sponsor",
    "receivers": {
      "solana": "CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK",
      "ethereum": "0x125a63a553f5494313565F3baa099DD73dA500Bc"
    },
    "gas_info": {
      "target_chain": "solana",
      "estimated_gas": 0.000005,
      "native_token": "SOL",
      "token_price_usd": 100.0,
      "gas_description": "Solana transaction gas: 0.000005 SOL (~$0.0005)"
    }
  }
}
```

### 2. Gas费用估算
```
POST /estimate-gas
```

**请求体:** (同execute端点)

**响应:**
```json
{
  "gas_estimate": {
    "chain": "solana",
    "estimated_gas": 0.000006,
    "native_token": "SOL",
    "token_price_usd": 100.0,
    "usdc_amount": 0.0012,
    "description": "Solana transaction gas: 0.000006 SOL (~$0.0012)"
  },
  "payment_info": {
    "amount_usdc": 0.0012,
    "receivers": {
      "solana": "CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK",
      "ethereum": "0x125a63a553f5494313565F3baa099DD73dA500Bc"
    },
    "supported_networks": ["solana", "ethereum"]
  }
}
```

## Gas估算逻辑

### Solana
- 基础费用: 0.000005 SOL
- 复杂交易: 2x 倍数
- 实时SOL价格获取

### Ethereum
- 基础Gas Limit: 21,000 (转账) / 100,000 (合约)
- Gas Price: 20 gwei (可配置)
- 实时ETH价格获取

### Sui
- 基础费用: 0.001 SUI
- 复杂交易: 0.003 SUI
- 实时SUI价格获取

## 支付流程

1. **Agent发起请求** → 调用 `/execute` 或 `/estimate-gas`
2. **系统估算gas** → 解析交易数据，计算实际gas费用
3. **价格转换** → 将原生代币费用转换为USDC金额
4. **返回402** → 包含精确的USDC支付金额和多链收款地址
5. **用户选择网络** → 根据自己的钱包选择支付网络
6. **执行支付** → 向对应收款地址转账USDC
7. **重试请求** → 带支付凭证重新调用 `/execute`
8. **验证通过** → 转发到执行引擎完成代付

## 费用计算示例

```
Sui交易: 0.3 SUI gas费用
SUI价格: $1.50
USDC金额: 0.3 × $1.50 × 1.2 (安全边际) = $0.54 USDC

用户支付: $0.54 USDC 到 Sui收款地址
系统代付: 0.3 SUI 的实际gas费用
```

## 安全特性

- **防重放攻击**: 支付凭证只能使用一次
- **安全边际**: 自动添加20%缓冲，避免gas不足
- **最小/最大限制**: 最少$0.01，最多$10.00
- **实时价格**: 从CoinGecko获取最新价格

## 运行

```bash
go run main.go config.go gas_calculator.go
```

## 测试

```bash
go run test_client.go
```
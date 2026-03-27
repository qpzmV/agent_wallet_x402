# x402 Gas Sponsor System

这是一个基于 x402 (Payment Required) 协议的 Web3 Agent Gas 代付系统。它支持 EVM (Sepolia), Solana (Devnet) 和 Sui (Testnet)。

## 架构

- **Agent**: 调用接口。
- **x402-server (端口 8080)**: 支付层。如果请求未携带支付 Token，则返回 402 和付款地址。
- **execution-engine (端口 8081)**: 执行层。负责 Gas 代付（Sponsor）和链上交易提交。

## 如何运行

### 1. 启动执行引擎
```bash
go run execution-engine/main.go
```

### 2. 启动 x402 中间件
```bash
go run x402-server/main.go
```

### 3. 配置
在 `execution-engine/sponsor/*.go` 文件中，填写您的 Sponsor 账户私钥。

## 测试

运行集成测试：
```bash
go test ./tests/...
```

## 逻辑流程
1. 用户发起 `POST /execute` 到 8080 端口。
2. 8080 中间件检查 `X-Payment-Token`。
3. 若无 Token，返回 `402 Payment Required`，并在 Header 中给出 USDC 收款地址和金额。
4. 用户支付后，重新发起请求并携带 Token。
5. 8080 转发请求到 8081。
6. 8081 根据链类型调用 Sponsor 逻辑，支付 Gas 并广播交易。

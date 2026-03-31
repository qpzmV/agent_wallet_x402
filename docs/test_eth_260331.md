# ETH USDC Gas 代付测试实现

## 概述

基于 SUI 测试的成功经验，实现了 ETH 版本的 USDC Gas 代付测试，展示在 Ethereum Sepolia 测试网上的 HTTP 402 Payment Required 协议。

## 文件结构

```
cmd/eth-gas-test/main.go          # ETH 测试主程序
test_usdc_gas_sponsor_on_eth.sh   # ETH 测试脚本
test_eth_connection.go            # ETH 连接测试工具
```

## 核心功能

### 1. ETH 连接和配置
- **网络**: Ethereum Sepolia Testnet
- **RPC**: `https://ethereum-sepolia-rpc.publicnode.com`
- **浏览器**: `https://sepolia.etherscan.io/tx/`
- **USDC 合约**: `0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238` (Sepolia USDC)

### 2. 账户配置
```go
// Sponsor 账户 (代付 gas)
EVMSponsorAddr = "0x125a63a553f5494313565F3baa099DD73dA500Bc"
EVMSponsorPK   = "0xa4385ca0cf7fc1614e093334d8228d26c39dd65a3f6a49cd21001b6762240b22"

// 用户账户 (有 USDC，无 ETH)
EVMUserAddr = "0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e"  
EVMUserPK   = "0xa4385ca0cf7fc1614e093334d8228d26c39dd65a3f6a49cd21001b6762240b23"
```

### 3. 测试流程

1. **检查 Sponsor ETH 余额** - 确保有足够 ETH 代付 gas
2. **构造 USDC 转账交易** - 用户转 1 USDC，Sponsor 代付 gas
3. **获取 Gas 费用估算** - 调用 x402 服务器获取费用
4. **用户支付 Gas 费用** - 转 USDC 给 Sponsor 作为 gas 费
5. **执行原本的转账** - 使用支付凭证执行用户的 USDC 转账
6. **等待交易确认** - 监控 ETH 交易状态

### 4. 关键技术实现

#### ERC20 USDC 操作
```go
// ERC20 ABI for transfer and balanceOf
const erc20ABI = `[
    {"name": "balanceOf", "inputs": [{"name": "_owner", "type": "address"}], ...},
    {"name": "transfer", "inputs": [{"name": "_to", "type": "address"}, {"name": "_value", "type": "uint256"}], ...}
]`

// 构造 USDC 转账交易
func buildUSDCTransferTx(recipientHex string, amount *big.Int) (string, error) {
    // 1. 连接 ETH 客户端
    // 2. 解析用户私钥和地址
    // 3. 检查 USDC 余额
    // 4. 获取 nonce 和 gas price
    // 5. 构造 ERC20 transfer 调用
    // 6. 签名交易
    // 7. 返回 hex 编码的交易数据
}
```

#### 余额检查
```go
func checkUserUSDCBalance(client *ethclient.Client, userAddr common.Address, requiredAmount *big.Int) error {
    // 调用 ERC20 balanceOf 方法
    data, _ := parsedABI.Pack("balanceOf", userAddr)
    result, _ := client.CallContract(context.Background(), ethereum.CallMsg{
        To: &contractAddr, Data: data,
    }, nil)
    balance := new(big.Int).SetBytes(result)
    // 检查余额是否充足
}
```

#### 交易确认
```go
func waitForConfirmation(txHashStr string) error {
    // ETH 确认时间较长，最多等待 90 秒
    for i := 0; i < 30; i++ {
        time.Sleep(3 * time.Second)
        receipt, err := client.TransactionReceipt(context.Background(), txHash)
        if err == nil && receipt != nil && receipt.Status == 1 {
            return nil // 交易成功
        }
    }
}
```

## 使用方法

### 1. 准备测试环境
```bash
# 为 Sponsor 充值 ETH
# 访问: https://faucet.sepolia.dev/
# 地址: 0x125a63a553f5494313565F3baa099DD73dA500Bc

# 为用户获取 USDC (需要通过 DEX 或其他方式)
# 地址: 0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e
```

### 2. 运行测试
```bash
# 测试连接
go run test_eth_connection.go

# 运行完整测试
./test_usdc_gas_sponsor_on_eth.sh
```

### 3. 预期结果
```
🚀 ETH USDC Gas代付完整测试
========================================
✅ Sponsor ETH余额充足
✅ USDC转账交易构造成功  
✅ Gas估算结果: 需要支付 $0.01 USDC
✅ 用户支付完成! (支付交易)
🚀 USDC转账交易执行成功! (原本交易)
✅ 交易已确认

ETH USDC代付Gas测试完成!
🎯 用户转账: 1.000000 USDC
💸 用户支付: $0.010000 USDC (gas费)  
⛽ 我们代付: 2笔交易的ETH gas费
```

## 与 SUI 版本的差异

| 特性 | SUI 版本 | ETH 版本 |
|------|----------|----------|
| **网络** | SUI Testnet | Ethereum Sepolia |
| **Gas 代币** | SUI | ETH |
| **USDC 标准** | Coin<USDC> | ERC20 |
| **交易构造** | PTB + Pay() | ERC20 transfer |
| **Object 管理** | ObjectRef + Version | Nonce |
| **确认时间** | ~2秒 | ~15秒 |
| **Gas 估算** | 固定 budget | 动态 gasLimit |

## 技术挑战和解决方案

### 1. **ERC20 合约交互**
- **挑战**: 需要正确编码 ERC20 函数调用
- **解决**: 使用 go-ethereum 的 ABI 包进行编码/解码

### 2. **Nonce 管理**  
- **挑战**: 连续交易需要正确的 nonce 序列
- **解决**: 每次构造交易时重新获取最新 nonce

### 3. **Gas 估算**
- **挑战**: ETH gas 费用波动较大
- **解决**: 使用 EstimateGas + SuggestGasPrice 动态计算

### 4. **交易确认**
- **挑战**: ETH 确认时间较长且不稳定
- **解决**: 增加超时时间到 90 秒，定期轮询状态

## 测试状态

- ✅ **代码实现完成** - 所有核心功能已实现
- ✅ **编译成功** - 无语法错误
- ✅ **RPC 连接正常** - Sepolia 网络可访问
- ⚠️ **需要测试币** - Sponsor 需要 ETH，用户需要 USDC
- 🔄 **待完整测试** - 需要充值后进行端到端测试

## 下一步

1. **获取测试币**
   - Sponsor ETH: https://faucet.sepolia.dev/
   - 用户 USDC: 通过 DEX 或测试网水龙头

2. **运行完整测试**
   - 验证 HTTP 402 协议在 ETH 上的实现
   - 确认 gas 代付机制正常工作

3. **性能优化**
   - 优化 gas 估算算法
   - 改进交易确认机制
   - 添加重试逻辑

这个实现展示了 HTTP 402 Payment Required 协议在不同区块链上的通用性，用户可以用 USDC 支付 gas 费，而服务提供商代付实际的原生代币成本。

===
# 执行日志
% ./test_usdc_gas_sponsor_on_eth.sh
🚀 ETH USDC Gas代付完整测试
========================================
清理旧进程...
编译组件...
✅ 编译成功
启动服务...
等待服务启动...
✅ 服务已启动

⚠️  重要提醒:
   请确保Sponsor地址有ETH: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   获取测试ETH: https://faucet.sepolia.dev/

   请确保用户地址有USDC: 0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e
   查询账户: https://sepolia.etherscan.io/address/0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e
   获取测试USDC: 使用Sepolia USDC水龙头或DEX

🎯 测试场景:
   1. 用户有USDC，没有ETH
   2. 用户想转1 USDC给别人
   3. 用户先转USDC给我们作为gas费 (我们代付这笔转账的ETH gas)
   4. 第3通过x402验证后，我们再代付用户原本想要的USDC转账的ETH gas

========================================
   ETH USDC Gas代付完整测试
========================================
Sponsor地址: 0x125a63a553f5494313565F3baa099DD73dA500Bc
用户地址: 0x84b13a5Ebb5dFBd6b9ffADababFe5b23FF50bbDa (有USDC，无ETH)
网络: Ethereum Sepolia Testnet
浏览器: https://sepolia.etherscan.io/tx/
测试场景: 用户转1 USDC，Sponsor代付ETH gas费

=== 步骤1: 检查Sponsor ETH余额 ===
   当前ETH余额: 0.040000 ETH
✅ Sponsor ETH余额充足

=== 步骤2: 构造用户USDC转账交易 ===
💰 用户想转1 USDC给别人，但没有ETH支付gas
⚠️  注意: 请确保用户地址 0x84b13a5Ebb5dFBd6b9ffADababFe5b23FF50bbDa 已有USDC余额
   用户USDC余额: 20.000000 USDC
   用户USDC转账: 1.000000 USDC
   接收者: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   Gas Limit: 62976
   Gas Price: %!s(*big.Float=0.001000012) Gwei
✅ USDC转账交易构造成功

=== 步骤3: 获取Gas费用估算 ===
✅ Gas估算结果:
   需要支付: $4.986720 USDC
   原生Gas: 0.002000 ETH
   ETH价格: $2077.8000
   ETH收款地址: 0x125a63a553f5494313565F3baa099DD73dA500Bc

=== 步骤4: 用户支付Gas费用 (USDC → Sponsor) ===
💰 用户需要支付: $4.986720 USDC 作为gas费
💡 原理: 用户转USDC给我们，我们代付这笔转账的ETH gas
   用户USDC余额: 20.000000 USDC
   用户USDC转账: 4.986720 USDC
   接收者: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   Gas Limit: 62976
   Gas Price: %!s(*big.Float=0.001000012) Gwei
🔄 执行用户支付gas费的USDC转账 (bootstrap代付)...
   [响应] 状态: 200
✅ 用户支付完成!
   支付交易Hash: 0x470efcb98ffe13f66a0e30a79bcf99e3d31d533d5fca9b30aea0db6e76b89e7d
   浏览器查看: https://sepolia.etherscan.io/tx/0x470efcb98ffe13f66a0e30a79bcf99e3d31d533d5fca9b30aea0db6e76b89e7d
   我们代付了用户支付gas费的ETH gas

=== 步骤5: 执行用户原本的USDC转账 ===
🔄 等待支付交易链上确认后，执行原交易...
   🔄 重新构造交易以获取最新的 nonce...
   用户USDC余额: 20.000000 USDC
   用户USDC转账: 1.000000 USDC
   接收者: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   Gas Limit: 62976
   Gas Price: %!s(*big.Float=0.001000012) Gwei
   [响应] 状态: 200
🚀 USDC转账交易执行成功!
   交易Hash: 0x42fc3e0c0dff1b06a459c1e9e2d7327eb0b8a636b159f78b7681de99f47a8ae9
   浏览器查看: https://sepolia.etherscan.io/tx/0x42fc3e0c0dff1b06a459c1e9e2d7327eb0b8a636b159f78b7681de99f47a8ae9
   ✅ 用户成功转账1.000000 USDC，我们代付了ETH gas费

=== 步骤6: 等待交易确认 ===
   等待确认.. ✅
✅ 交易已确认

========================================
   ETH USDC代付Gas测试完成!
   🎯 用户转账: 1.000000 USDC
   💸 用户支付: $4.986720 USDC (gas费)
   ⛽ 我们代付: 2笔交易的ETH gas费
   📊 支付交易: https://sepolia.etherscan.io/tx/0x470efcb98ffe13f66a0e30a79bcf99e3d31d533d5fca9b30aea0db6e76b89e7d
   📊 转账交易: https://sepolia.etherscan.io/tx/0x42fc3e0c0dff1b06a459c1e9e2d7327eb0b8a636b159f78b7681de99f47a8ae9
   💡 完整流程: 用户转USDC给我们→我们代付→用户转USDC给别人→我们代付
========================================

清理进程...
🎉 测试完成!
edy@edydeMacBook-Pro-4 agent_wallet_x402 % 


===
改的新版
===
完整的EVM链上验证实现
主要功能
交易哈希格式验证：确保是有效的以太坊交易哈希格式（0x开头，66字符长度）

防重放攻击：使用内存缓存记录已使用的交易哈希，防止同一笔交易被重复使用

链上交易确认验证：

连接以太坊Sepolia测试网RPC
重试机制等待交易确认（最多15次，每次间隔2秒）
验证交易执行状态（status=1表示成功）
USDC转账验证：

验证交易是发送到正确的USDC合约地址
解析Transfer事件日志，提取转账金额和接收者
验证接收者是指定的Sponsor地址
验证转账金额符合要求（允许0.001 USDC的误差范围）
完整的错误处理：

网络连接失败
交易未确认或不存在
交易执行失败
转账金额不足
接收者地址不正确
测试结果
测试成功完成了完整的gas代付流程：

✅ 支付验证：用户支付4.983456 USDC作为gas费
✅ 链上验证：系统验证了支付交易的真实性和有效性
✅ Gas代付：成功为用户的1 USDC转账代付ETH gas费
✅ 交易确认：两笔交易都在链上成功确认
现在x402-server具备了完整的EVM链上支付验证能力，不再是简单的格式检查，而是真正从以太坊区块链上验证交易的存在、状态、金额和接收者，确保了支付系统的安全性和可靠性。

===
% ./test_usdc_gas_sponsor_on_eth.sh
🚀 ETH USDC Gas代付完整测试
========================================
清理旧进程...
编译组件...
✅ 编译成功
启动服务...
等待服务启动...
✅ 服务已启动

⚠️  重要提醒:
   请确保Sponsor地址有ETH: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   获取测试ETH: https://faucet.sepolia.dev/

   请确保用户地址有USDC: 0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e
   查询账户: https://sepolia.etherscan.io/address/0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e
   获取测试USDC: 使用Sepolia USDC水龙头或DEX

🎯 测试场景:
   1. 用户有USDC，没有ETH
   2. 用户想转1 USDC给别人
   3. 用户先转USDC给我们作为gas费 (我们代付这笔转账的ETH gas)
   4. 第3通过x402验证后，我们再代付用户原本想要的USDC转账的ETH gas

========================================
   ETH USDC Gas代付完整测试
========================================
Sponsor地址: 0x125a63a553f5494313565F3baa099DD73dA500Bc
用户地址: 0x84b13a5Ebb5dFBd6b9ffADababFe5b23FF50bbDa (有USDC，无ETH)
网络: Ethereum Sepolia Testnet
浏览器: https://sepolia.etherscan.io/tx/
测试场景: 用户转1 USDC，Sponsor代付ETH gas费

=== 步骤1: 检查Sponsor ETH余额 ===
   当前ETH余额: 0.040000 ETH
✅ Sponsor ETH余额充足

=== 步骤2: 构造用户USDC转账交易 ===
💰 用户想转1 USDC给别人，但没有ETH支付gas
⚠️  注意: 请确保用户地址 0x84b13a5Ebb5dFBd6b9ffADababFe5b23FF50bbDa 已有USDC余额
   用户USDC余额: 9.030208 USDC
   用户USDC转账: 1.000000 USDC
   接收者: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   Gas Limit: 45427
   Gas Price: %!s(*big.Float=0.001000013) Gwei
✅ USDC转账交易构造成功

=== 步骤3: 获取Gas费用估算 ===
✅ Gas估算结果:
   需要支付: $4.983456 USDC
   原生Gas: 0.002000 ETH
   ETH价格: $2076.4400
   ETH收款地址: 0x125a63a553f5494313565F3baa099DD73dA500Bc

=== 步骤4: 用户支付Gas费用 (USDC → Sponsor) ===
💰 用户需要支付: $4.983456 USDC 作为gas费
💡 原理: 用户转USDC给我们，我们代付这笔转账的ETH gas
   用户USDC余额: 9.030208 USDC
   用户USDC转账: 4.983456 USDC
   接收者: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   Gas Limit: 45427
   Gas Price: %!s(*big.Float=0.001000013) Gwei
🔄 执行用户支付gas费的USDC转账 (bootstrap代付)...
   [响应] 状态: 200
✅ 用户支付完成!
   支付交易Hash: 0x9866e742a39c1fc683e29ab4600fada1b884e9b4f38dbf708ec4044b871cb9b7
   浏览器查看: https://sepolia.etherscan.io/tx/0x9866e742a39c1fc683e29ab4600fada1b884e9b4f38dbf708ec4044b871cb9b7
   我们代付了用户支付gas费的ETH gas

=== 步骤5: 执行用户原本的USDC转账 ===
🔄 等待支付交易链上确认后，执行原交易...
   🔄 重新构造交易以获取最新的 nonce...
   用户USDC余额: 9.030208 USDC
   用户USDC转账: 1.000000 USDC
   接收者: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   Gas Limit: 45427
   Gas Price: %!s(*big.Float=0.001000013) Gwei
   [响应] 状态: 200
🚀 USDC转账交易执行成功!
   交易Hash: 0x47ebd388b824507ed7bde0ff29dd016b2c1fec4c8fce9f5e0d317919a203b2c9
   浏览器查看: https://sepolia.etherscan.io/tx/0x47ebd388b824507ed7bde0ff29dd016b2c1fec4c8fce9f5e0d317919a203b2c9
   ✅ 用户成功转账1.000000 USDC，我们代付了ETH gas费

=== 步骤6: 等待交易确认 ===
   等待确认.... ✅
✅ 交易已确认

========================================
   ETH USDC代付Gas测试完成!
   🎯 用户转账: 1.000000 USDC
   💸 用户支付: $4.983456 USDC (gas费)
   ⛽ 我们代付: 2笔交易的ETH gas费
   📊 支付交易: https://sepolia.etherscan.io/tx/0x9866e742a39c1fc683e29ab4600fada1b884e9b4f38dbf708ec4044b871cb9b7
   📊 转账交易: https://sepolia.etherscan.io/tx/0x47ebd388b824507ed7bde0ff29dd016b2c1fec4c8fce9f5e0d317919a203b2c9
   💡 完整流程: 用户转USDC给我们→我们代付→用户转USDC给别人→我们代付
========================================

清理进程...
🎉 测试完成!
edy@edydeMacBook-Pro-4 agent_wallet_x402 % 


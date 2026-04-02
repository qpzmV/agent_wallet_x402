% ./tests/scripts/test_usdc_gas_sponsor_on_eth.sh
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
   用户USDC余额: 43.046752 USDC
   用户USDC转账: 1.000000 USDC
   接收者: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   Gas Limit: 45427
   Gas Price: %!s(*big.Float=0.031459515) Gwei
✅ USDC转账交易构造成功

=== 步骤3: 获取Gas费用估算 ===
✅ Gas估算结果:
   需要支付: $4.972272 USDC
   原生Gas: 0.002000 ETH
   ETH价格: $2071.7800
   ETH收款地址: 0x125a63a553f5494313565F3baa099DD73dA500Bc

=== 步骤4: 用户支付Gas费用 (USDC → Sponsor) ===
💰 用户需要支付: $4.972272 USDC 作为gas费
💡 原理: 用户转USDC给我们，我们代付这笔转账的ETH gas
   用户USDC余额: 43.046752 USDC
   用户USDC转账: 4.972272 USDC
   接收者: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   Gas Limit: 45427
   Gas Price: %!s(*big.Float=0.035186838) Gwei
🔄 执行用户支付gas费的USDC转账 (bootstrap代付)...
   [响应] 状态: 200
✅ 用户支付完成!
   支付交易Hash: 0x5a2889284d1989ded86cd1fe8729ac8ff1bc5a686a807de443ce2c88be1cf261
   浏览器查看: https://sepolia.etherscan.io/tx/0x5a2889284d1989ded86cd1fe8729ac8ff1bc5a686a807de443ce2c88be1cf261
   我们代付了用户支付gas费的ETH gas

=== 步骤5: 执行用户原本的USDC转账 ===
🔄 等待支付交易链上确认后，执行原交易...
   🔄 重新构造交易以获取最新的 nonce...
   用户USDC余额: 43.046752 USDC
   用户USDC转账: 1.000000 USDC
   接收者: 0x125a63a553f5494313565F3baa099DD73dA500Bc
   Gas Limit: 45427
   Gas Price: %!s(*big.Float=0.035186838) Gwei
   [响应] 状态: 200
🚀 USDC转账交易执行成功!
   交易Hash: 0x4ef427f4897185460ec5743750db626bdca025c642399782721a00b37762a510
   浏览器查看: https://sepolia.etherscan.io/tx/0x4ef427f4897185460ec5743750db626bdca025c642399782721a00b37762a510
   ✅ 用户成功转账1.000000 USDC，我们代付了ETH gas费

=== 步骤6: 等待交易确认 ===
   等待确认... ✅
✅ 交易已确认

========================================
   ETH USDC代付Gas测试完成!
   🎯 用户转账: 1.000000 USDC
   💸 用户支付: $4.972272 USDC (gas费)
   ⛽ 我们代付: 2笔交易的ETH gas费
   📊 支付交易: https://sepolia.etherscan.io/tx/0x5a2889284d1989ded86cd1fe8729ac8ff1bc5a686a807de443ce2c88be1cf261
   📊 转账交易: https://sepolia.etherscan.io/tx/0x4ef427f4897185460ec5743750db626bdca025c642399782721a00b37762a510
   💡 完整流程: 用户转USDC给我们→我们代付→用户转USDC给别人→我们代付
========================================

清理进程...
🎉 测试完成!
edy@edydeMacBook-Pro-4 agent_wallet_x402 % git status

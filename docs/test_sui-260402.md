% ./tests/scripts/test_usdc_gas_sponsor_on_sui.sh
🚀 SUI USDC Gas代付完整测试
========================================
清理旧进程...
编译组件...
✅ 编译成功
启动服务...
等待服务启动...
✅ 服务已启动

⚠️  重要提醒:
   请确保Sponsor地址有SUI: 0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc
   获取测试SUI: https://faucet.testnet.sui.io/

   请确保用户地址有USDC: 0x5f31097cd2bd9957c0de799c088de591ece9747302a49e072528409014ed24dc
   查询账户: https://suiscan.xyz/testnet/account/0x5f31097cd2bd9957c0de799c088de591ece9747302a49e072528409014ed24dc

🎯 测试场景:
   1. 用户有USDC，没有SUI
   2. 用户想转1 USDC给别人
   3. 用户先转USDC给我们作为gas费 (我们代付这笔转账的SUI gas)
   4. 第3通过x402验证后，我们再代付用户原本想要的USDC转账的SUI gas

========================================
   SUI USDC Gas代付完整测试
========================================
Sponsor地址: 0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc
用户地址: 0x5f31097cd2bd9957c0de799c088de591ece9747302a49e072528409014ed24dc (有USDC，无SUI)
网络: SUI Testnet
浏览器: https://suiscan.xyz/testnet/tx/
测试场景: 用户转1 USDC，Sponsor代付SUI gas费

=== 步骤1: 检查Sponsor SUI余额 ===
   当前SUI余额: 0.953438 SUI
✅ Sponsor SUI余额充足

=== 步骤2: 构造用户USDC转账交易 ===
💰 用户想转1 USDC给别人，但没有SUI支付gas
⚠️  注意: 请确保用户地址 0x5f31097cd2bd9957c0de799c088de591ece9747302a49e072528409014ed24dc 已有USDC余额
   用户USDC余额: 10.900000 USDC
   转账金额: 1.000000 USDC
   接收者: 0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc
✅ USDC转账交易构造成功

=== 步骤3: 获取Gas费用估算 ===
✅ Gas估算结果:
   需要支付: $0.010000 USDC
   原生Gas: 0.001000 SUI
   SUI价格: $0.8615
   SUI收款地址: 0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc

=== 步骤4: 用户支付Gas费用 (USDC → Sponsor) ===
💰 用户需要支付: $0.010000 USDC 作为gas费
💡 原理: 用户转USDC给我们，我们代付这笔转账的SUI gas
   用户USDC余额: 10.900000 USDC
   转账金额: 0.010000 USDC
   接收者: 0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc
🔄 执行用户支付gas费的USDC转账 (bootstrap代付)...
   [响应] 状态: 200
✅ 用户支付完成!
   支付交易Digest: 34iACgPLeqcn9XRECCSJf9nhkEjiHYJATup8XsD5Rpfh
   浏览器查看: https://suiscan.xyz/testnet/tx/34iACgPLeqcn9XRECCSJf9nhkEjiHYJATup8XsD5Rpfh
   我们代付了用户支付gas费的SUI gas

=== 步骤5: 执行用户原本的USDC转账 ===
🔄 等待支付交易链上确认后，执行原交易...
   🔄 重新构造交易以获取最新的 coin objects...
   用户USDC余额: 10.890000 USDC
   转账金额: 1.000000 USDC
   接收者: 0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc
   [响应] 状态: 200
🚀 USDC转账交易执行成功!
   交易Digest: B8wEnpfGznEghnfzjqSDWFgMwWvfnrK3j5jL53Msd4hT
   浏览器查看: https://suiscan.xyz/testnet/tx/B8wEnpfGznEghnfzjqSDWFgMwWvfnrK3j5jL53Msd4hT
   ✅ 用户成功转账1.000000 USDC，我们代付了SUI gas费

=== 步骤6: 等待交易确认 ===
   等待确认. ✅
✅ 交易已确认

========================================
   SUI USDC代付Gas测试完成!
   🎯 用户转账: 1.000000 USDC
   💸 用户支付: $0.010000 USDC (gas费)
   ⛽ 我们代付: 2笔交易的SUI gas费
   📊 支付交易: https://suiscan.xyz/testnet/tx/34iACgPLeqcn9XRECCSJf9nhkEjiHYJATup8XsD5Rpfh
   📊 转账交易: https://suiscan.xyz/testnet/tx/B8wEnpfGznEghnfzjqSDWFgMwWvfnrK3j5jL53Msd4hT
   💡 完整流程: 用户转USDC给我们→我们代付→用户转USDC给别人→我们代付
========================================

清理进程...
🎉 测试完成!
edy@edydeMacBook-Pro-4 agent_wallet_x402 % 
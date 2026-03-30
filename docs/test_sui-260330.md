🎯 测试场景:
   1. 用户有20 USDC，没有SUI
   2. 用户想转1 USDC给别人, 通过x402 server得到转gas的交易
   3. 用户先转USDC给我们作为gas费 (我们代付这笔转账的SUI gas)
   4. 第3通过x402验证后，我们再代付用户原本想要的USDC转账的SUI gas
模仿./test_usdc_gas_sponsor.sh, 搞一个./test_usdc_gas_sponsor_on_sui.sh   

% go build -o sui-gas-test ./cmd/sui-gas-test/main.go && ./test_usdc_gas_sponsor_on
_sui.sh
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
   当前SUI余额: 0.995657 SUI
✅ Sponsor SUI余额充足

=== 步骤2: 构造用户USDC转账交易 ===
💰 用户想转1 USDC给别人，但没有SUI支付gas
⚠️  注意: 请确保用户地址 0x5f31097cd2bd9957c0de799c088de591ece9747302a49e072528409014ed24dc 已有USDC余额
   用户USDC余额: 19.990000 USDC
   转账金额: 1.000000 USDC
   接收者: 0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc
✅ USDC转账交易构造成功

=== 步骤3: 获取Gas费用估算 ===
✅ Gas估算结果:
   需要支付: $0.010000 USDC
   原生Gas: 0.001000 SUI
   SUI价格: $0.8819
   SUI收款地址: 0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc

=== 步骤4: 用户支付Gas费用 (USDC → Sponsor) ===
💰 用户需要支付: $0.010000 USDC 作为gas费
💡 原理: 用户转USDC给我们，我们代付这笔转账的SUI gas
   用户USDC余额: 19.990000 USDC
   转账金额: 0.010000 USDC
   接收者: 0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc
🔄 执行用户支付gas费的USDC转账 (bootstrap代付)...
   [响应] 状态: 200
✅ 用户支付完成!
   支付交易Digest: 3gbs1spfvecWVo5Mofq1km9GuLdfn8FYQkeVavQwgHd9
   浏览器查看: https://suiscan.xyz/testnet/tx/3gbs1spfvecWVo5Mofq1km9GuLdfn8FYQkeVavQwgHd9
   我们代付了用户支付gas费的SUI gas

=== 步骤5: 执行用户原本的USDC转账 ===
🔄 等待支付交易链上确认后，执行原交易...
   🔄 重新构造交易以获取最新的 coin objects...
   用户USDC余额: 19.980000 USDC
   转账金额: 1.000000 USDC
   接收者: 0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc
   [响应] 状态: 200
🚀 USDC转账交易执行成功!
   交易Digest: DQq2tV5LZ2Co9L7NG1E1Jpq9E5Mh74EcK6thQgfb8Q1C
   浏览器查看: https://suiscan.xyz/testnet/tx/DQq2tV5LZ2Co9L7NG1E1Jpq9E5Mh74EcK6thQgfb8Q1C
   ✅ 用户成功转账1.000000 USDC，我们代付了SUI gas费

=== 步骤6: 等待交易确认 ===
   等待确认. ✅
✅ 交易已确认

========================================
   SUI USDC代付Gas测试完成!
   🎯 用户转账: 1.000000 USDC
   💸 用户支付: $0.010000 USDC (gas费)
   ⛽ 我们代付: 2笔交易的SUI gas费
   📊 支付交易: https://suiscan.xyz/testnet/tx/3gbs1spfvecWVo5Mofq1km9GuLdfn8FYQkeVavQwgHd9
   📊 转账交易: https://suiscan.xyz/testnet/tx/DQq2tV5LZ2Co9L7NG1E1Jpq9E5Mh74EcK6thQgfb8Q1C
   💡 完整流程: 用户转USDC给我们→我们代付→用户转USDC给别人→我们代付
========================================

清理进程...
🎉 测试完成!
edy@edydeMacBook-Pro-4 agent_wallet_x402 % 

===
问题解决总结
我们成功解决了两个关键问题：

1. GetCoins Hex 解析错误 ✅
问题：encoding/hex: invalid byte: U+0079 'y'
根因：当 limit=1 时，SUI RPC 返回的第一个 coin object 包含无效 hex 数据
解决方案：将 GetCoins 的 limit 参数从 1 改为 2
代码修改：cli.GetCoins(context.Background(), *sponsorAddr, nil, nil, 2)
2. Object Version 冲突 ✅
问题：第二笔交易使用了已被第一笔交易消费的旧版本 coin objects
根因：SUI 链上 objects 被消费后版本号会改变，需要重新获取最新版本
解决方案：在执行第二笔交易前重新构造交易，获取最新的 coin objects
代码修改：在步骤5中调用 buildUSDCTransferTx 重新构造交易
测试结果
✅ 完整的 SUI USDC Gas 代付流程成功执行：

用户状态：有 19.99 USDC，无 SUI
第一笔交易：用户支付 0.01 USDC 作为 gas 费给 Sponsor
第二笔交易：用户转账 1 USDC 给接收者
Gas 代付：Sponsor 为两笔交易都代付了 SUI gas 费
最终状态：用户剩余 18.98 USDC，成功完成转账
交易链接
支付交易：https://suiscan.xyz/testnet/tx/3gbs1spfvecWVo5Mofq1km9GuLdfn8FYQkeVavQwgHd9
转账交易：https://suiscan.xyz/testnet/tx/DQq2tV5LZ2Co9L7NG1E1Jpq9E5Mh74EcK6thQgfb8Q1C
这个解决方案展示了完整的 HTTP 402 Payment Required 协议在 SUI 链上的实现，用户可以用 USDC 支付 gas 费，而 Sponsor 代付实际的 SUI gas 成本。
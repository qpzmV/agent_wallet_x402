 % ./test_usdc_gas_sponsor.sh   
🚀 USDC Gas代付完整测试
========================================
清理旧进程...
编译组件...
✅ 编译成功
启动服务...
等待服务启动...
✅ 服务已启动

⚠️  重要提醒:
   请确保Sponsor地址有SOL: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK
   获取测试SOL: https://faucet.solana.com/

🎯 测试场景:
   1. 用户有20 USDC，没有SOL
   2. 用户想转1 USDC给别人
   3. 用户先转USDC给我们作为gas费 (我们代付这笔转账的SOL gas)
   4. 我们代付用户原本想要的USDC转账的SOL gas

========================================
   Solana 真实链上测试
========================================
Sponsor地址: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK
用户地址: 5yHFDH8SSHAwUKkTkzcP3vFjcjxMxWM5XtxJ4JmR4zpz (有20 USDC，无SOL)
网络: Solana Devnet
浏览器: https://explorer.solana.com/?cluster=devnet
测试场景: 用户转1 USDC，我们代付SOL gas费

=== 步骤1: 检查Sponsor余额 ===
   当前余额: 9.999950000 SOL
✅ Sponsor余额充足

=== 步骤1.5: 验证用户账户状态 ===
   用户SOL余额: 0.000000000 SOL
✅ 用户没有SOL，符合代付测试场景
   假设用户有20 USDC (devnet无法直接查询)

=== 步骤2: 构造USDC转账交易 ===
💰 用户想转1 USDC给别人，但没有SOL支付gas
⚠️  注意: 请确保用户地址 5yHFDH8SSHAwUKkTkzcP3vFjcjxMxWM5XtxJ4JmR4zpz 已经有USDC余额
   如果没有，请先转账USDC到该地址
   用户地址: 5yHFDH8SSHAwUKkTkzcP3vFjcjxMxWM5XtxJ4JmR4zpz
   目标地址: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK
   转账金额: 1.000000 USDC
   Fee Payer: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK (Sponsor代付gas)
   用户Token账户: HBRRrHEzFsffVP8GTGVrVuCpoWiQycHyMmDYSTeZQimF
   目标Token账户: CnQtgDZmcwpeKepc1UHFudP6hgZHhykrsQmti4WzVsbC
   场景: 用户转USDC，Sponsor代付交易gas费
✅ USDC转账交易构造成功

=== 步骤3: 获取Gas费用估算 ===
✅ Gas估算结果:
   需要支付: $0.010000 USDC
   原生Gas: 0.000005000 SOL
   SOL价格: $84.03
   收款地址: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK

=== 步骤4: 用户支付Gas费用 ===
💰 用户需要支付: $0.010000 USDC 作为gas费用
📍 收款地址: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK
💡 原理: 用户转USDC给我们作为gas费，我们代付这笔转账的SOL gas
   支付金额: 0.010000 USDC
   从: 5yHFDH8SSHAwUKkTkzcP3vFjcjxMxWM5XtxJ4JmR4zpz
   到: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK
🔄 执行用户支付gas费用的USDC转账...
✅ 用户支付完成，交易Hash: 5vGCzp4tphJQtxamwZ8t4AiZbzGAdpNUku3CWcLTyDhUggRq59GYB17MCkYiyB5h5pyaHPc7FBhot1Ahs6gp6Hwx
   我们代付了用户支付gas费用这笔转账的SOL gas

=== 步骤5: 执行用户原本的USDC转账 ===
   响应状态: 200 OK
   响应状态: 200 OK
🚀 USDC转账交易执行成功!
   交易Hash: 65jgVkoQAXoJhkQjfpt3VWaBLoLWqWoZHzfmWYNJ7TWHMhHZNryegmb3v4FcNMHKsy8oGLgotCjMbStbM9iVeReU
   浏览器查看: https://explorer.solana.com/tx/65jgVkoQAXoJhkQjfpt3VWaBLoLWqWoZHzfmWYNJ7TWHMhHZNryegmb3v4FcNMHKsy8oGLgotCjMbStbM9iVeReU?cluster=devnet
   ✅ 用户成功转账1 USDC，我们代付了SOL gas费

=== 步骤6: 等待交易确认 ===
   等待确认. ✅
✅ 交易已确认

========================================
   Solana USDC代付Gas测试完成!
   🎯 用户转账: 1 USDC
   💸 用户支付: $0.010000 USDC (gas费)
   ⛽ 我们代付: 2笔交易的SOL gas费
   📊 支付交易: 5vGCzp4tphJQtxamwZ8t4AiZbzGAdpNUku3CWcLTyDhUggRq59GYB17MCkYiyB5h5pyaHPc7FBhot1Ahs6gp6Hwx
   📊 转账交易: 65jgVkoQAXoJhkQjfpt3VWaBLoLWqWoZHzfmWYNJ7TWHMhHZNryegmb3v4FcNMHKsy8oGLgotCjMbStbM9iVeReU
   💡 完整流程: 用户转USDC给我们→我们代付→用户转USDC给别人→我们代付
========================================

清理进程...
🎉 测试完成!
edy@edydeMacBook-Pro-4 agent_wallet_x402 % 


===
 %  ./test_usdc_gas_sponsor_on_sol.sh
🚀 USDC Gas代付完整测试
========================================
清理旧进程...
编译组件...
✅ 编译成功
启动服务...
等待服务启动...
✅ 服务已启动

⚠️  重要提醒:
   请确保Sponsor地址有SOL: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK
   获取测试SOL: https://faucet.solana.com/

🎯 测试场景:
   1. 用户有20 USDC，没有SOL
   2. 用户想转1 USDC给别人
   3. 用户先转USDC给我们作为gas费 (我们代付这笔转账的SOL gas)
   4. 我们代付用户原本想要的USDC转账的SOL gas

========================================
   Solana 真实链上测试
========================================
Sponsor地址: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK
用户地址: 5yHFDH8SSHAwUKkTkzcP3vFjcjxMxWM5XtxJ4JmR4zpz (有20 USDC，无SOL)
网络: Solana Devnet
浏览器: https://explorer.solana.com/?cluster=devnet
测试场景: 用户转1 USDC，我们代付SOL gas费

=== 步骤1: 检查Sponsor余额 ===
   当前余额: 9.999910000 SOL
✅ Sponsor余额充足

=== 步骤1.5: 验证用户账户状态 ===
   用户SOL余额: 0.000000000 SOL
✅ 用户没有SOL，符合代付测试场景
   假设用户有20 USDC (devnet无法直接查询)

=== 步骤2: 构造USDC转账交易 ===
💰 用户想转1 USDC给别人，但没有SOL支付gas
⚠️  注意: 请确保用户地址 5yHFDH8SSHAwUKkTkzcP3vFjcjxMxWM5XtxJ4JmR4zpz 已经有USDC余额
   如果没有，请先转账USDC到该地址
   用户地址: 5yHFDH8SSHAwUKkTkzcP3vFjcjxMxWM5XtxJ4JmR4zpz
   目标地址: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK
   转账金额: 1.000000 USDC
   Fee Payer: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK (Sponsor代付gas)
   用户Token账户: HBRRrHEzFsffVP8GTGVrVuCpoWiQycHyMmDYSTeZQimF
   目标Token账户: CnQtgDZmcwpeKepc1UHFudP6hgZHhykrsQmti4WzVsbC
   场景: 用户转USDC，Sponsor代付交易gas费
✅ USDC转账交易构造成功

=== 步骤3: 获取Gas费用估算 ===
✅ Gas估算结果:
   需要支付: $0.010000 USDC
   原生Gas: 0.000005000 SOL
   SOL价格: $82.94
   收款地址: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK

=== 步骤4: 用户支付Gas费用 ===
💰 用户需要支付: $0.010000 USDC 作为gas费用
📍 收款地址: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK
💡 原理: 用户转USDC给我们作为gas费，我们代付这笔转账的SOL gas
   支付金额: 0.010000 USDC
   从: 5yHFDH8SSHAwUKkTkzcP3vFjcjxMxWM5XtxJ4JmR4zpz
   到: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK
🔄 执行用户支付gas费用的USDC转账...
✅ 用户支付完成，交易Hash: 4t7PnausQrzTnquVk8ThqTh99BSD4XhETAYHL8BBikPPJCRyqptnNrsyvf29io6V6Xi78bN8pkKiRu1h7YK6rSoR
   我们代付了用户支付gas费用这笔转账的SOL gas
⏳ 等待支付交易确认...

=== 步骤5: 执行用户原本的USDC转账 ===
   响应状态: 200 OK
   响应状态: 200 OK
🚀 USDC转账交易执行成功!
   交易Hash: 2ieXEnjJNsuuckU7MdvVJj9Hi7edRhuLEKbDJS7LdLCPQ57zTzvxA6iHGSjVL1zNBSe69ZVXK4hkPkMofUUCJd17
   浏览器查看: https://explorer.solana.com/tx/2ieXEnjJNsuuckU7MdvVJj9Hi7edRhuLEKbDJS7LdLCPQ57zTzvxA6iHGSjVL1zNBSe69ZVXK4hkPkMofUUCJd17?cluster=devnet
   ✅ 用户成功转账1 USDC，我们代付了SOL gas费

=== 步骤6: 等待交易确认 ===
   等待确认. ✅
✅ 交易已确认

========================================
   Solana USDC代付Gas测试完成!
   🎯 用户转账: 1 USDC
   💸 用户支付: $0.010000 USDC (gas费)
   ⛽ 我们代付: 2笔交易的SOL gas费
   📊 支付交易: 4t7PnausQrzTnquVk8ThqTh99BSD4XhETAYHL8BBikPPJCRyqptnNrsyvf29io6V6Xi78bN8pkKiRu1h7YK6rSoR
   📊 转账交易: 2ieXEnjJNsuuckU7MdvVJj9Hi7edRhuLEKbDJS7LdLCPQ57zTzvxA6iHGSjVL1zNBSe69ZVXK4hkPkMofUUCJd17
   💡 完整流程: 用户转USDC给我们→我们代付→用户转USDC给别人→我们代付
========================================

清理进程...
🎉 测试完成!
edy@edydeMacBook-Pro-4 agent_wallet_x402 % git status

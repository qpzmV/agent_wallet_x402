#!/bin/bash

echo "🚀 ETH USDC Gas代付完整测试"
echo "========================================"

# 清理旧进程
echo "清理旧进程..."
lsof -ti :8080,8081 | xargs kill -9 2>/dev/null || true
sleep 1

# 编译所有组件
echo "编译组件..."
go build -o execution-engine/main ./execution-engine/main.go && \
go build -o x402-server/main ./x402-server/main.go ./x402-server/config.go ./x402-server/gas_calculator.go && \
go build -o eth-gas-test ./cmd/eth-gas-test/main.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"

# 启动后台服务
echo "启动服务..."
./execution-engine/main > /tmp/execution-engine.log 2>&1 &
ENGINE_PID=$!
./x402-server/main > /tmp/x402-server.log 2>&1 &
X402_PID=$!

# 等待服务启动
echo "等待服务启动..."
sleep 3

# 检查服务状态
if ! curl -s http://localhost:8080/payment-info > /dev/null; then
    echo "❌ 服务启动失败"
    echo "--- execution-engine 日志 ---"
    tail -20 /tmp/execution-engine.log
    echo "--- x402-server 日志 ---"
    tail -20 /tmp/x402-server.log
    kill $ENGINE_PID $X402_PID 2>/dev/null || true
    exit 1
fi

echo "✅ 服务已启动"
echo ""
echo "⚠️  重要提醒:"
echo "   请确保Sponsor地址有ETH: ${SPONSOR_ADDR:-0x125a63a553f5494313565F3baa099DD73dA500Bc}"
echo "   获取测试ETH: https://faucet.sepolia.dev/"
echo ""
echo "   请确保用户地址有USDC: ${USER_ADDR:-0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e}"
echo "   查询账户: https://sepolia.etherscan.io/address/0x742d35Cc6634C0532925a3b8D4C9db96C4b5Da5e"
echo "   获取测试USDC: 使用Sepolia USDC水龙头或DEX"
echo ""
echo "🎯 测试场景:"
echo "   1. 用户有USDC，没有ETH"
echo "   2. 用户想转1 USDC给别人"
echo "   3. 用户先转USDC给我们作为gas费 (我们代付这笔转账的ETH gas)"
echo "   4. 第3通过x402验证后，我们再代付用户原本想要的USDC转账的ETH gas"
echo ""

# 运行完整测试
./eth-gas-test

EXIT_CODE=$?

# 清理
echo ""
echo "清理进程..."
kill $ENGINE_PID $X402_PID 2>/dev/null || true

if [ $EXIT_CODE -eq 0 ]; then
    echo "🎉 测试完成!"
else
    echo "❌ 测试失败 (exit code: $EXIT_CODE)"
    echo ""
    echo "调试提示:"
    echo "  1. 检查 execution-engine 日志: tail -50 /tmp/execution-engine.log"
    echo "  2. 检查 x402-server 日志: tail -50 /tmp/x402-server.log"
    echo "  3. 确保 Sponsor 有 ETH: https://sepolia.etherscan.io/address/$(grep EVMSponsorAddr common/config.go | head -1 | awk -F'"' '{print $2}')"
    echo "  4. 确保用户有 USDC: https://sepolia.etherscan.io/address/$(grep EVMUserAddr common/config.go | head -1 | awk -F'"' '{print $2}')"
    echo "  5. 获取测试币:"
    echo "     - ETH: https://faucet.sepolia.dev/"
    echo "     - USDC: 需要通过DEX或其他方式获取Sepolia测试网USDC"
    exit $EXIT_CODE
fi
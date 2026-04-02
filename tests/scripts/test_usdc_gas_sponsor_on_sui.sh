#!/bin/bash

echo "🚀 SUI USDC Gas代付完整测试"
echo "========================================"

# 清理旧进程
echo "清理旧进程..."
lsof -ti :8080 | xargs kill -9 2>/dev/null || true
sleep 1

# 编译所有组件
echo "编译组件..."
go build -o x402-server/main ./x402-server/ && \
go build -o sui-gas-test ./cmd/sui-gas-test/main.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"

# 启动后台服务
echo "启动服务..."
./x402-server/main > /tmp/x402-server.log 2>&1 &
X402_PID=$!

# 等待服务启动
echo "等待服务启动..."
sleep 3

# 检查服务状态
if ! curl -s http://localhost:8080/payment-info > /dev/null; then
    echo "❌ 服务启动失败"
    echo "--- x402-server 日志 ---"
    tail -20 /tmp/x402-server.log
    kill $X402_PID 2>/dev/null || true
    exit 1
fi

echo "✅ 服务已启动"
echo ""
echo "⚠️  重要提醒:"
echo "   请确保Sponsor地址有SUI: ${SPONSOR_ADDR:-0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc}"
echo "   获取测试SUI: https://faucet.testnet.sui.io/"
echo ""
echo "   请确保用户地址有USDC: ${USER_ADDR:-0x5f31097cd2bd9957c0de799c088de591ece9747302a49e072528409014ed24dc}"
echo "   查询账户: https://suiscan.xyz/testnet/account/0x5f31097cd2bd9957c0de799c088de591ece9747302a49e072528409014ed24dc"
echo ""
echo "🎯 测试场景:"
echo "   1. 用户有USDC，没有SUI"
echo "   2. 用户想转1 USDC给别人"
echo "   3. 用户先转USDC给我们作为gas费 (我们代付这笔转账的SUI gas)"
echo "   4. 第3通过x402验证后，我们再代付用户原本想要的USDC转账的SUI gas"
echo ""

# 运行完整测试
./sui-gas-test

EXIT_CODE=$?

# 清理
echo ""
echo "清理进程..."
kill $X402_PID 2>/dev/null || true

if [ $EXIT_CODE -eq 0 ]; then
    echo "🎉 测试完成!"
else
    echo "❌ 测试失败 (exit code: $EXIT_CODE)"
    echo ""
    echo "调试提示:"
    echo "  1. 检查 x402-server 日志: tail -50 /tmp/x402-server.log"
    echo "  2. 确保 Sponsor 有 SUI: https://suiscan.xyz/testnet/account/$(grep SuiSponsorAddr common/config.go | head -1 | awk -F'"' '{print $2}')"
    exit $EXIT_CODE
fi

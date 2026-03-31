#!/bin/bash

echo "🚀 测试Solana支付验证修复"
echo "========================================"

# 清理旧进程
echo "清理旧进程..."
lsof -ti :8080,8081 | xargs kill -9 2>/dev/null || true
sleep 1

# 编译组件
echo "编译组件..."
go build -o execution-engine/main ./execution-engine/main.go && \
go build -o x402-server/main ./x402-server/main.go ./x402-server/config.go ./x402-server/gas_calculator.go && \
go build -o sol-gas-test ./cmd/sol-gas-test/main.go

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
echo "🎯 测试修复后的Solana支付验证"
echo ""

# 运行测试
./sol-gas-test

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
    echo "  3. 确保 Sponsor 有 SOL: https://faucet.solana.com/"
    echo "  4. 确保用户有 USDC"
    exit $EXIT_CODE
fi
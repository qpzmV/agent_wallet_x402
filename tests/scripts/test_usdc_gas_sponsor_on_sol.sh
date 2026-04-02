#!/bin/bash

echo "🚀 USDC Gas代付完整测试"
echo "========================================"

# 清理旧进程
echo "清理旧进程..."
lsof -ti :8080 | xargs kill -9 2>/dev/null || true
sleep 1

# 编译所有组件
echo "编译组件..."
go build -o x402-server/main ./x402-server/ && \
go build -o sol-gas-test ./cmd/sol-gas-test/main.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"

# 启动后台服务
echo "启动服务..."
./x402-server/main > /dev/null 2>&1 &
X402_PID=$!

# 等待服务启动
echo "等待服务启动..."
sleep 3

# 检查服务状态
if ! curl -s http://localhost:8080/payment-info > /dev/null; then
    echo "❌ 服务启动失败"
    kill $X402_PID 2>/dev/null || true
    exit 1
fi

echo "✅ 服务已启动"
echo ""
echo "⚠️  重要提醒:"
echo "   请确保Sponsor地址有SOL: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK"
echo "   获取测试SOL: https://faucet.solana.com/"
echo ""
echo "🎯 测试场景:"
echo "   1. 用户有20 USDC，没有SOL"
echo "   2. 用户想转1 USDC给别人"
echo "   3. 用户先转USDC给我们作为gas费 (我们代付这笔转账的SOL gas)"
echo "   4. 我们代付用户原本想要的USDC转账的SOL gas"
echo ""

# 运行完整测试
./sol-gas-test

# 清理
echo ""
echo "清理进程..."
kill $X402_PID 2>/dev/null || true

echo "🎉 测试完成!"
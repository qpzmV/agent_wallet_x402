#!/bin/bash

echo "🚀 Solana x402 真实链上测试 - 一键运行"
echo "========================================"

# 清理旧进程
echo "清理旧进程..."
lsof -ti :8080,8081 | xargs kill -9 2>/dev/null || true
sleep 1

# 编译所有组件
echo "编译组件..."
go build -o execution-engine/main ./execution-engine/main.go && \
go build -o x402-server/main ./x402-server/main.go ./x402-server/config.go ./x402-server/gas_calculator.go && \
go build -o solana-real-test ./cmd/real-test/main.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"

# 启动后台服务
echo "启动服务..."
./execution-engine/main > /dev/null 2>&1 &
ENGINE_PID=$!
./x402-server/main > /dev/null 2>&1 &
X402_PID=$!

# 等待服务启动
echo "等待服务启动..."
sleep 3

# 检查服务状态
if ! curl -s http://localhost:8080/payment-info > /dev/null; then
    echo "❌ 服务启动失败"
    kill $ENGINE_PID $X402_PID 2>/dev/null || true
    exit 1
fi

echo "✅ 服务已启动"
echo ""
echo "⚠️  重要提醒:"
echo "   请确保Sponsor地址有SOL: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK"
echo "   获取测试SOL: https://faucet.solana.com/"
echo ""

# 运行真实测试
./solana-real-test

# 清理
echo ""
echo "清理进程..."
kill $ENGINE_PID $X402_PID 2>/dev/null || true

echo "🎉 测试完成!"
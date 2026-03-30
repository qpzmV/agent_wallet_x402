#!/bin/bash

echo "=========================================="
echo "   Solana 真实链上测试 - x402 Gas 代付"
echo "=========================================="

# 清理现有进程
echo "清理现有进程..."
lsof -ti :8080,8081 | xargs kill -9 2>/dev/null || true
sleep 1

# 构建程序
echo "构建程序..."
go build -o execution-engine/main ./execution-engine/main.go
go build -o x402-server/main ./x402-server/main.go ./x402-server/config.go ./x402-server/gas_calculator.go
go build -o solana-real-test ./cmd/real-test/main.go

# 启动服务
echo "启动执行引擎..."
./execution-engine/main > execution-engine.log 2>&1 &
ENGINE_PID=$!

echo "启动x402服务器..."
./x402-server/main > x402-server.log 2>&1 &
X402_PID=$!

echo "等待服务启动..."
sleep 5

# 检查服务是否正常启动
if ! curl -s http://localhost:8080/payment-info > /dev/null; then
    echo "❌ x402服务器启动失败"
    echo "=== x402服务器日志 ==="
    cat x402-server.log
    kill $ENGINE_PID $X402_PID 2>/dev/null || true
    exit 1
fi

if ! curl -s http://localhost:8081/execute -X POST -H "Content-Type: application/json" -d '{}' > /dev/null; then
    echo "❌ 执行引擎启动失败"
    echo "=== 执行引擎日志 ==="
    cat execution-engine.log
    kill $ENGINE_PID $X402_PID 2>/dev/null || true
    exit 1
fi

echo "✅ 服务启动成功"

# 显示重要信息
echo -e "\n=== 重要提醒 ==="
echo "请确保Solana Sponsor地址有足够的SOL:"
echo "地址: CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK"
echo "获取测试SOL: https://faucet.solana.com/"
echo ""
echo "本测试将执行真实的Solana链上交易!"
echo "交易将在Solana Devnet上执行，可在区块链浏览器查看。"

echo -e "\n按回车继续真实链上测试..."
read

# 运行真实测试
echo -e "\n=== 开始Solana真实链上测试 ==="
./solana-real-test

# 保存日志
echo -e "\n=== 保存服务日志 ==="
echo "执行引擎日志已保存到: execution-engine.log"
echo "x402服务器日志已保存到: x402-server.log"

# 清理
echo -e "\n清理进程..."
kill $ENGINE_PID $X402_PID 2>/dev/null || true

echo -e "\n=========================================="
echo "   Solana真实链上测试完成!"
echo "=========================================="
echo "请在Solana浏览器中查看交易详情:"
echo "https://explorer.solana.com/?cluster=devnet"
echo ""
echo "搜索交易hash或地址来查看具体信息。"
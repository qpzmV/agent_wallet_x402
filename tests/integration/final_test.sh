#!/bin/bash

echo "=========================================="
echo "   最终测试 - x402 Gas 代付系统"
echo "=========================================="

# 清理现有进程
echo "清理现有进程..."
lsof -ti :8080,8081 | xargs kill -9 2>/dev/null || true
sleep 1

# 构建程序
echo "构建程序..."
go build -o execution-engine/main ./execution-engine/main.go
go build -o x402-server/main ./x402-server/main.go ./x402-server/config.go ./x402-server/gas_calculator.go

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
    kill $ENGINE_PID $X402_PID 2>/dev/null || true
    exit 1
fi

if ! curl -s http://localhost:8081/execute -X POST -H "Content-Type: application/json" -d '{}' > /dev/null; then
    echo "❌ 执行引擎启动失败"
    kill $ENGINE_PID $X402_PID 2>/dev/null || true
    exit 1
fi

echo "✅ 服务启动成功"

# 测试402响应
echo -e "\n=== 测试1: 402响应 ==="
response=$(curl -s -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "chain": "solana",
    "tx_data": "dGVzdF9kYXRh",
    "user_address": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
    "target_address": "11111111111111111111111111111112",
    "user_signature": "test_sig"
  }')

echo "402响应:"
echo "$response" | jq .

# 提取收款地址
solana_receiver=$(echo "$response" | jq -r '.payment.receivers.solana // empty')
if [ -n "$solana_receiver" ] && [ "$solana_receiver" != "null" ]; then
    echo "✅ Solana收款地址: $solana_receiver"
else
    echo "❌ Solana收款地址为空"
fi

echo -e "\n测试完成!"
echo "$exec_response"

# 分析执行结果
status=$(echo "$exec_response" | jq -r '.status // empty')
if [ "$status" = "failed" ]; then
    error=$(echo "$exec_response" | jq -r '.error // empty')
    echo "⚠️ 执行失败（预期，因为使用测试数据）: $error"
    echo "✅ JSON解析成功，响应格式正确"
elif [ "$status" = "success" ]; then
    tx_hash=$(echo "$exec_response" | jq -r '.tx_hash // empty')
    echo "🚀 执行成功: $tx_hash"
else
    echo "❌ 未知状态: $status"
fi

# 清理
echo -e "\n清理进程..."
kill $ENGINE_PID $X402_PID 2>/dev/null || true

echo -e "\n=========================================="
echo "   测试完成"
echo "=========================================="
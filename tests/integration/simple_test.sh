#!/bin/bash

echo "停止现有进程..."
lsof -ti :8080,8081 | xargs kill -9 2>/dev/null || true

echo "构建程序..."
go build -o execution-engine/main ./execution-engine/main.go
go build -o x402-server/main ./x402-server/main.go ./x402-server/config.go ./x402-server/gas_calculator.go

echo "启动服务..."
./execution-engine/main &
ENGINE_PID=$!
./x402-server/main &
X402_PID=$!

echo "等待服务启动..."
sleep 3

echo "测试402响应..."
curl -s -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "chain": "solana",
    "tx_data": "test_data",
    "user_address": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
    "target_address": "test_target",
    "user_signature": "test_sig"
  }' | jq .

echo -e "\n测试支付执行..."
curl -s -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -H "X-402-Payment: paid-123" \
  -d '{
    "chain": "solana",
    "tx_data": "test_data",
    "user_address": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
    "target_address": "test_target",
    "user_signature": "test_sig"
  }'

echo -e "\n\n清理进程..."
kill $ENGINE_PID $X402_PID 2>/dev/null || true
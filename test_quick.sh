#!/bin/bash

echo "========================================"
echo "   快速测试x402修复"
echo "========================================"

# 测试402响应
echo "测试402响应..."
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "chain": "solana",
    "tx_data": "test_data",
    "user_address": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
    "target_address": "test_target",
    "user_signature": "test_sig"
  }' | jq .

echo -e "\n\n测试支付执行..."
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -H "X-402-Payment: paid-123" \
  -d '{
    "chain": "solana",
    "tx_data": "test_data",
    "user_address": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
    "target_address": "test_target",
    "user_signature": "test_sig"
  }'

echo -e "\n\n测试完成!"
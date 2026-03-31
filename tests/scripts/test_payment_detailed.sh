#!/bin/bash

echo "🚀 详细测试Solana支付验证"
echo "========================================"

# 清理旧进程
echo "清理旧进程..."
lsof -ti :8080 | xargs kill -9 2>/dev/null || true
sleep 1

# 编译x402-server
echo "编译x402-server..."
go build -o x402-server/main ./x402-server/main.go ./x402-server/config.go ./x402-server/gas_calculator.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"

# 启动x402-server
echo "启动x402-server..."
./x402-server/main > /tmp/x402-server.log 2>&1 &
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

# 测试1: 使用无效的支付凭证
echo "🔍 测试1: 使用无效的支付凭证"
cat > /tmp/test_request.json << EOF
{
  "chain": "solana",
  "txData": "simple_test",
  "userSignature": "test_signature",
  "userAddress": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
  "targetAddress": "CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK"
}
EOF

RESPONSE1=$(curl -s -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -H "X-Payment-Chain: solana" \
  -H "X-Payment-Proof: invalid_signature" \
  -d @/tmp/test_request.json)

echo "响应1: $RESPONSE1"
echo ""

# 测试2: 使用真实的交易签名
echo "🔍 测试2: 使用真实的交易签名"
RESPONSE2=$(curl -s -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -H "X-Payment-Chain: solana" \
  -H "X-Payment-Proof: p5gg2WzvJsjLkwh4byMv4GRMnWH4L35JjvTvytftECqeVJG5pqgy2bw8jx4oiW4kWFie39CbHGK5QgH9EpTMXZu" \
  -d @/tmp/test_request.json)

echo "响应2: $RESPONSE2"
echo ""

# 测试3: 使用模拟支付凭证
echo "🔍 测试3: 使用模拟支付凭证"
RESPONSE3=$(curl -s -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -H "X-Payment-Chain: solana" \
  -H "X-Payment-Proof: paid-123" \
  -d @/tmp/test_request.json)

echo "响应3: $RESPONSE3"
echo ""

# 等待一下让日志输出
sleep 2

# 清理
echo "清理进程..."
kill $X402_PID 2>/dev/null || true

echo ""
echo "详细日志:"
echo "--- x402-server 日志 ---"
tail -50 /tmp/x402-server.log

exit 0
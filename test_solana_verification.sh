#!/bin/bash

echo "🚀 测试Solana支付验证"
echo "========================================"

# 清理旧进程
echo "清理旧进程..."
lsof -ti :8080,8081 | xargs kill -9 2>/dev/null || true
sleep 1

# 编译组件
echo "编译组件..."
go build -o execution-engine/main ./execution-engine/main.go && \
go build -o x402-server/main ./x402-server/main.go ./x402-server/config.go ./x402-server/gas_calculator.go

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
    kill $ENGINE_PID $X402_PID 2>/dev/null || true
    exit 1
fi

echo "✅ 服务已启动"
echo ""

# 测试支付验证API
echo "🔍 测试Solana支付验证API"
echo "使用交易: p5gg2WzvJsjLkwh4byMv4GRMnWH4L35JjvTvytftECqeVJG5pqgy2bw8jx4oiW4kWFie39CbHGK5QgH9EpTMXZu"

# 构造测试请求
cat > /tmp/test_request.json << EOF
{
  "chain": "solana",
  "txData": "dummy_tx_data",
  "userSignature": "dummy_signature",
  "userAddress": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
  "targetAddress": "CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK"
}
EOF

# 发送请求
echo "发送执行请求..."
RESPONSE=$(curl -s -X POST http://localhost:8081/execute \
  -H "Content-Type: application/json" \
  -H "X-Payment-Chain: solana" \
  -H "X-Payment-Proof: p5gg2WzvJsjLkwh4byMv4GRMnWH4L35JjvTvytftECqeVJG5pqgy2bw8jx4oiW4kWFie39CbHGK5QgH9EpTMXZu" \
  -d @/tmp/test_request.json)

echo "响应: $RESPONSE"

# 检查响应
if echo "$RESPONSE" | grep -q '"status":"success"'; then
    echo "✅ Solana支付验证成功!"
    EXIT_CODE=0
elif echo "$RESPONSE" | grep -q 'Payment required'; then
    echo "❌ 支付验证失败"
    echo "详细错误: $(echo "$RESPONSE" | jq -r '.message // .error // "未知错误"' 2>/dev/null || echo "$RESPONSE")"
    EXIT_CODE=1
else
    echo "❓ 未知响应格式"
    EXIT_CODE=1
fi

# 清理
echo ""
echo "清理进程..."
kill $ENGINE_PID $X402_PID 2>/dev/null || true

if [ $EXIT_CODE -eq 0 ]; then
    echo "🎉 测试完成!"
else
    echo "❌ 测试失败"
    echo ""
    echo "调试信息:"
    echo "--- execution-engine 日志 ---"
    tail -20 /tmp/execution-engine.log
    echo "--- x402-server 日志 ---"
    tail -20 /tmp/x402-server.log
fi

exit $EXIT_CODE
#!/bin/bash

echo "🚀 测试Solana支付验证 (仅支付部分)"
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

# 测试支付验证API - 发送一个简单的请求来触发支付验证
echo "🔍 测试Solana支付验证"
echo "使用交易: p5gg2WzvJsjLkwh4byMv4GRMnWH4L35JjvTvytftECqeVJG5pqgy2bw8jx4oiW4kWFie39CbHGK5QgH9EpTMXZu"

# 构造测试请求 - 使用一个简单的请求
cat > /tmp/test_request.json << EOF
{
  "chain": "solana",
  "txData": "simple_test",
  "userSignature": "test_signature",
  "userAddress": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
  "targetAddress": "CW7ovTXjw47DnRXXR6zCNyWRfHagj2ryFmaJsq76CZCK"
}
EOF

# 发送请求到x402-server (不需要execution-engine)
echo "发送支付验证请求..."
RESPONSE=$(curl -s -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -H "X-Payment-Chain: solana" \
  -H "X-Payment-Proof: p5gg2WzvJsjLkwh4byMv4GRMnWH4L35JjvTvytftECqeVJG5pqgy2bw8jx4oiW4kWFie39CbHGK5QgH9EpTMXZu" \
  -d @/tmp/test_request.json)

echo "响应: $RESPONSE"

# 检查响应 - 我们期望看到支付验证的错误信息，而不是执行引擎的错误
if echo "$RESPONSE" | grep -q 'Payment required'; then
    # 检查是否是我们期望的Solana支付验证错误
    if echo "$RESPONSE" | grep -q 'commitment\|交易状态\|历史交易'; then
        echo "✅ Solana支付验证逻辑正在工作 (检测到改进的错误信息)"
        EXIT_CODE=0
    else
        echo "❌ 支付验证失败，但不是预期的错误"
        echo "详细错误: $(echo "$RESPONSE" | jq -r '.message // .error // "未知错误"' 2>/dev/null || echo "$RESPONSE")"
        EXIT_CODE=1
    fi
elif echo "$RESPONSE" | grep -q 'Bad Gateway'; then
    echo "✅ 支付验证通过，但执行引擎未启动 (这是预期的)"
    EXIT_CODE=0
else
    echo "❓ 未知响应格式"
    EXIT_CODE=1
fi

# 清理
echo ""
echo "清理进程..."
kill $X402_PID 2>/dev/null || true

if [ $EXIT_CODE -eq 0 ]; then
    echo "🎉 支付验证测试完成!"
else
    echo "❌ 测试失败"
    echo ""
    echo "调试信息:"
    echo "--- x402-server 日志 ---"
    tail -30 /tmp/x402-server.log
fi

exit $EXIT_CODE
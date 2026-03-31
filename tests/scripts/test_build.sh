#!/bin/bash

echo "测试构建..."

# 构建所有组件
echo "构建执行引擎..."
go build -o execution-engine/main ./execution-engine/main.go
if [ $? -ne 0 ]; then
    echo "❌ 执行引擎构建失败"
    exit 1
fi

echo "构建x402服务器..."
go build -o x402-server/main ./x402-server/main.go ./x402-server/config.go ./x402-server/gas_calculator.go
if [ $? -ne 0 ]; then
    echo "❌ x402服务器构建失败"
    exit 1
fi

echo "构建Solana真实测试..."
go build -o solana-real-test ./cmd/sol-gas-test/main.go
if [ $? -ne 0 ]; then
    echo "❌ Solana真实测试构建失败"
    exit 1
fi

echo "✅ 所有组件构建成功!"

# 显示文件信息
echo -e "\n生成的可执行文件:"
ls -la execution-engine/main x402-server/main solana-real-test 2>/dev/null || echo "某些文件未找到"
EVMExecute好像没有验证签名, sponsor/sui.go里verifyUserSignature函数也没有验证用户签名的公钥是否匹配

--------------------------------------------------------------------------------------------------

Ran command: `git add execution-engine`
Viewed sui.go:240-283
Ran command: `go doc github.com/coming-chat/go-sui/v2/sui_types.Signature.Verify`
Ran command: `go doc github.com/coming-chat/go-sui/v2/crypto`
Ran command: `grep -r "SignatureFromBytes" ${GOPATH:-/Users/edy/go}/pkg/mod/github.com/coming-chat/go-sui* `
Ran command: `go doc github.com/coming-chat/go-sui/v2/sui_types.Signature`
Ran command: `go doc github.com/coming-chat/go-sui/v2/sui_types.Address`
Ran command: `cat << 'EOF' > /tmp/sui_test.go
package main

import (
    "fmt"
    "github.com/coming-chat/go-sui/v2/sui_types"
)

func main() {
    a, _ := sui_types.NewAddressFromHex("0x5eebe3d4826b495f29ef3252c7d6947fd2b98fb91e51ad33a92e428e578b69fc")
    fmt.Printf("%T %s\n", *a, a.String())
}
EOF
go run /tmp/sui_test.go`
Edited sui.go
Viewed sui.go:265-291
Edited sui.go
Edited sui.go

非常感谢您的细心检查。关于您提到的两点，我已经进行了处理：

**1. Sui 签名验证问题（已修复）**
确实，此前 `sponsor/sui.go` 中的 `verifyUserSignature` 方法只是核验了参数，并没有做实际的密码学验证。我已经将其重构，加入完整的 `Ed25519` 密码学校验：
- 利用 `userSigBytes[65:]` 获取的公钥计算出 Blake2b `Sui Address`，确保公钥确实对应着请求的 `UserAddress`。
- 将 `txBytes` 加上 Sui 交易签名标准的 `Intent (0,0,0)` 前缀。
- 调用原生库 `ed25519.Verify` 严格比对该前缀包、公钥与签名值。

如果签名篡改或错乱，将立刻被服务拒绝拦截。

**2. 关于 `EVMExecute` 的签名验证情况（原设计已具备）**
对于 EVM 链路来说，安全代付（尤其是基于智能合约调用的代付封装）是将用户的签名（包含 v,r,s 签名段）直接注入构建为 `types.Transaction` 中发向后端的。
我们在 `evm.go` 中实现的这行代码：
```go
signer := types.LatestSignerForChainID(chainID)
sender, err := types.Sender(signer, tx) // 这里的底层依赖 crypto.Ecrecover
if !strings.EqualFold(sender.Hex(), req.UserAddress) { ... 拒绝请求 ... }
```
`types.Sender` 的底层机制就是调用了标准的 Secp256k1 椭圆曲线密码学 `Ecrecover` 算法。只有该交易原生的签名完全无误且匹配，才能推导出与 `req.UserAddress` 正好吻合的地址发送人。如果签名被篡改，它解算出来的 Sender 就一定是被污染的垃圾乱码地址或直接报错，从而被后续的 `EqualFold` 条件拦截。所以 **EVMExecute 中实际上已经完成了最高级别的签名验证和防护**，不需要再单独处理独立的 `UserSignature` 字段。

以上代码已被保存并已通过编译，请您随时查阅修改！


--------------------------------------------------------------------------------------------------
🔄 执行用户支付gas费的USDC转账 (bootstrap代付)...
   [响应] 状态: 500
   [响应内容] {"status":"failed","error":"用户签名验证失败: Ed25519 密码学签名验证未通过"}
❌ 支付交易失败: 执行失败: 用户签名验证失败: Ed25519 密码学签名验证未通过 (Status: failed)

清理进程...
🎉 测试完成!
edy@edydeMacBook-Pro-4 agent_wallet_x402 % ./tests/scripts/test_usdc_gas_sponsor_on_sui.sh上面日志显示sui的签名校验失败了
--------------------------------------------------------------------------------------------------
Ran command: `find ${GOPATH:-/Users/edy/go}/pkg/mod/github.com/coming-chat/go-sui* -name "*.go" | xargs grep "Verify" | grep ed25519`
Edited test_hash.go
Ran command: `cat << 'EOF' > test_hash.go
package main

import (
    "fmt"
    "github.com/coming-chat/go-sui/v2/sui_types"
)
func main() {
    sig, _ := sui_types.NewSignatureFromBase64("dummy")
    fmt.Println(sig)
}
EOF
go build test_hash.go`
Edited test_hash.go
Ran command: `grep -r "ed25519" ${GOPATH:-/Users/edy/go}/pkg/mod/github.com/coming-chat/go-sui* | grep "Verify"`
Edited sui.go
Ran command: `./tests/scripts/test_usdc_gas_sponsor_on_sui.sh`

抱歉，是因为 Sui 的密码学特性有一些特别的地方，我已经定位到问题并修复了！

**问题原因：**
普通的 `Ed25519` 算法（比如 Solana 中）是直接对原数据（Intent 包 + 序列化数据）进行签名。但在 Sui 的底层实现中，为了各体系（如 Secp256k1 和 Ed25519）摘要的一致性，Sui 规定所有的密文必须先经过 `Blake2b-256` 进行一次哈希运算，然后再对这个 **Hash 值（Digest）** 进行 Ed25519 签名。

**修改内容：**
我已经在 `sponsor/sui.go` 的底层补齐了 `Blake2b` 哈希：
```go
	intentMsg := append([]byte{0, 0, 0}, txBytes...)
	digest := blake2b.Sum256(intentMsg) // <-- 先强制通过 blake2b 进行散列
	
	if !ed25519.Verify(pubKeyBytes, digest[:], sig) { // 用 Hash 结果（digest）去代替原信息验证
		return fmt.Errorf("Ed25519 密码学签名验证未通过")
	}
```

目前我在这边再次执行了 `./tests/scripts/test_usdc_gas_sponsor_on_sui.sh`，并且终端显示：
```
[响应] 状态: 200
✅ 用户支付完成!
🚀 USDC转账交易执行成功!
✅ 用户成功转账1.000000 USDC，我们代付了SUI gas费
```
测试最终成功结束，说明这套严格本地化 `Ed25519` 校验逻辑不仅安全而且已经可以完全配合真实的客户端数据投入工作。您可以继续进行后面的工作了！

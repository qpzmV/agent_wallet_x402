package sponsor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"agent-wallet-gas-sponsor/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func EVMExecute(req common.ExecuteRequest) (common.ExecuteResponse, error) {
	client, err := ethclient.Dial(common.EVMSepoliaRPC)
	if err != nil {
		return common.ExecuteResponse{}, fmt.Errorf("无法连接到 Sepolia: %v", err)
	}

	// 1. 解析用户提交的交易数据 (假设是已签名的 hex)
	rawTx, err := hexutil.Decode(req.TxData)
	if err != nil {
		return common.ExecuteResponse{}, fmt.Errorf("无效的交易数据: %v", err)
	}

	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(rawTx); err != nil {
		return common.ExecuteResponse{}, fmt.Errorf("解析交易失败: %v", err)
	}

	// 2. 这里实现 Gas 代付的一种方式：由 Sponsor 提交交易并支付 Gas
	// 注意：在 EVM 中，只有 Meta-Transaction 或 Account Abstraction 才能让第三方付 Gas。
	// 这里演示由交易引擎直接广播已签名的交易，或者演示向用户地址转入 Gas 币。
	
	// 演示：广播交易 (假设用户已经构造好了，只是我们需要协助提交)
	err = client.SendTransaction(context.Background(), tx)
	if err != nil {
		// 如果是因为没钱付 Gas 而失败，我们可以尝试给用户转一点 Gas
		fmt.Printf("广播失败，正在尝试为用户 %s 转入 Gas 币...\n", req.UserAddress)
		if err := transferGasETH(client, req.UserAddress); err != nil {
			return common.ExecuteResponse{}, fmt.Errorf("Gas 充值失败: %v", err)
		}
		
		// 等待gas转账完成
		fmt.Printf("等待 Gas 转账确认...\n")
		time.Sleep(10 * time.Second) // 等待10秒让转账被挖矿
		
		// 再次尝试
		err = client.SendTransaction(context.Background(), tx)
		if err != nil {
			return common.ExecuteResponse{}, fmt.Errorf("再次广播失败: %v", err)
		}
	}

	return common.ExecuteResponse{
		TxHash: tx.Hash().Hex(),
		Status: "success",
	}, nil
}

func transferGasETH(client *ethclient.Client, toAddress string) error {
	// 实现简单的转账逻辑，从 Sponsor 账户给用户转 0.01 ETH
	privateKey, _ := crypto.ToECDSA(hexutil.MustDecode(common.EVMSponsorPK))
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKey)

	nonce, _ := client.PendingNonceAt(context.Background(), fromAddress)
	value := big.NewInt(10000000000000000) // 0.01 ETH
	gasLimit := uint64(21000)
	gasPrice, _ := client.SuggestGasPrice(context.Background())

	tx := types.NewTransaction(nonce, ethcommon.HexToAddress(toAddress), value, gasLimit, gasPrice, nil)
	chainID, _ := client.NetworkID(context.Background())
	signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)

	return client.SendTransaction(context.Background(), signedTx)
}

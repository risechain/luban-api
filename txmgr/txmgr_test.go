package txmgr

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/risechain/luban-api/client"
)

func TestTxmgr(t *testing.T) {
	l := testlog.Logger(t, log.LevelCrit)
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f292")
	chainId := big.NewInt(7028081469)
	addr := crypto.PubkeyToAddress(key.PublicKey)
	cfg := &txmgr.Config{
		From:           addr,
		NetworkTimeout: time.Second,
		Signer: func(ctx context.Context, from common.Address, tx *types.Transaction) (*types.Transaction, error) {
			signer := types.LatestSignerForChainID(chainId)
			return types.SignTx(tx, signer, key)
		},
	}
	cl, err := rpc.Dial("https://rpc.bootnode-1.taiyi-devnet-0.preconfs.org")
	if err != nil {
		panic(err)
	}
	rpc := ethclient.NewClient(cl)

	preconfer, err := client.NewClient("https://gateway.taiyi-devnet-0.preconfs.org", key)
	if err != nil {
		panic(err)
	}

	blobs := []*eth.Blob{&eth.Blob{}}
	txmanager := NewPreconfTxMgr(l, rpc, cfg, preconfer, "https://bn.bootnode-1.taiyi-devnet-0.preconfs.org")

	cand := txmgr.TxCandidate{Blobs: blobs, To: &addr}
	_, err = txmanager.Send(context.Background(), cand)
	if err != nil {
		panic(err)
	}
}

package types

import (
	"math/big"
	"testing"

	"github.com/google/uuid"
	u256 "github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestSubmitTxDigest(t *testing.T) {
	id, _ := uuid.Parse("a1a2a3a4-b1b2-c1c2-d1d2-d3d4d5d6d7d8")

	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	txdata := &types.LegacyTx{
		Nonce:    1,
		Gas:      1,
		GasPrice: big.NewInt(2),
		Data:     []byte("abcdef"),
	}
	tx, _ := types.SignNewTx(key, types.NewEIP2930Signer(big.NewInt(2)), txdata)

	want := common.HexToHash("0xaad7d05bfbfe465c0d439bc8db6fb1857971bbeac70f66f40fe4759ccf448b60")
	if have := SubmitTxDigest(id, tx); have != want {
		t.Fatalf("Wrong submit tx digest. Have %v, want %v", have, want)
	}
}

func TestReserveBlockSpaceDigest(t *testing.T) {
	req := ReserveBlockSpaceRequest{
		BlobCount:  10,
		Deposit:    hexutil.U256(*u256.NewInt(13)),
		GasLimit:   3,
		TargetSlot: 23,
		Tip:        hexutil.U256(*u256.NewInt(7)),
	}

	want := common.HexToHash("0x287d63b3d67b5d357a458cae88c9d7ec6b1a6882e74ddb13f8c5b65f0cbb5aac")
	if have := req.Digest(); have != want {
		t.Fatalf("Wrong reserve blockspace digest. Have %v, want %v", have, want)
	}
}

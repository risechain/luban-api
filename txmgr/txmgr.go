package txmgr

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	eth "github.com/ethereum/go-ethereum/core/types"

	"github.com/risechain/luban-api/client"
	"github.com/risechain/luban-api/types"
)

type PreconfClient interface {
	GetEpochInfo(ctx context.Context) ([]types.SlotInfo, error)
	GetPreconfFee(ctx context.Context, slot uint64) (uint64, error)

	ReserveBlockspace(
		ctx context.Context,
		req types.ReserveBlockSpaceRequest,
	) (*types.ReserveBlockSpaceResponse, error)

	SubmitTransaction(ctx context.Context, reqId uuid.UUID, tx types.Transaction) error
}

type PreconfTxMgr struct {
	txmgr.TxManager
	client client.PreconfClient
}

func NewPreconfTxMgr(inner txmgr.TxManager, client client.PreconfClient) PreconfTxMgr {
	return PreconfTxMgr{inner, client}
}

func (m *PreconfTxMgr) Send(ctx context.Context, candidate TxCandidate) (*eth.Receipt, error) {
	// TODO:
	return &eth.Receipt{}, nil
}

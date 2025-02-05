package txmgr

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	luban "github.com/risechain/luban-api/types"
)

type PreconfClient interface {
	GetEpochInfo(ctx context.Context) ([]luban.SlotInfo, error)
	GetPreconfFee(ctx context.Context, slot uint64) (uint64, error)

	ReserveBlockspace(
		ctx context.Context,
		req luban.ReserveBlockSpaceRequest,
	) (*luban.ReserveBlockSpaceResponse, error)

	SubmitTransaction(ctx context.Context, reqId uuid.UUID, tx types.Transaction) error
}

type PreconfTxMgr struct {
	backend txmgr.ETHBackend
	client  PreconfClient

	l   log.Logger
	cfg *txmgr.Config

	nonce     *uint64
	nonceLock sync.RWMutex
}

func NewPreconfTxMgr(l log.Logger, backend txmgr.ETHBackend, cfg *txmgr.Config, client PreconfClient) *PreconfTxMgr {
	return &PreconfTxMgr{
		backend: backend,
		client:  client,
		l:       l,
		cfg:     cfg,
	}
}

func (m *PreconfTxMgr) Send(ctx context.Context, candidate txmgr.TxCandidate) (*types.Receipt, error) {
	tx, err := m.prepare(ctx, candidate)
	if err != nil {
		return nil, fmt.Errorf("preparing tx failed: %w", err)
	}

	nBlobs := uint32(len(candidate.Blobs))

	for {
		slots, err := m.client.GetEpochInfo(ctx)
		// TODO: retry here or on a preconf client side?
		if err != nil {
			return nil, fmt.Errorf("geting epoch info for preconf failed: %w", err)
		}

		slot := uint64(0)
		for _, s := range slots {
			if s.BlobsAvailable < nBlobs {
				continue
			}
			if s.GasAvailable < candidate.GasLimit {
				continue
			}
			slot = s.Slot
			break
		}
		if slot == 0 {
			// XXX: Figure out if we should wait till next slot or it should be fatal
			return nil, errors.New("No slots available for transaction")
		}

		resp, err := m.client.ReserveBlockspace(ctx, luban.ReserveBlockSpaceRequest{
			Id:         uuid.New(),
			TxHash:     tx.Hash(),
			BlobCount:  nBlobs,
			GasLimit:   candidate.GasLimit,
			TargetSlot: slot,
		})
		if err != nil {
			m.l.Warn("Reserving blockspace for tx failed. Someone probably took our slot. Retrying...", "err", err)
			continue
		}

		err = m.client.SubmitTransaction(ctx, resp.RequestId, *tx)
		if err != nil {
			m.l.Error("Sending preconfed tx failed. Slashing preconfer...", "err", err)
			// TODO: slash preconfer
			continue
		}
		break
	}

	return &types.Receipt{}, nil
}

// Copied from op-service/txmgr/txmgr.go

// prepare prepares the transaction for sending.
func (m *PreconfTxMgr) prepare(ctx context.Context, candidate txmgr.TxCandidate) (*types.Transaction, error) {
	tx, err := retry.Do(ctx, 30, retry.Fixed(2*time.Second), func() (*types.Transaction, error) {
		tx, err := m.craftTx(ctx, candidate)
		if err != nil {
			m.l.Warn("Failed to create a transaction, will retry", "err", err)
		}
		return tx, err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create the tx: %w", err)
	}
	return tx, nil
}

// craftTx creates the signed transaction
// It queries L1 for the current fee market conditions as well as for the nonce.
// NOTE: This method SHOULD NOT publish the resulting transaction.
// NOTE: If the [TxCandidate.GasLimit] is non-zero, it will be used as the transaction's gas.
// NOTE: Otherwise, the [SimpleTxManager] will query the specified backend for an estimate.
func (m *PreconfTxMgr) craftTx(ctx context.Context, candidate txmgr.TxCandidate) (*types.Transaction, error) {
	m.l.Debug("crafting Transaction", "blobs", len(candidate.Blobs), "calldata_size", len(candidate.TxData))

	var err error
	var sidecar *types.BlobTxSidecar
	var blobHashes []common.Hash
	if len(candidate.Blobs) > 0 {
		if candidate.To == nil {
			return nil, errors.New("blob txs cannot deploy contracts")
		}
		if sidecar, blobHashes, err = txmgr.MakeSidecar(candidate.Blobs); err != nil {
			return nil, fmt.Errorf("failed to make sidecar: %w", err)
		}
	}

	gasLimit := candidate.GasLimit
	// If the gas limit is set, we can use that as the gas
	if gasLimit == 0 {
		// Calculate the intrinsic gas for the transaction
		callMsg := ethereum.CallMsg{
			From:      m.cfg.From,
			To:        candidate.To,
			GasTipCap: big.NewInt(0),
			GasFeeCap: big.NewInt(0),
			Data:      candidate.TxData,
			Value:     candidate.Value,
		}
		if len(blobHashes) > 0 {
			callMsg.BlobGasFeeCap = big.NewInt(0)
			callMsg.BlobHashes = blobHashes
		}
		gas, err := m.backend.EstimateGas(ctx, callMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas: %w", errutil.TryAddRevertReason(err))
		}
		gasLimit = gas
	}

	var txMessage types.TxData
	if sidecar != nil {
		txMessage = &types.BlobTx{
			To:         *candidate.To,
			Data:       candidate.TxData,
			Gas:        gasLimit,
			BlobHashes: blobHashes,
			Sidecar:    sidecar,
		}
	} else {
		txMessage = &types.DynamicFeeTx{
			// TODO: get chain id somewhere
			// ChainID:   m.chainID,
			To:        candidate.To,
			GasTipCap: big.NewInt(0),
			GasFeeCap: big.NewInt(0),
			Value:     candidate.Value,
			Data:      candidate.TxData,
			Gas:       gasLimit,
		}
	}
	return m.signWithNextNonce(ctx, txMessage) // signer sets the nonce field of the tx
}

// signWithNextNonce returns a signed transaction with the next available nonce.
// The nonce is fetched once using eth_getTransactionCount with "latest", and
// then subsequent calls simply increment this number. If the transaction manager
// is reset, it will query the eth_getTransactionCount nonce again. If signing
// fails, the nonce is not incremented.
func (m *PreconfTxMgr) signWithNextNonce(ctx context.Context, txMessage types.TxData) (*types.Transaction, error) {
	m.nonceLock.Lock()
	defer m.nonceLock.Unlock()

	if m.nonce == nil {
		// Fetch the sender's nonce from the latest known block (nil `blockNumber`)
		childCtx, cancel := context.WithTimeout(ctx, m.cfg.NetworkTimeout)
		defer cancel()
		nonce, err := m.backend.NonceAt(childCtx, m.cfg.From, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce: %w", err)
		}
		m.nonce = &nonce
	} else {
		*m.nonce++
	}

	switch x := txMessage.(type) {
	case *types.DynamicFeeTx:
		x.Nonce = *m.nonce
	case *types.BlobTx:
		x.Nonce = *m.nonce
	default:
		return nil, fmt.Errorf("unrecognized tx type: %T", x)
	}
	ctx, cancel := context.WithTimeout(ctx, m.cfg.NetworkTimeout)
	defer cancel()
	tx, err := m.cfg.Signer(ctx, m.cfg.From, types.NewTx(txMessage))
	if err != nil {
		// decrement the nonce, so we can retry signing with the same nonce next time
		// signWithNextNonce is called
		*m.nonce--
	}
	return tx, err
}

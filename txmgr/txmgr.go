package txmgr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	u256 "github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	luban "github.com/risechain/luban-api/types"
)

type PreconfClient interface {
	GetSlots(ctx context.Context) ([]luban.SlotInfo, error)
	GetPreconfFee(ctx context.Context, slot uint64) (uint64, uint64, error)

	ReserveBlockspace(ctx context.Context, req luban.ReserveBlockSpaceRequest) (uuid.UUID, error)

	SubmitTransaction(ctx context.Context, reqId uuid.UUID, tx *types.Transaction) error
}

type ETHBackend interface {
	txmgr.ETHBackend

	BlockByNumber(ctx context.Context, num *big.Int) (*types.Block, error)
}

type PreconfTxMgr struct {
	backend ETHBackend
	client  PreconfClient

	l   log.Logger
	cfg *txmgr.Config

	nonce     *uint64
	beaconUrl string
	nonceLock sync.RWMutex
}

func NewPreconfTxMgr(l log.Logger, backend ETHBackend, cfg *txmgr.Config, client PreconfClient, beaconUrl string) *PreconfTxMgr {
	return &PreconfTxMgr{
		backend:   backend,
		client:    client,
		l:         l,
		cfg:       cfg,
		beaconUrl: beaconUrl,
	}
}

func (m *PreconfTxMgr) getHeadSlot() (uint64, error) {
	url := fmt.Sprintf("%s/eth/v1/node/syncing", m.beaconUrl)

	// Make the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to make GET request: %w", err)
	}
	defer resp.Body.Close()

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Define a struct to match the JSON structure
	var syncingResponse struct {
		Data struct {
			HeadSlot     string `json:"head_slot"`
			SyncDistance string `json:"sync_distance"`
			IsSyncing    bool   `json:"is_syncing"`
			IsOptimistic bool   `json:"is_optimistic"`
			ELOffline    bool   `json:"el_offline"`
		} `json:"data"`
	}

	// Decode the JSON response directly into the struct
	if err := json.NewDecoder(resp.Body).Decode(&syncingResponse); err != nil {
		return 0, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Convert the head_slot to uint64
	headSlot, err := strconv.ParseUint(syncingResponse.Data.HeadSlot, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid head_slot value: %w", err)
	}

	return headSlot, nil
}

func (m *PreconfTxMgr) getSlotForCandidate(ctx context.Context, candidate *txmgr.TxCandidate) (uint64, error) {
	slots, err := m.client.GetSlots(ctx)
	// TODO: retry here or on a preconf client side?
	if err != nil {
		return 0, fmt.Errorf("geting slots for preconf failed: %w", err)
	}

	head, err := m.getHeadSlot()
	if err != nil {
		return 0, fmt.Errorf("geting head slot for preconf failed: %w", err)
	}

	nBlobs := uint32(len(candidate.Blobs))
	slot := uint64(0)
	for _, s := range slots {
		// TODO: once luban fixes sending old slots remove it or
		// filter only the first slot
		if s.Slot <= head+1 {
			continue
		}
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
		return 0, errors.New("No slots available for transaction")
	}

	return slot, nil
}

// TODO: wrap in timeout?
func (m *PreconfTxMgr) Send(ctx context.Context, candidate txmgr.TxCandidate) (*types.Receipt, error) {
	tx, err := m.prepare(ctx, candidate)
	if err != nil {
		return nil, fmt.Errorf("preparing tx failed: %w", err)
	}

	var slot uint64

	nBlobs := uint32(len(candidate.Blobs))

	for {
		slot, err = m.getSlotForCandidate(ctx, &candidate)
		// XXX: Figure out if we should wait till next slot or it should be fatal
		if err != nil {
			return nil, fmt.Errorf("Failed to get slot for preconf: %w", err)
		}
		m.l.Debug("Got slot for preconf", "slot", slot)

		gasPrice, blobPrice, err := m.client.GetPreconfFee(ctx, slot)
		if err != nil {
			return nil, fmt.Errorf("Failed to get preconf fee: %w", err)
		}
		m.l.Debug("Got preconf fee", "gasPrice", gasPrice, "blobPrice", blobPrice)

		// { gas_limit * gas_fee + blob_count * blob_gas_fee } * 0.5
		gas := u256.NewInt(tx.Gas())
		gas = gas.Mul(gas, u256.NewInt(gasPrice))
		blob := u256.NewInt(uint64(nBlobs))
		blob = blob.Mul(blob, u256.NewInt(blobPrice))
		deposit := gas.Add(gas, blob).Div(gas, u256.NewInt(2))

		reserveReq := luban.ReserveBlockSpaceRequest{
			BlobCount:  nBlobs,
			Deposit:    hexutil.U256(*deposit),
			GasLimit:   tx.Gas(),
			TargetSlot: slot,
			// Tip is actually the same as deposit
			Tip: hexutil.U256(*deposit),
		}
		id, err := m.client.ReserveBlockspace(ctx, reserveReq)
		if err != nil {
			m.l.Warn(
				"Reserving blockspace for tx failed. Someone probably took our slot. Retrying...",
				"err", err,
			)
			continue
		}

		m.l.Debug("Reserved blockspace", "req", reserveReq)

		err = m.client.SubmitTransaction(ctx, id, tx)
		if err != nil {
			m.l.Error("Sending preconfed tx failed. Slashing preconfer...", "err", err)
			// TODO: slash preconfer
			continue
		}
		break
	}

	head, err := m.getHeadSlot()
	getHeadErr := fmt.Errorf("Failed getting head, while waiting for preconf to fire: %w", err)
	if err != nil {
		return nil, getHeadErr
	}

	m.l.Debug("Waiting for preconf", "slot", slot, "head", head)

	for head < slot+1 {
		time.Sleep(time.Second)
		head, err = m.getHeadSlot()
		if err != nil {
			return nil, getHeadErr
		}
	}

	// TODO: Get err once there is an endpoint in case no receipt
	return m.backend.TransactionReceipt(ctx, tx.Hash())
}

func (m *PreconfTxMgr) getBaseFees(ctx context.Context) (*big.Int, *big.Int, error) {
	bn, err := m.backend.BlockNumber(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get last block number: %w", err)
	}
	blk, err := m.backend.BlockByNumber(ctx, big.NewInt(int64(bn)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get latest block (#%d): %w", bn, err)
	}
	// XXX: assume mainnet
	baseFee := eip1559.CalcBaseFee(params.MainnetChainConfig, blk.Header(), blk.Header().Time+1)
	blobFee := eip4844.CalcBlobFee(*blk.Header().ExcessBlobGas)

	return baseFee.Mul(baseFee, big.NewInt(2)), blobFee.Mul(blobFee, big.NewInt(2)), nil
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

	baseFee, blobFee, err := m.getBaseFees(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get base fees: %w", err)
	}

	var txMessage types.TxData
	if sidecar != nil {
		txMessage = &types.BlobTx{
			To:         *candidate.To,
			Data:       candidate.TxData,
			Gas:        gasLimit,
			GasFeeCap:  u256.MustFromBig(baseFee),
			BlobFeeCap: u256.MustFromBig(blobFee),
			BlobHashes: blobHashes,
			Sidecar:    sidecar,
		}
	} else {
		txMessage = &types.DynamicFeeTx{
			To:        candidate.To,
			GasFeeCap: baseFee,
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

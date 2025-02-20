package client

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"testing"
	"time"

	u256 "github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/risechain/luban-api/escrow"
	luban "github.com/risechain/luban-api/types"
)

type testSetup struct {
	Escrow    *escrow.Escrow
	Rpc       *ethclient.Client
	Key       *ecdsa.PrivateKey
	Preconfer *Client
	ChainId   *big.Int
	beaconUrl string
	ctx       context.Context
}

func (t *testSetup) getHeadSlot() (uint64, error) {
	url := fmt.Sprintf("%s/eth/v1/node/syncing", t.beaconUrl)

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

func (t *testSetup) getBaseFee() *big.Int {
	bn, err := t.Rpc.BlockNumber(t.ctx)
	if err != nil {
		panic(err)
	}
	blk, err := t.Rpc.BlockByNumber(t.ctx, big.NewInt(int64(bn)))
	if err != nil {
		panic(err)
	}
	// XXX: assume mainnet
	return eip1559.CalcBaseFee(params.MainnetChainConfig, blk.Header(), blk.Header().Time+1)
}

func (t *testSetup) getBlobFee() *big.Int {
	bn, err := t.Rpc.BlockNumber(t.ctx)
	if err != nil {
		panic(err)
	}
	blk, err := t.Rpc.BlockByNumber(t.ctx, big.NewInt(int64(bn)))
	if err != nil {
		panic(err)
	}
	return eip4844.CalcBlobFee(*blk.Header().ExcessBlobGas)
}

func (t *testSetup) Balance() *big.Int {
	return t.BalanceOf(crypto.PubkeyToAddress(t.Key.PublicKey))
}

func (t *testSetup) BalanceOf(addr common.Address) *big.Int {
	opts := &bind.CallOpts{Context: t.ctx}
	balance, err := t.Escrow.BalanceOf(opts, addr)
	if err != nil {
		panic(err)
	}
	return balance
}

func (t *testSetup) Deposit(amount *big.Int) {
	opts, err := bind.NewKeyedTransactorWithChainID(t.Key, t.ChainId)
	if err != nil {
		panic(err)
	}
	opts.Value = amount

	fmt.Println("Deposit")
	deposit, err := t.Escrow.Deposit(opts)
	if err != nil {
		panic(err)
	}
	fmt.Println("Sending deposit")
	if err := t.Rpc.SendTransaction(t.ctx, deposit); err != nil {
		panic(err)
	}
	fmt.Println("Getting receipt")
	receipt, err := t.Rpc.TransactionReceipt(t.ctx, deposit.Hash())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Receipt: %v\n", receipt)
}

func newSetup(key, escrowAddr, gateway, beacon, rpcUrl string, chainId *big.Int) *testSetup {
	ecdsa, err := crypto.HexToECDSA(key)
	if err != nil {
		panic(err)
	}

	escrowAddress, err := hex.DecodeString(escrowAddr)
	if err != nil {
		panic(err)
	}

	cl, err := rpc.Dial(rpcUrl)
	if err != nil {
		panic(err)
	}

	rpc := ethclient.NewClient(cl)
	escrow, err := escrow.NewEscrow(common.Address(escrowAddress), rpc)
	if err != nil {
		panic(err)
	}

	preconfer, err := NewClient(gateway, ecdsa)
	if err != nil {
		panic(err)
	}

	return &testSetup{
		Rpc:       rpc,
		Escrow:    escrow,
		Key:       ecdsa,
		Preconfer: preconfer,
		ChainId:   chainId,
		beaconUrl: beacon,
		ctx:       context.Background(),
	}
}

func newTestSetup() *testSetup {
	return newSetup(
		"b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f292",
		"894B19A54A829b00Ad9F1394DD82cB6746531ce0",
		"https://gateway.taiyi-devnet-0.preconfs.org",
		"https://bn.bootnode-1.taiyi-devnet-0.preconfs.org",
		"https://rpc.bootnode-1.taiyi-devnet-0.preconfs.org",
		big.NewInt(7028081469),
	)
}

func TestSubmitTxDigest(t *testing.T) {
	setup := newTestSetup()

	balance := setup.Balance()
	fmt.Printf("Our balance is %v\n", balance)

	if balance.Cmp(big.NewInt(params.Ether)) < 1 {
		ethBalance, err := setup.Rpc.PendingBalanceAt(setup.ctx, crypto.PubkeyToAddress(setup.Key.PublicKey))
		if err != nil {
			panic(err)
		}
		fmt.Printf("Our eth balance is %v\n", ethBalance)

		setup.Deposit(big.NewInt(params.Ether))
	}

	addr := crypto.PubkeyToAddress(setup.Key.PublicKey)
	callMsg := ethereum.CallMsg{
		To:        &addr,
		GasTipCap: big.NewInt(0),
		GasFeeCap: big.NewInt(0),
	}
	gas, err := setup.Rpc.EstimateGas(setup.ctx, callMsg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Transfer costs %v\n", gas)

	slots, err := setup.Preconfer.GetSlots(setup.ctx)
	if err != nil {
		panic(err)
	}

	head, err := setup.getHeadSlot()
	if err != nil {
		panic(err)
	}

	var slot uint64
	for _, s := range slots {
		if s.Slot <= head+1 {
			continue
		}
		if s.GasAvailable < gas {
			continue
		}
		slot = s.Slot
		break
	}
	if slot == 0 {
		panic("No empty slots")
	}

	gasPrice, _, err := setup.Preconfer.GetPreconfFee(setup.ctx, slot)
	if err != nil {
		panic(err)
	}

	// gas_limit * gas_fee * 0.5
	gasLimit := u256.NewInt(gas)
	gasLimit = gasLimit.Mul(gasLimit, u256.NewInt(gasPrice))
	deposit := gasLimit.Div(gasLimit, u256.NewInt(2))

	id, err := setup.Preconfer.ReserveBlockspace(setup.ctx, luban.ReserveBlockSpaceRequest{
		Deposit:    hexutil.U256(*deposit),
		GasLimit:   gas,
		TargetSlot: slot,
		Tip:        hexutil.U256(*deposit),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Preconf id: %v\n", id)

	nonce, err := setup.Rpc.NonceAt(setup.ctx, addr, nil)
	if err != nil {
		panic(err)
	}

	txMessage := &types.DynamicFeeTx{
		ChainID:   setup.ChainId,
		To:        &addr,
		GasTipCap: big.NewInt(0),
		GasFeeCap: setup.getBaseFee(),
		Gas:       gas,
		Nonce:     nonce,
	}
	signer := types.LatestSignerForChainID(setup.ChainId)
	tx := types.MustSignNewTx(setup.Key, signer, txMessage)
	err = setup.Preconfer.SubmitTransaction(setup.ctx, id, tx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Submitted tx with hash: %v\n", tx.Hash())

	head, err = setup.getHeadSlot()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Waiting for slot %d (head is %d)\n", slot, head)
	for head < slot+1 {
		time.Sleep(12 * time.Second)
		head, err = setup.getHeadSlot()
		if err != nil {
			panic(err)
		}
	}

	otherTx, pending, err := setup.Rpc.TransactionByHash(setup.ctx, tx.Hash())
	if err != nil {
		panic(err)
	}
	fmt.Printf("TX: %#+v\n", otherTx)
	fmt.Printf("pending: %v\n", pending)

	receipt, err := setup.Rpc.TransactionReceipt(setup.ctx, tx.Hash())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Tx was confirmed. Receipt: %#+v\n", receipt)
}

func TestSubmitBlob(t *testing.T) {
	setup := newTestSetup()

	balance := setup.Balance()
	fmt.Printf("Our balance is %v\n", balance)

	if balance.Cmp(big.NewInt(params.Ether)) < 1 {
		ethBalance, err := setup.Rpc.PendingBalanceAt(setup.ctx, crypto.PubkeyToAddress(setup.Key.PublicKey))
		if err != nil {
			panic(err)
		}
		fmt.Printf("Our eth balance is %v\n", ethBalance)

		setup.Deposit(big.NewInt(params.Ether))
	}

	blobs := []*eth.Blob{&eth.Blob{}}
	sidecar, blobHashes, err := txmgr.MakeSidecar(blobs)
	if err != nil {
		panic(err)
	}

	addr := crypto.PubkeyToAddress(setup.Key.PublicKey)
	slots, err := setup.Preconfer.GetSlots(setup.ctx)
	if err != nil {
		panic(err)
	}

	head, err := setup.getHeadSlot()
	if err != nil {
		panic(err)
	}

	var slot uint64
	for _, s := range slots {
		if s.Slot <= head+1 {
			continue
		}
		if s.BlobsAvailable == 0 {
			continue
		}
		slot = s.Slot
		break
	}
	if slot == 0 {
		panic("No empty slots")
	}

	gasPrice, blobPrice, err := setup.Preconfer.GetPreconfFee(setup.ctx, slot)
	if err != nil {
		panic(err)
	}

	callMsg := ethereum.CallMsg{
		To:         &addr,
		BlobHashes: blobHashes,
	}
	gasEstimate, err := setup.Rpc.EstimateGas(setup.ctx, callMsg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Price is: %v %v\n", gasPrice, blobPrice)

	// { gas_limit * gas_fee + blob_count * blob_gas_fee } * 0.5
	gas := u256.NewInt(gasEstimate)
	gas = gas.Mul(gas, u256.NewInt(gasPrice))
	blob := u256.NewInt(blobPrice)
	deposit := gas.Add(gas, blob).Div(gas, u256.NewInt(2))

	id, err := setup.Preconfer.ReserveBlockspace(setup.ctx, luban.ReserveBlockSpaceRequest{
		Deposit:    hexutil.U256(*deposit),
		Tip:        hexutil.U256(*deposit),
		BlobCount:  1,
		GasLimit:   gasEstimate,
		TargetSlot: slot,
	})
	if err != nil {
		panic(err)
	}

	nonce, err := setup.Rpc.NonceAt(setup.ctx, addr, nil)
	if err != nil {
		panic(err)
	}

	txMessage := &types.BlobTx{
		To:         addr,
		BlobHashes: blobHashes,
		Sidecar:    sidecar,
		Gas:        gasEstimate,
		GasFeeCap:  u256.MustFromBig(setup.getBaseFee()),
		BlobFeeCap: u256.MustFromBig(setup.getBlobFee()),
		Nonce:      nonce,
	}
	signer := types.LatestSignerForChainID(setup.ChainId)
	tx := types.MustSignNewTx(setup.Key, signer, txMessage)
	err = setup.Preconfer.SubmitTransaction(setup.ctx, id, tx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Submitted tx with hash: %v\n", tx.Hash())

	head, err = setup.getHeadSlot()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Waiting for slot %d (head is %d)\n", slot, head)
	for head < slot+1 {
		head, err = setup.getHeadSlot()
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second)
	}

	otherTx, pending, err := setup.Rpc.TransactionByHash(setup.ctx, tx.Hash())
	if err != nil {
		panic(err)
	}
	fmt.Printf("TX: %#+v\n", otherTx)
	fmt.Printf("pending: %v\n", pending)

	receipt, err := setup.Rpc.TransactionReceipt(setup.ctx, tx.Hash())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Tx was confirmed. Receipt: %#+v\n", receipt)
}

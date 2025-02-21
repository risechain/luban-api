# Golang client for Luban's Taiyi gateway

This repo is a collection of things to work with Taiyi gateway in Golang.

## Modules

This repo, consists of several go modules:

- [github.com/risechain/luban-api/types](./types) module with types used by client

- [github.com/risechain/luban-api/client](./client) for interacting with gateway. Simplest way to interact is like this:

```go
import (
  "github.com/risechain/luban-api/client"
  "github.com/risechain/luban-api/types"
)

cl := client.NewClient(gatewayUrl, privateKey)
slots, _ := cl.GetSlots(ctx)
slot := slots[0].Slot
gasFee, blobFee, _ := cl.GetPreconfFee(ctx)
id, _ := cl.ReserveBlockspace(ctx, types.ReserveBlockSpaceRequest{
  GasLimit: tx.Gas(),
  BlobCount: 0,
  TargetSlot: slot,
  Deposit: /* SNIP, basically { gas_limit * gas_fee + blob_count * blob_gas_fee } * 0.5 */
  Tip: /* SNIP, basically { gas_limit * gas_fee + blob_count * blob_gas_fee } * 0.5 */
})
cl.SubmitTransaction(ctx, id, tx)
```

- [github.com/risechain/luban-api/escrow](./escrow) module for interacting with Escrow contact of Taiyi

```go
import (
  "github.com/ethereum/go-ethereum/accounts/abi/bind"
  "github.com/risechain/luban-api/escrow"
)

escrow := escrow.NewEscrow(escrowAddr, rpc)
balance, _ := escrow.BalanceOf(&bind.CallOpts{Context: t.ctx}, ourAddr)

opts, _ := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
opts.Value = 1000000000000000000 // 1 Eth
deposit, _ := escrow.Deposit(opts)
rpc.SendTransaction(ctx, deposit)
```

- [github.com/risechain/luban-api/txmgr](./txmgr) module for synchronous transaction sending, in similar fashion to regular [github.com/ethereum/go-ethereum/ethclient](https://pkg.go.dev/github.com/ethereum/go-ethereum/ethclient), but mainly for OP stack drop-in replacement for TransactionManager API.

```go
import (
  "github.com/risechain/luban-api/client"
  "github.com/risechain/luban-api/types"
  "github.com/risechain/luban-api/txmgr"
)

preconfer := client.NewClient(...)
cfg := &txmgr.Config{
	From:           addr,
	NetworkTimeout: time.Second,
	Signer: /*SNIP*/,
}
txmgr := txmgr.NewPreconfTxMgr(logger, rpc, cfg, preconfer)

cand := txmgr.TxCandidate{/*SNIP*/}
receipt, _ := txmanager.Send(ctx, cand)
```


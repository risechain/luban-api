package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/risechain/luban-api/escrow"
)

type DepositCli struct {
	Escrow  *escrow.Escrow
	Rpc     *ethclient.Client
	Key     *ecdsa.PrivateKey
	ChainId *big.Int

	DepositValue *big.Int
}

func NewDepositCliFromCommand(cmd *cli.Command) (*DepositCli, error) {
	cl, err := rpc.Dial(cmd.String("rpc"))
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to RPC: %w", err)
	}
	rpc := ethclient.NewClient(cl)

	esc := cmd.String("escrow")
	esc = strings.TrimPrefix(esc, "0x")
	escrowAddress, err := hex.DecodeString(esc)
	if err != nil {
		return nil, fmt.Errorf("Invalid escrow addr: %w", err)
	}
	escrow, err := escrow.NewEscrow(common.Address(escrowAddress), rpc)
	if err != nil {
		return nil, fmt.Errorf("Invalid escrow addr: %w", err)
	}

	priv := cmd.String("private-key")
	priv = strings.TrimPrefix(priv, "0x")
	key, err := crypto.HexToECDSA(priv)
	if err != nil {
		return nil, fmt.Errorf("Invalid private key: %w", err)
	}

	chainId := big.NewInt(cmd.Int("chain-id"))

	// XXX: This conversion is good enough for now
	deposit := int64(float64(params.Ether) * cmd.Float("deposit"))

	cli := DepositCli{
		Escrow:       escrow,
		Rpc:          rpc,
		Key:          key,
		ChainId:      chainId,
		DepositValue: big.NewInt(deposit),
	}

	return &cli, nil
}

func (c *DepositCli) Action(ctx context.Context) error {
	ourAddr := crypto.PubkeyToAddress(c.Key.PublicKey)
	balance, err := c.Escrow.BalanceOf(&bind.CallOpts{Context: ctx}, ourAddr)
	if err != nil {
		return fmt.Errorf("Failed to get balance in escrow: %w", err)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(c.Key, c.ChainId)
	if err != nil {
		return fmt.Errorf("Failed to make transactor: %w", err)
	}
	opts.Value = c.DepositValue
	deposit, err := c.Escrow.Deposit(opts)
	if err != nil {
		return fmt.Errorf("Failed to construct deposit transaction: %w", err)
	}

	if err := c.Rpc.SendTransaction(ctx, deposit); err != nil {
		return fmt.Errorf("Failed to deposit to escrow: %w", err)
	}
	receipt, err := c.Rpc.TransactionReceipt(ctx, deposit.Hash())
	if err != nil {
		return fmt.Errorf("Failed to get receipt for deposit to escrow: %w", err)
	}

	newBalance, err := c.Escrow.BalanceOf(&bind.CallOpts{Context: ctx}, ourAddr)
	if err != nil {
		return fmt.Errorf("Failed to get balance in escrow: %w", err)
	}

	log.Printf("Receipt is %v\n", receipt)
	log.Printf("Our balance changed from %v to %v\n", balance, newBalance)

	return nil
}

func main() {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "rpc",
				Value:    "https://rpc.bootnode-1.taiyi-devnet-0.preconfs.org",
				Required: true,
				Usage:    "RPC url for L1 for deposit",
			},
			&cli.StringFlag{
				Name:     "escrow",
				Value:    "0x894B19A54A829b00Ad9F1394DD82cB6746531ce0",
				Required: true,
				Usage:    "Address for escrow contract",
			},
			&cli.IntFlag{
				Name:     "chain-id",
				Value:    7028081469,
				Required: true,
				Usage:    "Set chain id for chain",
			},
			&cli.StringFlag{
				Name:     "private-key",
				Usage:    "Private key for depositing",
				Sources:  cli.EnvVars("PRIVATE_KEY"),
				Required: true,
			},
			&cli.FloatFlag{
				Name:     "deposit",
				Usage:    "Float value in Ether to deposit",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dep, err := NewDepositCliFromCommand(cmd)
			if err != nil {
				return fmt.Errorf("Failed to parse CLI: %w", err)
			}
			return dep.Action(ctx)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

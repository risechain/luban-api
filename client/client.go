package client

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/crypto"

	internal "github.com/risechain/luban-api/internal/client"
	"github.com/risechain/luban-api/types"
)

type Client struct {
	*internal.ClientWithResponses

	key *ecdsa.PrivateKey
}

// FIXME: reexport options
func NewClient(server string, key *ecdsa.PrivateKey, opts ...internal.ClientOption) (*Client, error) {
	cl, err := internal.NewClientWithResponses(server, opts...)
	if err != nil {
		return nil, err
	}
	client := Client{ClientWithResponses: cl, key: key}
	return &client, nil
}

func (cl *Client) GetSlots(ctx context.Context) ([]types.SlotInfo, error) {
	resp, err := cl.ClientWithResponses.GetSlotsWithResponse(ctx)
	if err != nil {
		return []types.SlotInfo{}, err
	}
	if resp.JSON200 == nil {
		return []types.SlotInfo{}, fmt.Errorf("GetSlots return code %v", resp.Status())
	}
	return *resp.JSON200, nil
}

// First one is gas, second one for blob
func (cl *Client) GetPreconfFee(ctx context.Context, slot uint64) (uint64, uint64, error) {
	resp, err := cl.ClientWithResponses.GetFeeWithResponse(ctx, slot)
	if err != nil {
		return 0, 0, err
	}
	if resp.JSON200 == nil {
		return 0, 0, fmt.Errorf("GetPreconfFee return code %v and error `%v`", resp.Status(), resp.JSON500)
	}
	return resp.JSON200.GasFee, resp.JSON200.BlobGasFee, nil
}

func (cl *Client) signReserveBlockspace(req *types.ReserveBlockSpaceRequest) (string, error) {
	signature, err := crypto.Sign(req.Digest().Bytes(), cl.key)
	if err != nil {
		return "", err
	}
	addr := crypto.PubkeyToAddress(cl.key.PublicKey)
	return fmt.Sprintf("%v:0x%s", addr, hex.EncodeToString(signature)), nil
}

func (cl *Client) ReserveBlockspace(
	ctx context.Context,
	req types.ReserveBlockSpaceRequest,
) (uuid.UUID, error) {
	sig, err := cl.signReserveBlockspace(&req)
	if err != nil {
		return uuid.UUID{}, err
	}
	signature := internal.ReserveBlockspaceParams{sig}
	body := internal.ReserveBlockSpaceRequest(req)
	resp, err := cl.ClientWithResponses.ReserveBlockspaceWithResponse(ctx, &signature, body)
	if err != nil {
		return uuid.UUID{}, err
	}
	fmt.Printf("Response %#v\n", string(resp.Body))
	if resp.JSON200 == nil {
		return uuid.UUID{}, fmt.Errorf("ReserveBlockspace return code %v", resp.Status())
	}
	return uuid.UUID(*resp.JSON200), nil
}

func (cl *Client) signSubmitTx(reqId uuid.UUID, tx *types.Transaction) (string, error) {
	signature, err := crypto.Sign(types.SubmitTxDigest(reqId, tx).Bytes(), cl.key)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%s", hex.EncodeToString(signature)), nil
}

// TODO: Handle slashing and everything
func (cl *Client) SubmitTransaction(ctx context.Context, reqId uuid.UUID, tx types.Transaction) error {
	sig, err := cl.signSubmitTx(reqId, &tx)
	if err != nil {
		return err
	}

	params := internal.SubmitTransactionParams{sig}
	req := internal.SubmitTransactionRequest{reqId, &tx}
	resp, err := cl.SubmitTransactionWithResponse(ctx, &params, req)
	if err != nil {
		return err
	}
	if resp.JSON200 == nil {
		return fmt.Errorf("SubmitTransaction return code %v", resp.Status())
	}
	return err
}

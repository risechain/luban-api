package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	internal "github.com/risechain/luban-api/internal/client"
	"github.com/risechain/luban-api/types"
)

type Client struct {
	*internal.ClientWithResponses

	key ecdsa.PrivateKey
}

// FIXME: reexport options
func NewClient(server string, key ecdsa.PrivateKey, opts ...internal.ClientOption) (*Client, error) {
	cl, err := internal.NewClientWithResponses(server, opts...)
	if err != nil {
		return nil, err
	}
	client := Client{ClientWithResponses: cl, key: key}
	return &client, nil
}

func (cl *Client) GetEpochInfo(ctx context.Context) ([]types.SlotInfo, error) {
	resp, err := cl.ClientWithResponses.GetSlotsWithResponse(ctx)
	if err != nil {
		return []types.SlotInfo{}, err
	}
	if resp.JSON200 == nil {
		return []types.SlotInfo{}, fmt.Errorf("GetEpochInfo return code %v", resp.Status())
	}
	return *resp.JSON200, nil
}

func (cl *Client) GetPreconfFee(ctx context.Context, slot int64) (int64, error) {
	resp, err := cl.ClientWithResponses.GetFeeWithResponse(ctx, &internal.GetFeeParams{slot})
	if err != nil {
		return 0, err
	}
	if resp.JSON200 == nil {
		return 0, fmt.Errorf("GetPreconfFee return code %v and error `%v`", resp.Status(), resp.JSON500)
	}
	return *resp.JSON200, nil
}

func (cl *Client) sign(bytes []byte) (string, error) {
	signature, err := crypto.Sign(bytes, &cl.key)
	if err != nil {
		return "", err
	}

	// TODO: signature to hex?
	return string(signature), nil
}

func (cl *Client) ReserveBlockspace(
	ctx context.Context,
	req types.ReserveBlockSpaceRequest,
) (*types.ReserveBlockSpaceResponse, error) {
	id_bin, err := req.Id.MarshalBinary()
	if err != nil {
		return nil, err
	}

	sig, err := cl.sign(append(id_bin, req.TxHash.Bytes()...))
	if err != nil {
		return nil, err
	}
	signature := internal.ReserveBlockspaceParams{sig}
	body := internal.ReserveBlockspaceJSONRequestBody{
		BlobCount:     int(req.BlobCount),
		EscrowDeposit: int(req.EscrowDeposit),
		GasLimit:      int(req.GasLimit),
		TargetSlot:    int(req.TargetSlot),
	}
	resp, err := cl.ClientWithResponses.ReserveBlockspaceWithResponse(ctx, &signature, body)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("ReserveBlockspace return code %v", resp.Status())
	}
	response := types.ReserveBlockSpaceResponse{resp.JSON200.RequestId, resp.JSON200.Signature}
	return &response, nil
}

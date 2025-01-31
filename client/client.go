package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	common_eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"

	internal "github.com/risechain/luban-api/internal/client"
)

type SlotInfo = internal.SlotInfo

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

func (cl *Client) GetEpochInfo(ctx context.Context) ([]SlotInfo, error) {
	resp, err := cl.ClientWithResponses.GetCommitmentsV1EpochInfoWithResponse(ctx)
	if err != nil {
		return []SlotInfo{}, err
	}
	if resp.JSON200 == nil {
		return []SlotInfo{}, fmt.Errorf("GetEpochInfo return code %v", resp.Status())
	}
	return resp.JSON200.AvailableSlots, nil
}

func (cl *Client) GetPreconfFee(ctx context.Context, slot int) (int, error) {
	resp, err := cl.ClientWithResponses.GetCommitmentsV1PreconfFeeWithResponse(ctx, &internal.GetCommitmentsV1PreconfFeeParams{
		Slot: slot,
	})
	if err != nil {
		return 0, err
	}
	if resp.JSON200 == nil {
		return 0, fmt.Errorf("GetPreconfFee return code %v", resp.Status())
	}
	return *resp.JSON200, nil
}

type ReserveBlockSpaceRequest struct {
	Id            uuid.UUID
	TxHash        common_eth.Hash
	BlobCount     int
	EscrowDeposit int
	GasLimit      int
	TargetSlot    int
}

func (cl *Client) signReq(id uuid.UUID, txHash common_eth.Hash) (string, error) {
	id_bin, err := id.MarshalBinary()
	if err != nil {
		return "", err
	}

	signature, err := crypto.Sign(append(id_bin, txHash.Bytes()...), &cl.key)
	if err != nil {
		return "", err
	}

	// TODO: signature to hex?
	return string(signature), nil
}

func (cl *Client) ReserveBlockspace(ctx context.Context, req ReserveBlockSpaceRequest) (*internal.ReserveBlockSpaceResponse, error) {
	sig, err := cl.signReq(req.Id, req.TxHash)
	if err != nil {
		return nil, err
	}
	signature := internal.PostCommitmentsV1ReserveBlockspaceParams{&sig}
	body := internal.PostCommitmentsV1ReserveBlockspaceJSONRequestBody{
		BlobCount:     req.BlobCount,
		EscrowDeposit: req.EscrowDeposit,
		GasLimit:      req.GasLimit,
		TargetSlot:    req.TargetSlot,
	}
	resp, err := cl.ClientWithResponses.PostCommitmentsV1ReserveBlockspaceWithResponse(ctx, &signature, body)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("ReserveBlockspace return code %v", resp.Status())
	}
	return resp.JSON200, nil
}

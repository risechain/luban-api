package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"

	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
	internal "github.com/risechain/luban-api/internal/client"
)

type (
	Transaction = txmgr.TxCandidate
	ClientOption = internal.ClientOption
	SlotInfo = internal.SlotInfo

	ReserveBlockSpaceRequest struct {
		Id            uuid.UUID
		TxHash        common.Hash
		BlobCount     uint
		EscrowDeposit uint
		GasLimit      uint
		TargetSlot    uint
	}
	ReserveBlockSpaceResponse struct {
		RequestId uuid.UUID
		// TODO: figure out type for signature
		Signature string
	}
)

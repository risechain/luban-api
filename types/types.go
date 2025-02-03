package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"

	internal "github.com/risechain/luban-api/internal/client"
)

type (
	Transaction = types.Transaction
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

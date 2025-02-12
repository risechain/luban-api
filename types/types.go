package types

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"

	internal "github.com/risechain/luban-api/internal/client"
)

type (
	Transaction              = types.Transaction
	ClientOption             = internal.ClientOption
	SlotInfo                 = internal.SlotInfo
	ReserveBlockSpaceRequest internal.ReserveBlockSpaceRequest

	ReserveBlockSpaceResponse struct {
		RequestId uuid.UUID
		// TODO: figure out type for signature
		Signature string
	}
)

func appendUint256(o binary.AppendByteOrder, to []byte, u256 hexutil.U256) []byte {
	to = o.AppendUint64(to, u256[0])
	to = o.AppendUint64(to, u256[1])
	to = o.AppendUint64(to, u256[2])
	to = o.AppendUint64(to, u256[3])
	return to
}

func (req *ReserveBlockSpaceRequest) Digest() common.Hash {
	var digest []byte
	le := binary.LittleEndian

	// https://github.com/lu-bann/taiyi/blob/dev/crates/primitives/src/preconf_request.rs#L78-L84
	digest = le.AppendUint64(digest, req.TargetSlot)
	digest = le.AppendUint64(digest, req.GasLimit)
	digest = appendUint256(le, digest, req.Deposit)
	digest = appendUint256(le, digest, req.Tip)
	digest = le.AppendUint64(digest, uint64(req.BlobCount))

	return crypto.Keccak256Hash(digest)
}

// https://docs.rs/uuid/latest/uuid/struct.Uuid.html#method.to_bytes_le
func appendUuidToLe(to []byte, id uuid.UUID) []byte {
	i := ([16]byte)(id)
	to = append(to, i[3], i[2], i[1], i[0])
	to = append(to, i[5], i[4])
	to = append(to, i[7], i[6])
	to = append(to, i[8:]...)
	return to
}

func SubmitTxDigest(reqId uuid.UUID, tx *Transaction) common.Hash {
	var digest []byte

	// https://github.com/lu-bann/taiyi/blob/0c9ebba9010aa097e6a1f4017fb8262a0ee64705/crates/primitives/src/preconf_request.rs#L100-L103
	digest = appendUuidToLe(digest, reqId)
	digest = append(digest, tx.Hash().Bytes()...)

	return crypto.Keccak256Hash(digest)
}

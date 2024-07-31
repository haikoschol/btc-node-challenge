package network

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

type ObjectType uint32

const (
	Error                   ObjectType = 0
	MsgTx                   ObjectType = 1
	MsgBlock                ObjectType = 2
	MsgFilteredBlock        ObjectType = 3
	MsgCmpctBlock           ObjectType = 4
	MsgWitnessTx            ObjectType = 0x40000001
	MsgWitnessBlock         ObjectType = 0x40000002
	MsgFilteredWitnessBlock ObjectType = 0x40000003
)

func (t ObjectType) String() string {
	switch t {
	case Error:
		return "ERROR"
	case MsgTx:
		return "MSG_TX"
	case MsgBlock:
		return "MSG_BLOCK"
	case MsgFilteredBlock:
		return "MSG_FILTERED_BLOCK"
	case MsgCmpctBlock:
		return "MSG_CMPCT_BLOCK"
	case MsgWitnessTx:
		return "MSG_WITNESS_TX"
	case MsgWitnessBlock:
		return "MSG_WITNESS_BLOCK"
	case MsgFilteredWitnessBlock:
		return "MSG_FILTERED_WITNESS_BLOCK"
	default:
		return "UNKNOWN_FLYING_OBJECT"
	}
}

var ErrInvalidInvMessage = errors.New("invalid inv message")

const invVecSize = 36

type InvVec struct {
	Type ObjectType
	Hash [32]byte
}

func (v *InvVec) String() string {
	return fmt.Sprintf("type=%s hash=%s", v.Type, hex.EncodeToString(v.Hash[:]))
}

type InvMessage struct {
	Count     uint64
	Inventory []InvVec
}

func decodeInvMessage(data []byte) (*InvMessage, error) {
	count, ok := decodeVarInt(data)
	if !ok {
		return nil, ErrInvalidInvMessage
	}

	invVecPart := data[count.Size:]
	netSize := uint64(len(invVecPart))

	if netSize/invVecSize != count.Value || netSize%invVecSize != 0 {
		return nil, ErrInvalidInvMessage
	}

	inventory := make([]InvVec, count.Value)
	r := bytes.NewReader(invVecPart)

	for i := 0; uint64(i) < count.Value; i++ {
		var vec InvVec
		if err := binary.Read(r, binary.LittleEndian, &vec); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidInvMessage, err)
		}
		inventory[i] = vec
	}

	return &InvMessage{
		Count:     count.Value,
		Inventory: inventory,
	}, nil
}

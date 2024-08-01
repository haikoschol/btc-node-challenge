package network

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/haikoschol/btc-node-challenge/internal/btc"
	"github.com/haikoschol/btc-node-challenge/internal/vartypes"
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
	Hash btc.BlockHash
}

func (v *InvVec) String() string {
	return fmt.Sprintf("type=%s hash=%s", v.Type, hex.EncodeToString(v.Hash[:]))
}

type InvWithSource struct {
	Inventory []InvVec
	Node      *Node
}

type InvMessage struct {
	Count     uint64
	Inventory []InvVec
}

func decodeInvMessage(data []byte) (*InvMessage, error) {
	buf := bytes.NewBuffer(data)
	count, ok := vartypes.DecodeVarInt(buf)
	if !ok {
		return nil, ErrInvalidInvMessage
	}

	netSize := uint64(buf.Len())

	if netSize/invVecSize != count.Value || netSize%invVecSize != 0 {
		return nil, ErrInvalidInvMessage
	}

	inventory := make([]InvVec, count.Value)

	for i := 0; uint64(i) < count.Value; i++ {
		var vec InvVec
		vec.Type = ObjectType(binary.LittleEndian.Uint32(buf.Next(4)))
		vec.Hash = btc.BlockHash(buf.Next(btc.BlockHashSize))
		inventory[i] = vec
	}

	return &InvMessage{
		Count:     count.Value,
		Inventory: inventory,
	}, nil
}

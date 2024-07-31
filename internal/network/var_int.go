package network

import (
	"encoding/binary"
)

type VarInt struct {
	Value uint64
	Size  int
}

func decodeVarInt(data []byte) (res VarInt, ok bool) {
	if len(data) == 0 {
		return
	}
	res.Size = 1

	switch data[0] {
	case 0xFD:
		res.Size = 3
		if len(data) < res.Size+1 {
			return
		}
		res.Value = uint64(binary.LittleEndian.Uint16(data[1:res.Size]))
	case 0xFE:
		if len(data) < res.Size+1 {
			return
		}
		res.Size = 5
		res.Value = uint64(binary.LittleEndian.Uint32(data[1:res.Size]))
	case 0xFF:
		if len(data) < res.Size+1 {
			return
		}
		res.Size = 9
		res.Value = binary.LittleEndian.Uint64(data[1:res.Size])
	default:
		res.Value = uint64(data[0])
	}

	ok = true
	return
}

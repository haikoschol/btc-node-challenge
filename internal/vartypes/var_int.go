package vartypes

import (
	"encoding/binary"
	"io"
)

type VarIntSize int

const (
	Uint8Size  VarIntSize = 1
	Uint16Size VarIntSize = 3
	Uint32Size VarIntSize = 5
	Uint64Size VarIntSize = 9
)

type VarInt struct {
	Value uint64
	Size  VarIntSize
}

func NewVarInt(value uint64) (v VarInt) {
	v.Value = value

	if value < 0xFD {
		v.Size = Uint8Size
	} else if v.Value <= 0xFFFF {
		v.Size = Uint16Size
	} else if v.Value <= 0xFFFFFFFF {
		v.Size = Uint32Size
	} else {
		v.Size = Uint64Size
	}
	return
}

func (v VarInt) Encode() []byte {
	e := make([]byte, v.Size)

	switch v.Size {
	case Uint8Size:
		e[0] = byte(v.Value)
	case Uint16Size:
		e[0] = 0xFD
		binary.BigEndian.PutUint16(e[1:], uint16(v.Value))
	case Uint32Size:
		e[0] = 0xFE
		binary.BigEndian.PutUint32(e[1:], uint32(v.Value))
	case Uint64Size:
		e[0] = 0xFF
		binary.BigEndian.PutUint64(e[1:], v.Value)
	}

	return e
}

func DecodeVarInt(data []byte) (res VarInt, ok bool) {
	if len(data) == 0 {
		return
	}
	res.Size = Uint8Size

	switch data[0] {
	case 0xFD:
		res.Size = Uint16Size
		if len(data) < int(res.Size)+1 {
			return
		}
		res.Value = uint64(binary.LittleEndian.Uint16(data[1:res.Size]))
	case 0xFE:
		if len(data) < int(res.Size)+1 {
			return
		}
		res.Size = Uint32Size
		res.Value = uint64(binary.LittleEndian.Uint32(data[1:res.Size]))
	case 0xFF:
		if len(data) < int(res.Size)+1 {
			return
		}
		res.Size = Uint64Size
		res.Value = binary.LittleEndian.Uint64(data[1:res.Size])
	default:
		res.Value = uint64(data[0])
	}

	return res, true
}

func WriteAsVarInt(w io.Writer, value uint64) error {
	v := NewVarInt(value)

	written, err := w.Write(v.Encode())
	if err != nil {
		return err
	}

	if written != int(v.Size) {
		return io.ErrShortWrite
	}
	return nil
}

package vartypes

import (
	"bytes"
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

func DecodeVarInt(buf *bytes.Buffer) (res VarInt, ok bool) {
	if buf.Len() == 0 {
		return
	}

	res.Size = Uint8Size
	b, err := buf.ReadByte()
	if err != nil {
		return
	}

	switch b {
	case 0xFD:
		res.Size = Uint16Size
		if buf.Len() < int(res.Size) {
			return
		}
		res.Value = uint64(binary.LittleEndian.Uint16(buf.Next(int(res.Size - 1))))
	case 0xFE:
		if buf.Len() < int(res.Size) {
			return
		}
		res.Size = Uint32Size
		res.Value = uint64(binary.LittleEndian.Uint32(buf.Next(int(res.Size - 1))))
	case 0xFF:
		if buf.Len() < int(res.Size) {
			return
		}
		res.Size = Uint64Size
		res.Value = binary.LittleEndian.Uint64(buf.Next(int(res.Size - 1)))
	default:
		res.Value = uint64(b)
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

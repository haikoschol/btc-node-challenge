package btc

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/haikoschol/btc-node-challenge/internal/vartypes"
	"io"
)

const staticHeaderSize = 80
const BlockHashSize = 32

type BlockHash [BlockHashSize]byte

func (h BlockHash) String() string {
	return hex.EncodeToString(h[:])
}

type Header struct {
	Version    int32
	PrevBlock  BlockHash
	MerkleRoot [32]byte
	Timestamp  uint32
	Bits       uint32
	Nonce      uint32
	TxnCount   vartypes.VarInt
}

func (h *Header) Size() int {
	return staticHeaderSize + int(h.TxnCount.Size)
}

func DecodeHeader(buf *bytes.Buffer) (*Header, error) {
	header := new(Header)
	header.Version = int32(binary.LittleEndian.Uint32(buf.Next(4))) // TODO check overflow

	if _, err := io.ReadFull(buf, header.PrevBlock[:]); err != nil {
		return nil, err
	}

	if _, err := io.ReadFull(buf, header.MerkleRoot[:]); err != nil {
		return nil, err
	}

	header.Timestamp = binary.LittleEndian.Uint32(buf.Next(4))
	header.Bits = binary.LittleEndian.Uint32(buf.Next(4))
	header.Nonce = binary.LittleEndian.Uint32(buf.Next(4))

	var ok bool
	header.TxnCount, ok = vartypes.DecodeVarInt(buf)
	if !ok {
		return nil, errors.New("invalid txn count")
	}

	return header, nil
}

func (h *Header) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, h.Version)
	if err != nil {
		return nil, err
	}

	buf.Write(h.PrevBlock[:])
	buf.Write(h.MerkleRoot[:])

	err = binary.Write(buf, binary.LittleEndian, h.Timestamp)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, h.Bits)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, h.Nonce)
	if err != nil {
		return nil, err
	}

	buf.Write(h.TxnCount.Encode())
	return buf.Bytes(), nil
}

func (h *Header) Hash() (BlockHash, error) {
	b, err := h.Encode()
	if err != nil {
		return BlockHash{}, err
	}

	inner := sha256.Sum256(b[:staticHeaderSize])
	return sha256.Sum256(inner[:]), nil
}

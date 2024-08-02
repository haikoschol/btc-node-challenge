package btc

import (
	"bytes"
	"github.com/haikoschol/btc-node-challenge/internal/vartypes"
	"io"
)

type Block struct {
	Header       Header
	Transactions []Transaction
}

func (b *Block) Hash() (BlockHash, error) {
	return b.Header.Hash()
}

func (b *Block) Encode() ([]byte, error) {
	encHeader, err := b.Header.Encode()
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	written, err := buf.Write(encHeader)
	if err != nil {
		return nil, err
	}
	if written != len(encHeader) {
		return nil, io.ErrShortWrite
	}

	err = vartypes.WriteAsVarInt(buf, uint64(len(b.Transactions)))
	if err != nil {
		return nil, err
	}

	for _, tx := range b.Transactions {
		encoded, err := tx.Encode()
		if err != nil {
			return nil, err
		}

		written, err := buf.Write(encoded)
		if err != nil {
			return nil, err
		}
		if written != len(encoded) {
			return nil, io.ErrShortWrite
		}
	}
	return buf.Bytes(), nil
}

func DecodeBlock(buf *bytes.Buffer) (*Block, error) {
	header, err := DecodeHeader(buf)
	if err != nil {
		return nil, err
	}

	txns := make([]Transaction, header.TxnCount.Value)
	for i := 0; uint64(i) < header.TxnCount.Value; i++ {
		txn, err := DecodeTransaction(buf)
		if err != nil {
			return nil, err
		}
		txns[i] = *txn
	}

	return &Block{
		Header:       *header,
		Transactions: txns,
	}, nil
}

type BlocksByTimestamp []*Block

func (a BlocksByTimestamp) Len() int           { return len(a) }
func (a BlocksByTimestamp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BlocksByTimestamp) Less(i, j int) bool { return a[i].Header.Timestamp < a[j].Header.Timestamp }

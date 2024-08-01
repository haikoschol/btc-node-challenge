package btc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/haikoschol/btc-node-challenge/internal/vartypes"
	"io"
	"log"
)

var ErrInvalidTransaction = errors.New("invalid transaction")
var ErrInvalidTxInput = errors.New("invalid tx input")
var ErrInvalidTxOutput = errors.New("invalid tx output")

type TxHash [32]byte

func (h TxHash) String() string {
	return hex.EncodeToString(h[:])
}

type Transaction struct {
	Version  uint32
	TxIn     []TxInput
	TxOut    []TxOutput
	LockTime uint32
}

type TxInput struct {
	PreviousOutput  OutPoint
	SignatureScript []byte
	Sequence        uint32
}

type TxOutput struct {
	Value        int64
	ScriptPubKey []byte
}

type OutPoint struct {
	Hash  TxHash
	Index uint32
}

func (tx *Transaction) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.LittleEndian, tx.Version); err != nil {
		return nil, err
	}

	if err := vartypes.WriteAsVarInt(buf, uint64(len(tx.TxIn))); err != nil {
		return nil, err
	}

	for _, input := range tx.TxIn {
		encoded, err := input.Encode()
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

	if err := vartypes.WriteAsVarInt(buf, uint64(len(tx.TxOut))); err != nil {
		return nil, err
	}

	for _, output := range tx.TxOut {
		encoded, err := output.Encode()
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

	if err := binary.Write(buf, binary.LittleEndian, tx.LockTime); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DecodeTransaction(data []byte) (*Transaction, error) {
	if len(data) < 4 {
		return nil, ErrInvalidTransaction
	}

	tx := new(Transaction)
	tx.Version = binary.LittleEndian.Uint32(data)

	data = data[4:]
	if len(data) < 1 {
		return nil, ErrInvalidTransaction
	}

	numInputs, ok := vartypes.DecodeVarInt(data)
	if !ok {
		return nil, ErrInvalidTransaction
	}

	var err error
	tx.TxIn = make([]TxInput, numInputs.Value)
	for i := uint64(0); i < numInputs.Value; i++ {
		tx.TxIn[i], err = DecodeTxInput(data)
		if err != nil {
			return nil, err
		}
	}

	numOutputs, ok := vartypes.DecodeVarInt(data)
	if !ok {
		return nil, ErrInvalidTransaction
	}

	tx.TxOut = make([]TxOutput, numOutputs.Value)
	for i := uint64(0); i < numOutputs.Value; i++ {
		tx.TxOut[i], err = DecodeTxOutput(data)
		if err != nil {
			return nil, err
		}
	}

	if len(data) < 4 {
		return nil, ErrInvalidTransaction
	}
	tx.LockTime = binary.LittleEndian.Uint32(data)
	return tx, nil
}

func DecodeTxInput(data []byte) (in TxInput, err error) {
	if len(data) < 40 {
		log.Println("tx input too short")
		return in, ErrInvalidTxInput
	}

	buf := bytes.NewBuffer(data)
	read, err := io.ReadFull(buf, in.PreviousOutput.Hash[:])
	if err != nil {
		log.Println("previous output hash bad")
		return
	}

	log.Println("previous output hash read", read)
	in.PreviousOutput.Index = binary.LittleEndian.Uint32(buf.Next(4))
	log.Println("previous output index", in.PreviousOutput.Index)
	data = buf.Bytes()
	scriptLen, ok := vartypes.DecodeVarInt(data)
	if !ok {
		log.Println("script len bad")
		return in, ErrInvalidTxInput
	}
	log.Println("script len", scriptLen.Value)

	data = data[scriptLen.Size:]
	if uint64(len(data)) < scriptLen.Value+uint64(4) {
		log.Println("script bad")
		return in, ErrInvalidTxInput
	}

	buf = bytes.NewBuffer(data)
	in.SignatureScript = buf.Next(int(scriptLen.Value)) // TODO check overflow

	if buf.Len() < 4 {
		log.Println("rest of the damn owl bad")
		return in, ErrInvalidTxInput
	}
	in.Sequence = binary.LittleEndian.Uint32(buf.Next(4))
	return
}

func (i TxInput) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	written, err := buf.Write(i.PreviousOutput.Hash[:])
	if err != nil {
		return nil, err
	}
	if written != len(i.PreviousOutput.Hash) {
		return nil, io.ErrShortWrite
	}

	err = binary.Write(buf, binary.LittleEndian, i.PreviousOutput.Index)
	if err != nil {
		return nil, err
	}

	err = vartypes.WriteAsVarInt(buf, uint64(len(i.SignatureScript)))
	if err != nil {
		return nil, err
	}

	written, err = buf.Write(i.SignatureScript)
	if err != nil {
		return nil, err
	}
	if written != len(i.SignatureScript) {
		return nil, io.ErrShortWrite
	}

	err = binary.Write(buf, binary.LittleEndian, i.Sequence)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeTxOutput(data []byte) (out TxOutput, err error) {
	if len(data) < 9 {
		return out, ErrInvalidTxOutput
	}

	err = binary.Read(bytes.NewReader(data[:4]), binary.LittleEndian, &out.Value)
	if err != nil {
		return
	}

	data = data[4:]
	spkLen, ok := vartypes.DecodeVarInt(data)
	if !ok {
		return out, ErrInvalidTxOutput
	}
	data = data[:spkLen.Size]

	if uint64(len(data)) < spkLen.Value {
		return out, ErrInvalidTxOutput
	}
	out.ScriptPubKey = data
	return
}

func (i TxOutput) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.LittleEndian, i.Value)
	if err != nil {
		return nil, err
	}

	err = vartypes.WriteAsVarInt(buf, uint64(len(i.ScriptPubKey)))
	if err != nil {
		return nil, err
	}

	written, err := buf.Write(i.ScriptPubKey)
	if err != nil {
		return nil, err
	}
	if written != len(i.ScriptPubKey) {
		return nil, io.ErrShortWrite
	}
	return buf.Bytes(), nil
}

package btc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/haikoschol/btc-node-challenge/internal/vartypes"
	"io"
)

var ErrInvalidTransaction = errors.New("invalid transaction")
var ErrInvalidTxInput = errors.New("invalid tx input")
var ErrInvalidTxOutput = errors.New("invalid tx output")
var ErrInvalidTxWitnesses = errors.New("invalid tx witnesses")

type TxHash [32]byte

func (h TxHash) String() string {
	return hex.EncodeToString(h[:])
}

type Transaction struct {
	Version      uint32
	HasWitnesses bool
	TxIn         []TxInput
	TxOut        []TxOutput
	TxWitnesses  []TxWitness
	LockTime     uint32
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

type TxWitness struct {
	ComponentData []byte
}

type TxWitnesses struct {
	Witnesses []TxWitness
	Size      uint64
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

func DecodeTransaction(buf *bytes.Buffer) (*Transaction, error) {
	if buf.Len() < 4 {
		return nil, ErrInvalidTransaction
	}

	var err error
	tx := new(Transaction)
	tx.Version = binary.LittleEndian.Uint32(buf.Next(4))

	tx.HasWitnesses, err = decodeWitnessFlag(buf)
	if err != nil {
		return nil, err
	}

	numInputs, ok := vartypes.DecodeVarInt(buf)
	if !ok {
		return nil, ErrInvalidTransaction
	}

	tx.TxIn = make([]TxInput, numInputs.Value)
	for i := uint64(0); i < numInputs.Value; i++ {
		tx.TxIn[i], err = DecodeTxInput(buf)
		if err != nil {
			return nil, err
		}
	}

	numOutputs, ok := vartypes.DecodeVarInt(buf)
	if !ok {
		return nil, ErrInvalidTransaction
	}

	tx.TxOut = make([]TxOutput, numOutputs.Value)
	for i := uint64(0); i < numOutputs.Value; i++ {
		tx.TxOut[i], err = DecodeTxOutput(buf)
		if err != nil {
			return nil, err
		}
	}

	if tx.HasWitnesses {
		tx.TxWitnesses, err = decodeTxWitnesses(buf)
	}

	if buf.Len() < 4 {
		return nil, ErrInvalidTransaction
	}

	tx.LockTime = binary.LittleEndian.Uint32(buf.Next(4))
	return tx, nil
}

func DecodeTxInput(buf *bytes.Buffer) (in TxInput, err error) {
	if buf.Len() < 40 {
		return in, ErrInvalidTxInput
	}

	if _, err = io.ReadFull(buf, in.PreviousOutput.Hash[:]); err != nil {
		return
	}

	in.PreviousOutput.Index = binary.LittleEndian.Uint32(buf.Next(4))
	scriptLen, ok := vartypes.DecodeVarInt(buf)
	if !ok {
		return in, ErrInvalidTxInput
	}

	if uint64(buf.Len()) < scriptLen.Value+uint64(4) {
		return in, ErrInvalidTxInput
	}

	in.SignatureScript = buf.Next(int(scriptLen.Value)) // TODO check overflow

	if buf.Len() < 4 {
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

func DecodeTxOutput(buf *bytes.Buffer) (out TxOutput, err error) {
	if buf.Len() < 9 {
		return out, ErrInvalidTxOutput
	}

	err = binary.Read(buf, binary.LittleEndian, &out.Value)
	if err != nil {
		return
	}

	spkLen, ok := vartypes.DecodeVarInt(buf)
	if !ok {
		return out, ErrInvalidTxOutput
	}

	if uint64(buf.Len()) < spkLen.Value {
		return out, ErrInvalidTxOutput
	}
	out.ScriptPubKey = buf.Next(int(spkLen.Value))
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

func decodeTxWitnesses(buf *bytes.Buffer) (w []TxWitness, err error) {
	count, ok := vartypes.DecodeVarInt(buf)
	if !ok {
		return w, ErrInvalidTxWitnesses
	}

	witnesses := make([]TxWitness, count.Value)

	for i := uint64(0); i < count.Value; i++ {
		l, ok := vartypes.DecodeVarInt(buf)
		if !ok {
			return w, ErrInvalidTxWitnesses
		}

		witnesses = append(witnesses, TxWitness{ComponentData: buf.Next(int(l.Value))})
	}
	return
}

func decodeWitnessFlag(buf *bytes.Buffer) (bool, error) {
	flag := buf.Bytes()
	if len(flag) < 2 {
		return false, ErrInvalidTxWitnesses
	}

	if flag[0] == 0 && flag[1] == 1 {
		return true, nil
	}
	return false, nil
}

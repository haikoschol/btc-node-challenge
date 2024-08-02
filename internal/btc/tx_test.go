package btc

import (
	"bytes"
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTxInput(t *testing.T) {
	// copied from a wireshark capture
	b64 := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/////WAOzCw0bTWluZWQgYnkgQW50UG9vbDkwNgQBDgCbaM5l+r5tbdI8Mp3gGuSxnqAo/apRc06vVFjfQKJ4PqW82WixfgjIEAAAAAAAAADyYAAAukUFAAAAAAD/////"
	raw, err := base64.StdEncoding.DecodeString(b64)
	assert.NoError(t, err)

	txInput, err := DecodeTxInput(bytes.NewBuffer(raw))

	t.Run("decodes successfully", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("hash of previous output", func(t *testing.T) {
		expected := TxHash{}
		assert.Equal(t, expected, txInput.PreviousOutput.Hash)
	})

	t.Run("index of previous output", func(t *testing.T) {
		expected := uint32(4294967295)
		assert.Equal(t, expected, txInput.PreviousOutput.Index)
	})

	t.Run("script length", func(t *testing.T) {
		expected := 88
		assert.Equal(t, expected, len(txInput.SignatureScript))
	})

	t.Run("sequence", func(t *testing.T) {
		expected := uint32(4294967295)
		assert.Equal(t, expected, txInput.Sequence)
	})
}

func TestTransaction(t *testing.T) {
	// copied from a wireshark capture
	b64 := "AgAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP////8xA9ALDQRZ3KtmL0ZvdW5kcnkgVVNBIFBvb2wgI2Ryb3Bnb2xkL0HCCF1ncn40AQAAAP////8DIgIAAAAAAAAiUSA9qsqbgqUaypYMFJFYgkYCnX4PxJ4KvbzI/RdXS+XHS/pGExMAAAAAFgAUNfbeJgyfO97kdSTEc6YBbAwFXLkAAAAAAAAAACZqJKohqe1ExHijD6K1cXz6DqwswpPkpN+tXnRgeAfgepMxiW50vgAAAAABAAAAAlUPnrjZiqNcPVTllYUOCwKdK7TTfQHPJVzjoJFgl516AAAAAAD9////sCXtAMcWj5Ba6nnBQLM+VZK8pXBsuUErGRtubCH/drYAAAAAAA=="
	raw, err := base64.StdEncoding.DecodeString(b64)
	assert.NoError(t, err)

	tx, err := DecodeTransaction(bytes.NewBuffer(raw))

	t.Run("decodes successfully", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("version", func(t *testing.T) {
		assert.Equal(t, uint32(2), tx.Version)
	})

	t.Run("number of tx inputs", func(t *testing.T) {
		assert.Equal(t, 1, len(tx.TxIn))
	})

	t.Run("hash of previous output", func(t *testing.T) {
		expected := TxHash{}
		assert.Equal(t, expected, tx.TxIn[0].PreviousOutput.Hash)
	})

	t.Run("index of previous output", func(t *testing.T) {
		expected := uint32(4294967295)
		assert.Equal(t, expected, tx.TxIn[0].PreviousOutput.Index)
	})

	t.Run("script length", func(t *testing.T) {
		assert.Equal(t, 49, len(tx.TxIn[0].SignatureScript))
	})

	t.Run("script content", func(t *testing.T) {
		script := string(tx.TxIn[0].SignatureScript)
		assert.Contains(t, script, "Foundry USA Pool #dropgold")
	})

	t.Run("sequence", func(t *testing.T) {
		expected := uint32(4294967295)
		assert.Equal(t, expected, tx.TxIn[0].Sequence)
	})

	t.Run("number of tx outputs", func(t *testing.T) {
		assert.Equal(t, 3, len(tx.TxOut))
	})

	t.Run("first tx output value", func(t *testing.T) {
		assert.Equal(t, int64(546), tx.TxOut[0].Value)
	})

	t.Run("first tx output script length", func(t *testing.T) {
		assert.Equal(t, 34, len(tx.TxOut[0].ScriptPubKey))
	})

	t.Run("second tx output value", func(t *testing.T) {
		expected := int64(320030458)
		assert.Equal(t, expected, tx.TxOut[1].Value)
	})

	t.Run("second tx output script length", func(t *testing.T) {
		assert.Equal(t, 22, len(tx.TxOut[1].ScriptPubKey))
	})

	t.Run("third tx output value", func(t *testing.T) {
		assert.Equal(t, int64(0), tx.TxOut[2].Value)
	})

	t.Run("third tx output script length", func(t *testing.T) {
		assert.Equal(t, 38, len(tx.TxOut[2].ScriptPubKey))
	})
}

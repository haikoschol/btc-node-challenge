package btc

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTxInput(t *testing.T) {
	// copied from a wireshark capture
	//b64 := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/////WAOzCw0bTWluZWQgYnkgQW50UG9vbDkwNgQBDgCbaM5l+r5tbdI8Mp3gGuSxnqAo/apRc06vVFjfQKJ4PqW82WixfgjIEAAAAAAAAADyYAAAukUFAAAAAAD/////"
	b64 := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/////MQO9Cw0Ev7SrZi9Gb3VuZHJ5IFVTQSBQb29sICNkcm9wZ29sZC9T3QPg4QgAAAAAAAD/////"
	raw, err := base64.StdEncoding.DecodeString(b64)
	assert.NoError(t, err)

	txInput, err := DecodeTxInput(raw)

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
		expected := 49 // 88
		assert.Equal(t, expected, len(txInput.SignatureScript))
	})

	t.Run("sequence", func(t *testing.T) {
		expected := uint32(4294967295)
		assert.Equal(t, expected, txInput.Sequence)
	})
}

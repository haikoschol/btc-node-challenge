package network

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"net/netip"
	"testing"
)

const headerSize = 24

func TestVersionMessage(t *testing.T) {
	versionMessage, err := NewVersionMessage(
		int32(70014),
		None,
		netip.MustParseAddr("10.0.23.42"),
		uint16(8333),
		Network|GetUTXO|Witness,
		int32(0),
		false,
	)

	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = versionMessage.Write(buf)
	assert.NoError(t, err)
	encodedMessage := buf.Bytes()

	t.Run("encoded message starts with magic bytes", func(t *testing.T) {
		assert.Equal(t, Magic[:], encodedMessage[0:magicSize])
	})

	t.Run("encoded message has correct size", func(t *testing.T) {
		expectedSize := headerSize + versionMessage.Header.Size
		assert.Equal(t, expectedSize, uint32(len(encodedMessage)))
	})

	t.Run("encoded message contains 'version' command", func(t *testing.T) {
		assert.Equal(t, VersionCmd[:], encodedMessage[magicSize:magicSize+commandSize])
	})

	t.Run("encoded message contains payload checksum", func(t *testing.T) {
		var size uint32
		sizeOffset := magicSize + commandSize
		buf := bytes.NewBuffer(encodedMessage[sizeOffset : sizeOffset+4])

		err := binary.Read(buf, binary.LittleEndian, &size)

		assert.NoError(t, err)
		assert.Equal(t, versionMessage.Payload.Size(), size)
	})
}

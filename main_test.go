package main

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"io"
	"net/netip"
	"testing"
)

func TestPayload(t *testing.T) {
	emptyPayload := Payload{}

	t.Run("size of empty payload", func(t *testing.T) {
		assert.Equal(t, uint32(0), emptyPayload.Size())
	})

	t.Run("checksum of empty payload", func(t *testing.T) {
		expected := Checksum{0x5D, 0xF6, 0xE0, 0xE2}
		checksum := emptyPayload.Checksum()

		assert.Equal(t, expected, checksum)
	})
}

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

func TestReadMessage(t *testing.T) {
	t.Run("reads verack message", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte{
			0xF9, 0xBE, 0xB4, 0xD9,
			'v', 'e', 'r', 'a', 'c', 'k', 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0,
			0x5D, 0xF6, 0xE0, 0xE2,
		})

		message, err := ReadMessage(buf)

		assert.NoError(t, err)
		assert.True(t, message.Equal(VerackMessage))
	})

	t.Run("rejects invalid magic bytes", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte{
			0x12, 0x23, 0x56, 0x78,
			'v', 'e', 'r', 'a', 'c', 'k', 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0,
			0x5D, 0xF6, 0xE0, 0xE2,
		})

		message, err := ReadMessage(buf)

		assert.Nil(t, message)
		assert.ErrorIs(t, err, ErrInvalidHeader)
	})

	t.Run("rejects unknown command", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte{
			0xF9, 0xBE, 0xB4, 0xD9,
			'w', 'h', 'a', 's', 's', 'u', 'p', '?', 0, 0, 0, 0,
			0, 0, 0, 0,
			0x5D, 0xF6, 0xE0, 0xE2,
		})

		message, err := ReadMessage(buf)

		assert.Nil(t, message)
		assert.ErrorIs(t, err, ErrUnknownCommand)
	})

	t.Run("rejects invalid checksum", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte{
			0xF9, 0xBE, 0xB4, 0xD9,
			'v', 'e', 'r', 's', 'i', 'o', 'n', 0, 0, 0, 0, 0,
			5, 0, 0, 0,
			0x5D, 0xF6, 0xE0, 0xE2,
			0xBA, 0xDC, 0x0F, 0xFE, 0xE0,
		})

		message, err := ReadMessage(buf)

		assert.Nil(t, message)
		assert.ErrorIs(t, err, ErrInvalidChecksum)
	})

	t.Run("rejects size in header too large", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte{
			0xF9, 0xBE, 0xB4, 0xD9,
			'v', 'e', 'r', 's', 'i', 'o', 'n', 0, 0, 0, 0, 0,
			0x42, 0, 0, 0,
			0x27, 0x42, 0x89, 0x52,
			0xBA, 0xDC, 0x0F, 0xFE, 0xE0, 0xDE, 0xCA, 0xF0,
		})

		message, err := ReadMessage(buf)

		assert.Nil(t, message)
		assert.ErrorIs(t, io.ErrUnexpectedEOF, err)
	})
}

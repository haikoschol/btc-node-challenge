package network

import (
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"net/netip"
	"sync"
	"testing"
)

func TestHandshake(t *testing.T) {
	// the address of the peer from the perspective of the handshake function
	// (i.e. the peer simulated by the tests below)
	peerAddr := netip.MustParseAddr("127.0.0.1")

	t.Run("initiates handshake by sending a version message", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		local, peer := net.Pipe()

		go func() {
			defer wg.Done()
			peerVersionMessage, err := handshake(local, peerAddr, 8333, Network)
			assert.Error(t, err) // caused by the peer closing the connection
			assert.Nil(t, peerVersionMessage)
		}()

		msg, err := readMsg(peer)
		assert.NoError(t, err)

		command := msg[magicSize : magicSize+commandSize]
		assert.Equal(t, VersionCmd[:], command)

		peer.Close()
		wg.Wait()
	})

	t.Run("peer sends a verack instead of a version message first", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		local, peer := net.Pipe()

		go func() {
			defer wg.Done()
			peerVersionMessage, err := handshake(local, peerAddr, 8333, Network)
			assert.ErrorIs(t, err, ErrUnexpectedMessage)
			assert.Nil(t, peerVersionMessage)
			local.Close() // simulate the caller of handshake() handling the error by closing the connection
		}()

		_, err := readMsg(peer) // read their version message
		assert.NoError(t, err)

		_ = VerackMessage.Write(peer) // ignore the error caused by the connection being closed in the goroutine above
		wg.Wait()
	})

	versionMessage, err := NewVersionMessage(
		int32(70014),
		None,
		netip.MustParseAddr("127.0.0.1"),
		uint16(8333),
		Network,
		int32(0),
		false,
	)
	assert.NoError(t, err)

	t.Run("peer sends two version messages", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		local, peer := net.Pipe()

		go func() {
			defer wg.Done()
			peerVersionMessage, err := handshake(local, peerAddr, 8333, Network)
			assert.ErrorIs(t, err, ErrUnexpectedMessage)
			assert.Nil(t, peerVersionMessage)
			local.Close() // simulate the caller of handshake() handling the error by closing the connection
		}()

		// read their version message
		_, err = readMsg(peer)
		assert.NoError(t, err)

		err = versionMessage.Write(peer)
		assert.NoError(t, err)

		_ = versionMessage.Write(peer) // ignore the error caused by the connection being closed in the goroutine above
		wg.Wait()
	})

	t.Run("successful handshake", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		local, peer := net.Pipe()

		go func() {
			defer wg.Done()
			peerVersionMessage, err := handshake(local, peerAddr, 8333, Network)
			assert.NoError(t, err)
			assert.True(t, peerVersionMessage.Equal(versionMessage))
		}()

		msg, err := readMsg(peer)
		assert.NoError(t, err)

		command := msg[magicSize : magicSize+commandSize]
		assert.Equal(t, VersionCmd[:], command)

		err = versionMessage.Write(peer)
		assert.NoError(t, err)

		err = VerackMessage.Write(peer)
		assert.NoError(t, err)

		msg, err = readMsg(peer)
		assert.NoError(t, err)

		command = msg[magicSize : magicSize+commandSize]
		assert.Equal(t, VerackCmd[:], command)

		checksum := msg[len(msg)-checksumSize:]
		assert.Equal(t, VerackMessage.Header.Checksum[:], checksum)
		wg.Wait()
	})
}

func read(r io.Reader, size uint32) ([]byte, error) {
	buf := make([]byte, size)
	n, err := r.Read(buf)
	return buf[:n], err
}

func readMsg(r io.Reader) ([]byte, error) {
	hdr, err := read(r, headerSize)
	if err != nil {
		return nil, err
	}

	size := binary.LittleEndian.Uint32(hdr[magicSize+commandSize : magicSize+commandSize+4])
	if size == 0 {
		return hdr, nil
	}

	payload, err := read(r, size)
	if err != nil {
		return nil, err
	}

	return append(hdr, payload...), nil
}

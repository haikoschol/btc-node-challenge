package network

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var (
	VerackMessage = &Message{
		Header:  NewHeader(VerackCmd, Payload{}),
		Payload: Payload{},
	}

	GetaddrMessage = &Message{
		Header:  NewHeader(GetaddrCmd, Payload{}),
		Payload: Payload{},
	}
)

const (
	magicSize    = 4
	checksumSize = 4
)

type Checksum [checksumSize]byte
type Payload []byte

var (
	Magic     = [magicSize]byte{0xF9, 0xBE, 0xB4, 0xD9}
	UserAgent = append([]byte{0x11}, []byte("/Santitham:0.0.1/")...)

	ErrInvalidHeader       = errors.New("invalid header")
	ErrInvalidChecksum     = errors.New("invalid checksum")
	ErrUnexpectedMessage   = errors.New("received unexpected message")
	ErrInvalidPeerVersion  = errors.New("invalid peer version")
	ErrServicesUnavailable = errors.New("requested services unavailable")
)

func (p Payload) Checksum() Checksum {
	sum := sha256.Sum256(p)
	sum = sha256.Sum256(sum[:])
	return Checksum{sum[0], sum[1], sum[2], sum[3]}
}

func (p Payload) Size() uint32 {
	return uint32(len(p))
}

type Header struct {
	Magic    [magicSize]byte
	Command  Command
	Size     uint32
	Checksum Checksum
}

func (h Header) String() string {
	return fmt.Sprintf("command=%s size=%d checksum=%s", h.Command.String(), h.Size, hex.EncodeToString(h.Checksum[:]))
}

func NewHeader(command Command, payload Payload) Header {
	return Header{
		Magic:    Magic,
		Command:  command,
		Size:     payload.Size(),
		Checksum: payload.Checksum(),
	}
}

func ReadHeader(r io.Reader) (Header, error) {
	header := Header{}
	err := binary.Read(r, binary.LittleEndian, &header)
	if err != nil {
		return Header{}, err
	}

	if header.Magic != Magic {
		return Header{}, ErrInvalidHeader
	}

	return header, nil
}

type Message struct {
	Header  Header
	Payload Payload
}

func (m *Message) Command() string {
	return m.Header.Command.String()
}

func (m *Message) Equal(other *Message) bool {
	return m.Header == other.Header && bytes.Equal(m.Payload, other.Payload)
}

func (m *Message) Write(w io.Writer) error {
	err := binary.Write(w, binary.LittleEndian, m.Header)
	if err != nil {
		return err
	}

	if m.Header.Size == 0 {
		return nil
	}

	if written, err := w.Write(m.Payload); err != nil || written != len(m.Payload) {
		if err == nil {
			err = io.ErrShortWrite
		}
		return err
	}
	return nil
}

func ReadMessage(r io.Reader) (*Message, error) {
	header, err := ReadHeader(r)
	if err != nil {
		return nil, err
	}

	payload := Payload(make([]byte, header.Size))
	_, err = io.ReadFull(r, payload)
	if err != nil {
		return nil, err
	}

	if header.Checksum != payload.Checksum() {
		return nil, ErrInvalidChecksum
	}

	return &Message{header, payload}, nil
}

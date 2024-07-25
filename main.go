package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/netip"
	"time"
)

const (
	headerSize   = 24
	magicSize    = 4
	commandSize  = 12
	checksumSize = 4
)

type Command [commandSize]byte
type Checksum [checksumSize]byte
type Service uint64
type Services = Service

const (
	None           Service = 0
	Network        Service = 1
	GetUTXO        Service = 2
	Bloom          Service = 4
	Witness        Service = 8
	Xthin          Service = 16
	CompactFilters Service = 32
	NetworkLimited Service = 64
)

type Payload []byte

var (
	Magic      = [magicSize]byte{0xF9, 0xBE, 0xB4, 0xD9}
	UserAgent  = append([]byte{0x11}, []byte("/Santitham:0.0.1/")...)
	VersionCmd = Command{'v', 'e', 'r', 's', 'i', 'o', 'n', 0, 0, 0, 0, 0}
	VerackCmd  = Command{'v', 'e', 'r', 'a', 'c', 'k', 0, 0, 0, 0, 0, 0}

	VerackMessage = &Message{
		Header:  NewHeader(VerackCmd, Payload{}),
		Payload: Payload{},
	}

	ErrInvalidHeader     = errors.New("invalid header")
	ErrUnknownCommand    = errors.New("unknown command")
	ErrInvalidChecksum   = errors.New("invalid checksum")
	ErrUnexpectedMessage = errors.New("received unexpected message")
)

func (c Command) String() string {
	end := commandSize
	for i := 0; i < commandSize; i++ {
		if c[i] == 0 {
			end = i
			break
		}
	}

	return string(c[:end])
}

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

	if header.Command != VersionCmd && header.Command != VerackCmd {
		return Header{}, ErrUnknownCommand
	}

	return header, nil
}

type Message struct {
	Header  Header
	Payload Payload
}

func (m *Message) Equal(other *Message) bool {
	return m.Header == other.Header && bytes.Equal(m.Payload, other.Payload)
}

func (m *Message) Write(w io.Writer) error {
	err := binary.Write(w, binary.LittleEndian, m.Header)
	if err != nil {
		return err
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

func NewVersionMessage(
	version int32,
	services Services,
	peerAddr netip.Addr,
	peerPort uint16,
	peerServices Services,
	startHeight int32,
	relay bool,
) (*Message, error) {
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.LittleEndian, version)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, services)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, peerServices)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, peerAddr.As16())
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, peerPort)
	if err != nil {
		return nil, err
	}

	dummyAddrFrom := [26]byte{}
	_, err = buf.Write(dummyAddrFrom[:])
	if err != nil {
		return nil, err
	}

	nonce := rand.Uint64() // TODO probably should be stored somewhere
	err = binary.Write(buf, binary.LittleEndian, nonce)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(UserAgent)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, startHeight)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, relay)
	if err != nil {
		return nil, err
	}

	payload := buf.Bytes()

	return &Message{
		Header:  NewHeader(VersionCmd, payload),
		Payload: payload,
	}, nil
}

func Handshake(conn net.Conn, peerAddr string, peerPort int) error {
	versionMessage, err := NewVersionMessage(
		int32(70014),
		None,
		netip.MustParseAddr(peerAddr),
		uint16(peerPort),
		Network,
		int32(0),
		false,
	)
	if err != nil {
		return err
	}

	err = versionMessage.Write(conn)
	if err != nil {
		return err
	}
	log.Println("sent:", versionMessage.Header.String())

	message, err := ReadMessage(conn)
	if err != nil {
		return err
	}

	log.Println("received:", message.Header.String())
	if message.Header.Command != VersionCmd {
		return ErrUnexpectedMessage
	}

	message, err = ReadMessage(conn)
	if err != nil {
		return err
	}

	log.Println("received:", message.Header.String())
	if !message.Equal(VerackMessage) {
		return ErrUnexpectedMessage
	}

	err = VerackMessage.Write(conn)
	if err != nil {
		return err
	}
	log.Println("sent:", VerackMessage.Header.String())

	return nil
}

func main() {
	peerAddr := "14.6.5.33"
	peerPort := 8333

	peer := fmt.Sprintf("%s:%d", peerAddr, peerPort)
	log.Println("dialing", peer)

	conn, err := net.Dial("tcp", peer)
	if err != nil {
		log.Fatal("dial failed:", err)
	}

	defer conn.Close()

	err = Handshake(conn, peerAddr, peerPort)
	if err != nil {
		log.Fatal("handshake failed:", err)
	} else {
		log.Println("closing connection after successful handshake")
	}
}

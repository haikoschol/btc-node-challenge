package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"net/netip"
)

type Node struct {
	addr         netip.Addr
	port         uint16
	conn         net.Conn
	protoVersion int32
	services     Services
}

func Connect(addr netip.Addr, port uint16, requestedServices Services) (*Node, error) {
	peer := fmt.Sprintf("%s:%d", addr.String(), port)

	conn, err := net.Dial("tcp", peer)
	if err != nil {
		return nil, err
	}

	versionMsg, err := handshake(conn, addr, port, requestedServices)
	if err != nil {
		conn.Close()
		return nil, err
	}

	version := binary.LittleEndian.Uint32(versionMsg.Payload[:4])
	if version > math.MaxInt32 {
		conn.Close()
		return nil, ErrInvalidPeerVersion
	}
	protoVersion := int32(version)

	services := Services(binary.LittleEndian.Uint64(versionMsg.Payload[4:12]))
	if services&requestedServices != requestedServices {
		conn.Close()
		return nil, ErrServicesUnavailable
	}

	return &Node{
		addr,
		port,
		conn,
		protoVersion,
		services,
	}, nil
}

func (n *Node) Disconnect() {
	n.conn.Close()
}

func (n *Node) FindPeers() ([]NetAddr, error) {
	err := GetaddrMessage.Write(n.conn)
	if err != nil {
		return nil, err
	}

	var msg *Message
	for {
		msg, err = ReadMessage(n.conn)
		if err != nil {
			return nil, err
		}

		if msg.Header.Command == AddrCmd {
			break
		} else if msg.Header.Command == PingCmd {
			msg.Header.Command = PongCmd
			if err := msg.Write(n.conn); err != nil {
				return nil, err
			}
		} else {
			continue
		}
	}

	varIntSize := 1
	var addrCount uint64
	switch msg.Payload[0] {
	case 0xFD:
		varIntSize = 3
		addrCount = uint64(binary.LittleEndian.Uint16(msg.Payload[1:varIntSize]))
	case 0xFE:
		varIntSize = 5
		addrCount = uint64(binary.LittleEndian.Uint32(msg.Payload[1:varIntSize]))
	case 0xFF:
		varIntSize = 9
		addrCount = binary.LittleEndian.Uint64(msg.Payload[1:varIntSize])
	default:
		addrCount = uint64(msg.Payload[0])
	}

	addrPayloadSize := uint64(len(msg.Payload) - varIntSize)

	if addrPayloadSize/netAddrSize != addrCount || addrPayloadSize%netAddrSize != 0 {
		return nil, ErrCorruptPayload
	}

	buf := bytes.NewBuffer(msg.Payload[varIntSize:])
	result := make([]NetAddr, addrCount)

	for i := 0; i < int(addrCount); i++ {
		result[i] = decodeNetAddr(buf)
	}
	return result, nil
}

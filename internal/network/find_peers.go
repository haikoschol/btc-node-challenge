package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/netip"
)

const netAddrSize = 30

type NetAddr struct {
	Time     uint32
	Services Services
	IPAddr   [16]byte
	Port     uint16
}

func (na *NetAddr) String() string {
	addr := netip.AddrFrom16(na.IPAddr).String()
	return fmt.Sprintf("address=%s port=%d timestamp=%d services=%d", addr, na.Port, na.Time, na.Services)
}

func FindPeers(conn net.Conn) ([]NetAddr, error) {
	err := GetaddrMessage.Write(conn)
	if err != nil {
		return nil, err
	}

	var msg *Message
	for {
		msg, err = ReadMessage(conn)
		if err != nil {
			return nil, err
		}

		if msg.Header.Command == AddrCmd {
			break
		} else if msg.Header.Command == PingCmd {
			msg.Header.Command = PongCmd
			if err := msg.Write(conn); err != nil {
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
	log.Println("addrPayloadSize", addrPayloadSize, "addrCount", addrCount)

	if addrPayloadSize/netAddrSize != addrCount || addrPayloadSize%netAddrSize != 0 {
		return nil, ErrCorruptPayload
	}

	log.Printf("reading %d addresses from %d bytes of payload\n", addrCount, len(msg.Payload))

	buf := bytes.NewBuffer(msg.Payload[varIntSize:])
	result := make([]NetAddr, addrCount)

	for i := 0; i < int(addrCount); i++ {
		result[i] = decodeNetAddr(buf)
	}
	return result, nil
}

func decodeNetAddr(buf *bytes.Buffer) NetAddr {
	timestamp := binary.LittleEndian.Uint32(buf.Next(4))
	services := Services(binary.LittleEndian.Uint64(buf.Next(8)))

	var ipAddr [16]byte
	copy(ipAddr[:], buf.Next(16))

	port := binary.BigEndian.Uint16(buf.Next(2))

	return NetAddr{
		Time:     timestamp,
		Services: services,
		IPAddr:   ipAddr,
		Port:     port,
	}
}

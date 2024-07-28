package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

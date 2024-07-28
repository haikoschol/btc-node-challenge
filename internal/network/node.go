package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net"
	"net/netip"
	"sync"
	"time"
)

const maxPeerCount = 1000

type Node struct {
	OnDisconnect func()
	addr         netip.Addr
	port         uint16
	conn         net.Conn
	protoVersion int32
	services     Services
	lock         sync.Mutex
	peersCh      chan []NetAddr
}

func Connect(addr netip.Addr, port uint16, requestedServices Services) (*Node, error) {
	peer := fmt.Sprintf("%s:%d", addr.String(), port)
	network := "tcp"

	if addr.Is6() {
		peer = fmt.Sprintf("[%s]:%d", addr.String(), port)
		network = "tcp6"
	}

	var dialer net.Dialer
	dialer.Timeout = time.Second * 15
	conn, err := dialer.Dial(network, peer)
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
		addr:         addr,
		port:         port,
		conn:         conn,
		protoVersion: protoVersion,
		services:     services,
		lock:         sync.Mutex{},
		peersCh:      nil,
	}, nil
}

func (n *Node) Disconnect() {
	if n.OnDisconnect != nil {
		n.OnDisconnect()
	}
	n.conn.Close()
}

func (n *Node) Run() {
	peer := fmt.Sprintf("%s:%d", n.addr.String(), n.port)

	for {
		msg, err := ReadMessage(n.conn)
		if err != nil {
			log.Printf("reading message from %s failed: %v. closing connection", peer, err)
			n.Disconnect()
			return
		}

		if msg.Header.Command == PingCmd {
			log.Printf("received ping from %s", peer)
			msg.Header.Command = PongCmd

			if err = msg.Write(n.conn); err != nil {
				log.Printf("sending message to %s failed: %v. closing connection", peer, err)
				n.Disconnect()
				return
			}
		} else if msg.Header.Command == AddrCmd {
			if err = n.handleAddr(msg); err != nil {
				log.Printf("processing addr message from %s failed: %v. closing connection", peer, err)
				n.Disconnect()
				return
			}
		}
	}
}

func (n *Node) handleAddr(msg *Message) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.peersCh == nil {
		return nil
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
		return ErrCorruptPayload
	}

	buf := bytes.NewBuffer(msg.Payload[varIntSize:])
	peers := make([]NetAddr, addrCount)

	for i := 0; i < min(int(addrCount), maxPeerCount); i++ {
		peers[i] = decodeNetAddr(buf)
	}

	n.peersCh <- peers
	close(n.peersCh)
	n.peersCh = nil
	return nil
}

func (n *Node) FindPeers(peersCh chan []NetAddr) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.peersCh != nil {
		close(n.peersCh)
	}

	err := GetaddrMessage.Write(n.conn)
	if err != nil {
		peer := fmt.Sprintf("%s:%d", n.addr.String(), n.port)
		log.Printf("sending message to %s failed: %v. closing connection", peer, err)
		n.Disconnect()
		close(peersCh)
		return
	}
	n.peersCh = peersCh
}

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
	"sync/atomic"
	"time"
)

const maxPeerCount = 1000

// Node represents a node in the Bitcoin network.
// Instances should be created with Connect.
type Node struct {
	// OnDisconnect is executed after the connection to the peer has been closed by calling Disconnect.
	OnDisconnect func()
	// OnError is executed after the connection to the peer has been closed due to a network i/o or protocol error.
	OnError      func(error)
	addr         netip.Addr
	port         uint16
	conn         net.Conn
	protoVersion int32
	services     Services
	lock         sync.Mutex
	peersCh      chan []NetAddr
	stopWritesCh chan bool
	msgWriteCh   chan *Message
	shuttingDown int32
}

// Connect establishes a TCP connection with the host at addr:port and performs a Bitcoin protocol handshake. The
// requestedServices are passed to the host in the version message. If the version message response from the host does
// not contain these services, the connection is aborted and the function returns ErrServicesUnavailable.
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
		stopWritesCh: make(chan bool, 1),
		msgWriteCh:   make(chan *Message, 5),
		shuttingDown: 0,
	}, nil
}

// Disconnect closes the connection to the host and runs the OnDisconnect handler, if it has been set.
// The Node instance should be discarded after calling Disconnect.
func (n *Node) Disconnect() {
	n.disconnect(nil)
}

// Run starts processing messages from the host. It blocks until the connection has been closed, either due to an error
// during network i/o or by calling Disconnect.
func (n *Node) Run() {
	go n.processWrites()

	for {
		msg, err := ReadMessage(n.conn)
		if err != nil {
			n.disconnect(fmt.Errorf("closing connection to %s. reading message failed: %w", n.peer(), err))
			return
		}

		switch msg.Header.Command {
		case PingCmd:
			log.Printf("received ping from %s", n.peer())
			msg.Header.Command = PongCmd
			n.write(msg)
		case AddrCmd:
			n.handleAddr(msg)
		}
	}
}

// FindPeers requests addresses of peers from the host and sends the result over peersCh in one slice. Afterwards,
// FindPeers closes the channel. If an error occurs during sending of the getaddr message, the connection to the host
// is closed, OnDisconnect is called if set and peersCh is closed.
func (n *Node) FindPeers(peersCh chan []NetAddr) {
	n.setPeersCh(peersCh)
	n.write(GetaddrMessage)
}

func (n *Node) processWrites() {
	for {
		select {
		case msg := <-n.msgWriteCh:
			if err := msg.Write(n.conn); err != nil {
				n.disconnect(
					fmt.Errorf(
						"closing connection to %s. sending '%s' message failed: %w",
						msg.Command(),
						n.peer(),
						err,
					),
				)
				return
			}
		case <-n.stopWritesCh:
			return
		}
	}
}

func (n *Node) write(msg *Message) {
	n.msgWriteCh <- msg
}

func (n *Node) handleAddr(msg *Message) {
	if !n.hasPeersCh() {
		return
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
		log.Printf("received corrupt '%s' payload from %s. ignoring message", msg.Command(), n.peer())
		return
	}

	buf := bytes.NewBuffer(msg.Payload[varIntSize:])
	peers := make([]NetAddr, addrCount)

	for i := 0; i < min(int(addrCount), maxPeerCount); i++ {
		peers[i] = decodeNetAddr(buf)
	}

	n.peersCh <- peers
	n.setPeersCh(nil)
	return
}

func (n *Node) disconnect(err error) {
	if n.isShuttingDown() {
		return
	}

	atomic.AddInt32(&n.shuttingDown, 1)
	n.stopWritesCh <- true
	n.conn.Close()
	n.setPeersCh(nil)

	if err != nil && n.OnError != nil {
		n.OnError(err)
	} else if n.OnDisconnect != nil {
		n.OnDisconnect()
	}
}

func (n *Node) setPeersCh(ch chan []NetAddr) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.peersCh != nil {
		close(n.peersCh)

	}
	n.peersCh = ch
}

func (n *Node) hasPeersCh() bool {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.peersCh != nil
}

func (n *Node) peer() string {
	return fmt.Sprintf("%s:%d", n.addr.String(), n.port)
}

func (n *Node) isShuttingDown() bool {
	return n.shuttingDown > 0
}

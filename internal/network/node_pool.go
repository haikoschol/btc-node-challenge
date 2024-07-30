package network

import (
	"errors"
	mapset "github.com/deckarep/golang-set/v2"
	"log"
	"net/netip"
	"time"
)

const maxPeerAge = time.Hour * 24 * 10

type NodePool struct {
	minConnections int
	peersCh        chan []NetAddr
	peerAddrs      mapset.Set[NetAddr]
	nodes          mapset.Set[*Node]
	shutdownCh     chan bool
	errorCh        chan error
}

func NewNodePool(addr netip.Addr, port uint16, minConnections int) (*NodePool, error) {
	node, err := Connect(addr, port, Network)
	if err != nil {
		return nil, err
	}

	nodes := mapset.NewSet[*Node]()
	nodes.Add(node)

	pool := &NodePool{
		peersCh:        make(chan []NetAddr, 1),
		peerAddrs:      mapset.NewSet[NetAddr](),
		minConnections: minConnections,
		nodes:          nodes,
		shutdownCh:     make(chan bool, 1),
		errorCh:        make(chan error, 1),
	}

	node.OnDisconnect = func() {
		pool.nodes.Remove(node)
	}
	node.OnError = func(err error) {
		log.Println(err)
		pool.nodes.Remove(node)
	}

	go node.Run()
	go pool.run()
	node.FindPeers(pool.peersCh)

	return pool, nil
}

func (p *NodePool) Size() int {
	return p.nodes.Cardinality()
}

func (p *NodePool) Shutdown() {
	p.shutdownCh <- true

	// need to copy the set to avoid a deadlock when nodes remove themselves in OnDisconnect during iteration
	s := p.nodes.Clone()

	s.Each(func(n *Node) bool {
		n.Disconnect()
		return false
	})
}

func (p *NodePool) Error() chan error {
	return p.errorCh
}

func (p *NodePool) run() {
	select {
	case <-p.shutdownCh:
		return
	case peers := <-p.peersCh:
		for _, peer := range peers {
			p.peerAddrs.Add(peer)
		}
	}

	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			if p.Size() < p.minConnections {
				p.addConnections()
			}
		case <-p.shutdownCh:
			ticker.Stop()
			return
		}
	}
}

func (p *NodePool) addConnections() {
	before := p.Size()
	if before == 0 {
		p.errorCh <- errors.New("node pool needs at least one connection to add more. shutting down")
		p.Shutdown()
		return
	}

	log.Printf("connecting to more nodes... current: %d target: %d", before, p.minConnections)
	log.Printf("got %d more peers to try", p.peerAddrs.Cardinality())

	for p.Size() < p.minConnections && p.peerAddrs.Cardinality() > 0 {
		if p.isShuttingDown() {
			return
		}

		peer, ok := p.peerAddrs.Pop()
		if !ok {
			// TODO continously collect peers from addr from all connected nodes
			return
		}

		if time.Since(time.Unix(int64(peer.Time), 0)) > maxPeerAge {
			continue
		}

		addr := netip.AddrFrom16(peer.IPAddr).Unmap()
		n, err := Connect(addr, peer.Port, Network)
		if err != nil {
			continue
		}

		n.OnDisconnect = func() {
			p.nodes.Remove(n)
		}
		n.OnError = func(err error) {
			log.Println(err)
			p.nodes.Remove(n)
		}
		p.nodes.Add(n)
		go n.Run()
	}
	log.Printf("connected to %d more nodes", p.Size()-before)
}

func (p *NodePool) isShuttingDown() bool {
	select {
	case <-p.shutdownCh:
		return true
	default:
		return false
	}
}

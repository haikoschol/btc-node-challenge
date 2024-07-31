package network

import (
	"errors"
	mapset "github.com/deckarep/golang-set/v2"
	"log"
	"net/netip"
	"sync"
	"time"
)

const maxPeerAge = time.Hour * 24 * 10

type NodePool struct {
	minConnections int
	addrsCh        chan []NetAddr
	peerAddrs      mapset.Set[NetAddr]
	getAddrPending bool
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
		addrsCh:        make(chan []NetAddr, 1),
		peerAddrs:      mapset.NewSet[NetAddr](),
		getAddrPending: false,
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
	node.FindPeers(pool.addrsCh)
	pool.getAddrPending = true
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
	ticker := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-ticker.C:
			if p.Size() == 0 {
				p.errorCh <- errors.New("node pool needs at least one connection to add more. shutting down")
				ticker.Stop()
				p.Shutdown()
				return
			}

			lowOnPeerAddrs := p.peerAddrs.Cardinality() <= p.minConnections
			lowOnConnections := p.Size() < p.minConnections

			if lowOnConnections && !lowOnPeerAddrs {
				log.Printf(
					"trying to connect to more nodes. current: %d target: %d peer addresses left to try: %d",
					p.peerAddrs.Cardinality(),
					p.minConnections,
					p.peerAddrs.Cardinality(),
				)

				added := p.addConnections()
				if added > 0 {
					log.Printf("connected to %d more nodes", added)
				} else if added < 0 {
					log.Printf("lost %d connections", added*-1)
				} else {
					log.Println("failed to connect to more nodes")
				}
			} else if lowOnPeerAddrs && !p.getAddrPending {
				log.Println("running low on peer addresses. requesting more...")
				p.getAddrPending = p.requestPeerAddrs()
			}
		case addrs := <-p.addrsCh:
			log.Printf("received %d peer addresses", len(addrs))
			p.addPeerAddrs(addrs)
			p.getAddrPending = false
		case <-p.shutdownCh:
			ticker.Stop()
			return
		}
	}
}

func (p *NodePool) addPeerAddrs(addrs []NetAddr) {
	for _, addr := range addrs {
		p.peerAddrs.Add(addr)
	}
}

func (p *NodePool) addConnections() int {
	before := p.Size()
	batch := p.getPeerBatch()
	var wg sync.WaitGroup
	wg.Add(len(batch))

	for _, peer := range batch {
		go func() {
			defer wg.Done()

			if p.isShuttingDown() {
				return
			}

			n, err := p.connect(peer)
			if err != nil {
				return
			}
			go n.Run()
		}()
	}
	wg.Wait()

	return p.Size() - before
}

func (p *NodePool) connect(peer NetAddr) (*Node, error) {
	addr := netip.AddrFrom16(peer.IPAddr).Unmap()
	n, err := Connect(addr, peer.Port, Network)
	if err != nil {
		return nil, err
	}

	n.OnDisconnect = func() {
		p.nodes.Remove(n)
	}
	n.OnError = func(err error) {
		log.Println(err)
		p.nodes.Remove(n)
	}

	p.nodes.Add(n)
	return n, nil
}

func (p *NodePool) getPeerBatch() (batch []NetAddr) {
	batchSize := p.minConnections * 2

	for len(batch) < batchSize && p.peerAddrs.Cardinality() > 0 {
		peer, ok := p.peerAddrs.Pop()
		if !ok {
			return
		}

		if time.Since(time.Unix(int64(peer.Time), 0)) > maxPeerAge {
			continue
		}
		batch = append(batch, peer)
	}
	return
}

func (p *NodePool) requestPeerAddrs() bool {
	node, ok := p.nodes.Pop()
	if !ok {
		return false
	}

	p.nodes.Add(node)
	node.FindPeers(p.addrsCh)
	return true
}

func (p *NodePool) isShuttingDown() bool {
	select {
	case <-p.shutdownCh:
		return true
	default:
		return false
	}
}

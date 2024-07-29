package main

import (
	"context"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/haikoschol/btc-node-challenge/internal/network"
	"log"
	"net/netip"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	peerAddr := netip.MustParseAddr("155.4.214.12")
	peerPort := uint16(8333)

	node, err := network.Connect(peerAddr, peerPort, network.Network)
	if err != nil {
		log.Fatalf("unable to connect to %s:%d: %v", peerAddr.String(), peerPort, err)
	}
	go node.Run()

	var peers []network.NetAddr
	peersCh := make(chan []network.NetAddr, 1)
	node.FindPeers(peersCh)

	select {
	case <-ctx.Done():
		log.Println("shutting down...")
		node.Disconnect()
		return
	case peers = <-peersCh:
		log.Printf("found %d peers", len(peers))
	}

	nodes := mapset.NewSet[*network.Node]()
	nodes.Add(node)

	node.OnError = func(err error) {
		log.Println(err)
		nodes.Remove(node)
	}

	go connectAtLeast(peers, 5, nodes)

	<-ctx.Done()
	log.Println("shutting down...")

	// TODO call these in a goroutine and have a timeout after which os.Exit() is called
	node.Disconnect()
	for n := range nodes.Iter() {
		n.Disconnect()
	}
}

func connectAtLeast(peers []network.NetAddr, minConnections int32, nodes mapset.Set[*network.Node]) {
	maxAge := time.Hour * 24 * 10
	var connected int32
	offset := 0
	batchSize := 10

	for connected < minConnections {
		var wg sync.WaitGroup

		for _, peer := range peers[offset : offset+batchSize] {
			wg.Add(1)
			go func() {
				defer wg.Done()

				if connected >= minConnections {
					return
				}
				addr := netip.AddrFrom16(peer.IPAddr).Unmap()

				if time.Since(time.Unix(int64(peer.Time), 0)) > maxAge {
					log.Println(addr.String(), "is more than ten days old, skipping")
					return
				}
				log.Printf("connecting to %s:%d", addr.String(), peer.Port)

				n, err := network.Connect(addr, peer.Port, network.Network)
				if err != nil {
					log.Println(err)
					return
				}

				atomic.AddInt32(&connected, 1)
				nodes.Add(n)

				n.OnError = func(err error) {
					log.Println(err)
					atomic.AddInt32(&connected, -1)
					nodes.Remove(n)
				}
				go n.Run()
			}()
		}

		offset += batchSize
		if offset+batchSize >= len(peers) {
			break
		}
		wg.Wait()
	}
}

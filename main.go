package main

import (
	"context"
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

	peerAddr := netip.MustParseAddr("159.223.20.99")
	peerPort := uint16(8333)

	node, err := network.Connect(peerAddr, peerPort, network.Network)
	if err != nil {
		log.Fatalf("unable to connect to %s:%d: %v", peerAddr.String(), peerPort, err)
	}

	defer node.Disconnect()
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

	go connectAtLeast(peers, 5)

	<-ctx.Done()
	// TODO disconnect from all peers
	log.Println("shutting down...")
}

func connectAtLeast(peers []network.NetAddr, minConnections int32) {
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
				n.OnDisconnect = func() { atomic.AddInt32(&connected, -1) }
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

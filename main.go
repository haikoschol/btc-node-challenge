package main

import (
	"context"
	"github.com/haikoschol/btc-node-challenge/internal/network"
	"log"
	"net/netip"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	peerAddr := netip.MustParseAddr("95.168.169.66")
	peerPort := uint16(8333)

	pool, err := network.NewNodePool(peerAddr, peerPort, 10)
	if err != nil {
		log.Fatal(err)
	}

	select {
	case <-ctx.Done():
		log.Println("shutting down...")
		// TODO call this in a goroutine and have a timeout after which os.Exit() is called
		pool.Shutdown()
	case err := <-pool.Error():
		log.Fatal(err)
	}
}

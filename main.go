package main

import (
	"context"
	"github.com/haikoschol/btc-node-challenge/internal/network"
	"log"
	"net/netip"
	"os"
	"os/signal"
	"time"
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
		shutdown(pool)
	case err := <-pool.Error():
		log.Fatal(err)
	}
}

func shutdown(pool *network.NodePool) {
	timer := time.NewTimer(time.Second * 10)
	go func() {
		pool.Shutdown()
		timer.Stop()
		os.Exit(0)
	}()

	<-timer.C
	log.Println("shutdown timed out. aborting")
	os.Exit(1)
}

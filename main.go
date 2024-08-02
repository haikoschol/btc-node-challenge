package main

import (
	"context"
	"github.com/haikoschol/btc-node-challenge/internal/network"
	"log"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func main() {
	statePath, err := getStatePath("state.bin")
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	peerAddr := netip.MustParseAddr("95.168.169.66")
	peerPort := uint16(8333)

	pool, err := network.NewNodePool(peerAddr, peerPort, 10, statePath)
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

func getStatePath(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Abs(filepath.Join(wd, name))
}

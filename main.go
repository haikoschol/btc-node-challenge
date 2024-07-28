package main

import (
	"github.com/haikoschol/btc-node-challenge/internal/network"
	"log"
	"net/netip"
)

func main() {
	peerAddr := netip.MustParseAddr("159.223.20.99")
	peerPort := uint16(8333)

	node, err := network.Connect(peerAddr, peerPort, network.Network)
	if err != nil {
		log.Fatalf("unable to connect to %s:%d: %v", peerAddr.String(), peerPort, err)
	}

	defer node.Disconnect()

	peers, err := node.FindPeers()
	if err != nil {
		log.Fatal("finding peers failed: ", err)
	}

	log.Printf("found %d peers\n", len(peers))
	for _, peer := range peers {
		log.Println(peer.String())
	}
}

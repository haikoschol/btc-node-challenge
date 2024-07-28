package main

import (
	"fmt"
	"github.com/haikoschol/btc-node-challenge/internal/network"
	"log"
	"net"
	"net/netip"
)

func main() {
	peerAddr := netip.MustParseAddr("159.223.20.99")
	peerPort := 8333

	peer := fmt.Sprintf("%s:%d", peerAddr.String(), peerPort)
	log.Println("dialing", peer)

	conn, err := net.Dial("tcp", peer)
	if err != nil {
		log.Fatal("dial failed:", err)
	}

	defer conn.Close()

	err = network.Handshake(conn, peerAddr, peerPort)
	if err != nil {
		log.Fatal("handshake failed:", err)
	}

	peers, err := network.FindPeers(conn)
	if err != nil {
		log.Fatal("finding peers failed: ", err)
	}

	log.Printf("found %d peers\n", len(peers))
	for _, peer := range peers {
		log.Println(peer.String())
	}
}

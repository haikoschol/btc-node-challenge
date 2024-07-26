package main

import (
	"fmt"
	"github.com/haikoschol/btc-node-challenge/internal/network"
	"log"
	"net"
)

func main() {
	peerAddr := "14.6.5.33"
	peerPort := 8333

	peer := fmt.Sprintf("%s:%d", peerAddr, peerPort)
	log.Println("dialing", peer)

	conn, err := net.Dial("tcp", peer)
	if err != nil {
		log.Fatal("dial failed:", err)
	}

	defer conn.Close()

	err = network.Handshake(conn, peerAddr, peerPort)
	if err != nil {
		log.Fatal("handshake failed:", err)
	} else {
		log.Println("closing connection after successful handshake")
	}
}

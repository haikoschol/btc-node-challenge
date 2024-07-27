package network

import (
	"log"
	"net"
	"net/netip"
)

const protocolVersion = 70012

func Handshake(conn net.Conn, peerAddr string, peerPort int) error {
	versionMessage, err := NewVersionMessage(
		int32(protocolVersion),
		None,
		netip.MustParseAddr(peerAddr),
		uint16(peerPort),
		Network,
		int32(0),
		false,
	)
	if err != nil {
		return err
	}

	err = versionMessage.Write(conn)
	if err != nil {
		return err
	}
	log.Println("sent:", versionMessage.Header.String())

	message, err := ReadMessage(conn)
	if err != nil {
		return err
	}

	log.Println("received:", message.Header.String())
	if message.Header.Command != VersionCmd {
		return ErrUnexpectedMessage
	}

	message, err = ReadMessage(conn)
	if err != nil {
		return err
	}

	log.Println("received:", message.Header.String())
	if !message.Equal(VerackMessage) {
		return ErrUnexpectedMessage
	}

	err = VerackMessage.Write(conn)
	if err != nil {
		return err
	}
	log.Println("sent:", VerackMessage.Header.String())

	return nil
}

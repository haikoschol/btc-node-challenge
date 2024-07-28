package network

import (
	"log"
	"net"
	"net/netip"
)

const protocolVersion = 70012

func handshake(conn net.Conn, peerAddr netip.Addr, peerPort uint16, connServices Services) (*Message, error) {
	versionMessage, err := NewVersionMessage(
		int32(protocolVersion),
		connServices,
		peerAddr,
		peerPort,
		connServices,
		int32(0),
		false,
	)
	if err != nil {
		return nil, err
	}

	err = versionMessage.Write(conn)
	if err != nil {
		return nil, err
	}
	log.Println("sent:", versionMessage.Header.String())

	message, err := ReadMessage(conn)
	if err != nil {
		return nil, err
	}

	log.Println("received:", message.Header.String())
	if message.Header.Command != VersionCmd {
		return nil, ErrUnexpectedMessage
	}

	peerVersionMsg := message

	message, err = ReadMessage(conn)
	if err != nil {
		return nil, err
	}

	log.Println("received:", message.Header.String())
	if !message.Equal(VerackMessage) {
		return nil, ErrUnexpectedMessage
	}

	err = VerackMessage.Write(conn)
	if err != nil {
		return nil, err
	}
	log.Println("sent:", VerackMessage.Header.String())

	return peerVersionMsg, nil
}

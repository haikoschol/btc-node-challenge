package network

import (
	"net"
	"net/netip"
)

const protocolVersion = 70012

func handshake(conn net.Conn, peerAddr netip.Addr, peerPort uint16, connServices Services) (*Message, error) {
	//peer := fmt.Sprintf("%s:%d", peerAddr.String(), peerPort)

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
	//log.Printf("sent [%s] to %s", versionMessage.Header.String(), peer)

	message, err := ReadMessage(conn)
	if err != nil {
		return nil, err
	}

	//log.Printf("received [%s] from %s", message.Header.String(), peer)
	if message.Header.Command != VersionCmd {
		return nil, ErrUnexpectedMessage
	}

	peerVersionMsg := message

	message, err = ReadMessage(conn)
	if err != nil {
		return nil, err
	}

	//log.Printf("received [%s] from %s", message.Header.String(), peer)
	if !message.Equal(VerackMessage) {
		return nil, ErrUnexpectedMessage
	}

	err = VerackMessage.Write(conn)
	if err != nil {
		return nil, err
	}
	//log.Printf("sent [%s] to %s", VerackMessage.Header.String(), peer)

	return peerVersionMsg, nil
}

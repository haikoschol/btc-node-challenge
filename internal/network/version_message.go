package network

import (
	"bytes"
	"encoding/binary"
	"math/rand/v2"
	"net/netip"
	"time"
)

func NewVersionMessage(
	version int32,
	services Services,
	peerAddr netip.Addr,
	peerPort uint16,
	peerServices Services,
	startHeight int32,
	relay bool,
) (*Message, error) {
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.LittleEndian, version)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, services)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, peerServices)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, peerAddr.As16())
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, peerPort)
	if err != nil {
		return nil, err
	}

	dummyAddrFrom := [26]byte{}
	_, err = buf.Write(dummyAddrFrom[:])
	if err != nil {
		return nil, err
	}

	nonce := rand.Uint64() // TODO probably should be stored somewhere
	err = binary.Write(buf, binary.LittleEndian, nonce)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(UserAgent)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, startHeight)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, relay)
	if err != nil {
		return nil, err
	}

	payload := buf.Bytes()

	return &Message{
		Header:  NewHeader(VersionCmd, payload),
		Payload: payload,
	}, nil
}

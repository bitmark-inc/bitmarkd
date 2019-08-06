package p2p

import (
	"fmt"
)

func Send(address string, peerPublicKey []byte, messages [][]byte) (string, error) {
	addr, isV6, err := canonicalAddress(address)
	if nil != err {
		return "", fmt.Errorf("invalid connection address. error: %s", err)
	}

	socket, err := openSocket(peerPublicKey, addr, isV6)
	if nil != err {
		return "", fmt.Errorf("failed to open socket connection %q. error: %s", addr, err)
	}
	defer closeSocket(socket, addr)

	err = sendMessageBytes(socket, messages)
	if nil != err {
		return "", fmt.Errorf("send message error: %s", err)
	}

	m, err := receiveMessageBytes(socket)
	if nil != err {
		return "", fmt.Errorf("receive message error: %s", err)
	}

	return parseResponse(m), nil
}

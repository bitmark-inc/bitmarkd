package p2p

import (
	"crypto/rand"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	zmq "github.com/pebbe/zmq4"
)

const (
	publicKeyLength = 32
	socketIDLength  = 32
	connTimeout     = 15 * time.Second
)

func openSocket(serverPubKey []byte, address string, v6 bool) (*zmq.Socket, error) {
	socket, err := zmq.NewSocket(zmq.REQ)
	if nil != err {
		return nil, err
	}

	// set identity
	r := make([]byte, socketIDLength)
	_, err = rand.Read(r)
	if nil != err {
		return nil, err
	}
	id := string(r)
	err = socket.SetIdentity(id)
	if nil != err {
		return nil, err
	}

	// set key pair
	pk, sk, err := zmq.NewCurveKeypair()
	if nil != err {
		return nil, err
	}
	err = socket.SetCurvePublickey(pk)
	if nil != err {
		return nil, err
	}

	err = socket.SetCurveSecretkey(sk)
	if nil != err {
		return nil, err
	}

	// set server public key
	err = socket.SetCurveServerkey(string(serverPubKey))
	if nil != err {
		return nil, err
	}

	err = socket.SetImmediate(true)
	if nil != err {
		return nil, err
	}

	// set connection timeout
	err = socket.SetSndtimeo(connTimeout)
	if nil != err {
		return nil, err
	}
	err = socket.SetRcvtimeo(connTimeout)
	if nil != err {
		return nil, err
	}

	err = socket.SetIpv6(v6)
	if nil != err {
		return nil, err
	}

	err = socket.Connect(address)
	if nil != err {
		return nil, err
	}

	return socket, nil
}

func closeSocket(socket *zmq.Socket, address string) error {
	err := socket.Disconnect(address)
	if nil != err {
		return err
	}

	return socket.Close()
}

func sendMessageBytes(socket *zmq.Socket, messages [][]byte) error {
	lastIndex := len(messages) - 1
	for i, m := range messages {
		var flag zmq.Flag
		if lastIndex == i {
			flag = 0
		} else {
			flag = zmq.SNDMORE
		}
		_, err := socket.SendBytes(m, flag)
		if nil != err {
			return err
		}
	}
	return nil
}

func receiveMessageBytes(socket *zmq.Socket) ([][]byte, error) {
	return socket.RecvMessageBytes(0)
}

// canonical connection address
// try to lookup dns resolver if a host is provided instead of IP
//
// eg. 	192.168.1.1:2136 -> tcp://192.168.1.1:2136
// 			localhost:2136 -> tcp://[::1]:2136 (v6) or tcp://127.0.0.1:2136
func canonicalAddress(address string) (string, bool, error) {
	host, port, err := net.SplitHostPort(address)
	if nil != err {
		return "", false, err
	}

	ip := net.ParseIP(strings.Trim(host, " "))
	if nil == ip {
		ips, err := net.LookupIP(host)
		if nil != err {
			return "", false, err
		}

		if len(ips) < 1 {
			return "", false, fmt.Errorf("invalid ip address")
		}
		ip = ips[0]
	}

	numericPort, err := strconv.Atoi(strings.Trim(port, " "))
	if nil != err {
		return "", false, err
	}

	if numericPort < 1 || numericPort > 65535 {
		return "", false, fmt.Errorf("invalid port number")
	}

	isV6 := strings.Contains(ip.String(), ":")
	var ca string
	if isV6 {
		ca = fmt.Sprintf("tcp://[%s]:%d", ip, numericPort)
	} else {
		ca = fmt.Sprintf("tcp://%s:%d", ip, numericPort)
	}

	return ca, isV6, nil
}

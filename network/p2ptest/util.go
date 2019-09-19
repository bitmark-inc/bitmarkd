package main

import (
	"net"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
)

const (
	taggedPublic  = "PUBLIC:"
	taggedPrivate = "PRIVATE:"
	publicLength  = 32
	privateLength = 32
)

// ParseHostPort - parse host:port
func ParseHostPort(hostPort string) (string, string, error) {
	host, port, err := net.SplitHostPort(hostPort)
	if nil != err {
		return "", "", err
	}
	IP := strings.Trim(host, " ")
	numericPort, err := strconv.Atoi(strings.Trim(port, " "))
	if nil != err {
		return "", "", err
	}
	if numericPort < 1 || numericPort > 65535 {
		return "", "", fault.ErrInvalidPortNumber
	}
	return IP, strconv.Itoa(numericPort), nil
}

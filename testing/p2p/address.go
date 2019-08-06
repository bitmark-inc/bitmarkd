package p2p

import (
	"fmt"
)

func GetLocalPeerAddress(index int) string {
	return fmt.Sprintf("127.0.0.1:%d36", 21+index)
}

func GetLocalPeerIPV6Address(index int) string {
	return fmt.Sprintf("[::1]:%d36", 21+index)
}

func GetLocalRPCAddress(index int) string {
	return fmt.Sprintf("127.0.0.1:%d30", 21+index)
}

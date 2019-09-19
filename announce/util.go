package announce

import (
	"fmt"

	"github.com/bitmark-inc/bitmarkd/util"

	proto "github.com/golang/protobuf/proto"
	ma "github.com/multiformats/go-multiaddr"
)

func printBinaryAddrs(addrs []byte) string {
	maAddrs := Addrs{}
	err := proto.Unmarshal(addrs, &maAddrs)
	if err != nil {
		return ""
	}
	printAddrs := printMaAddrs(util.GetMultiAddrsFromBytes(maAddrs.Address))
	return printAddrs
}

func printMaAddrs(addrs []ma.Multiaddr) string {
	printAddrs := ""
	for idx, straddr := range addrs {
		if 0 == idx {
			printAddrs = straddr.String()
		} else {
			printAddrs = fmt.Sprintf("%s%s\n", printAddrs, straddr)
		}
	}
	return printAddrs
}

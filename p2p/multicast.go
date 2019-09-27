package p2p

import (
	"context"
	"fmt"
	"os"

	"github.com/bitmark-inc/bitmarkd/messagebus"

	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// SubHandler multicasting subscription handler
func (n *Node) SubHandler(ctx context.Context, sub *pubsub.Subscription) {
	log := n.log
	log.Info("-- Sub start listen --")

	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		req := &BusMessage{}
		err = proto.Unmarshal(msg.Data, req)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		switch req.Command {
		case "peer":
			dataLength := len(req.Parameters)
			if dataLength < 3 {
				log.Debugf("peer with too few data: %d items", dataLength)
				break
			}

			if 8 != len(req.Parameters[2]) {
				log.Debugf("peer with invalid timestamp=%v", req.Parameters[2])
				break
			}
			id, err := peer.IDFromBytes(req.Parameters[0])
			log.Infof("-->>sub Recieve: %v  ID:%s \n", req.Command, id.ShortString())
			if err != nil {
				log.Error("invalid id in requesting")
			}
			messagebus.Bus.Announce.Send("addpeer", req.Parameters[0], req.Parameters[1], req.Parameters[2])
		default:
			log.Infof("unreganized Command:%s ", req.Command)
		}
	}
}

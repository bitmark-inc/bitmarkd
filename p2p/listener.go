package p2p

import (
	"bufio"
	"errors"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/logger"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

const maxBytesRecieve = 2000

//ListenHandler is a host Listening  handler
type ListenHandler struct {
	ID        peerlib.ID
	registers []*network.Stream
	log       *logger.L
}

//NewListenHandler return a NewListenerHandler
func NewListenHandler(ID peerlib.ID, log *logger.L) ListenHandler {
	return ListenHandler{ID: ID, log: log}
}

func (l *ListenHandler) handleStream(stream network.Stream) {
	defer stream.Close()
	log := l.log
	log.Info("--- Start A New stream --")
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	//nodeChain := mode.ChainName()
	nodeChain := "local"
	log.Infof("chain Name:%v", nodeChain)
	for {
		req := make([]byte, maxBytesRecieve)
		reqLen, err := rw.Read(req)
		if err != nil {

			listenerSendError(rw, nodeChain, err, "-->READ", log)
			break
		}
		if reqLen < 1 {
			listenerSendError(rw, nodeChain, errors.New("length of byte recieved is less than 1"), "-->READ", log)
		}
		reqChain, fn, parameters, err := UnPackP2PMessage(req[:reqLen])
		if err != nil {
			listenerSendError(rw, nodeChain, err, "-->Unpack", log)
		}
		if len(reqChain) < 2 {
			listenerSendError(rw, nodeChain, errors.New("Invalid Chain"), "-->Unpack", log)
		}

		if reqChain != nodeChain {
			listenerSendError(rw, nodeChain, errors.New("Different Chain"), "-->Chain", log)
		}

		switch fn {
		case "R":
			reqID, reqMaAddrs, timestamp, err := UnPackRegisterData(parameters)
			if err != nil {
				listenerSendError(rw, nodeChain, err, "-->RegData", log)
				break
			}

			announce.AddPeer(reqID, reqMaAddrs, timestamp) // id, listeners, timestam
			randPeerID, randListeners, randTs, err := announce.GetRandom(reqID)
			var randData [][]byte
			if nil != err { // No Random Node sendback this Node
				randData, err = PackRegisterData(nodeChain, fn, reqID, reqMaAddrs, time.Now())
				break
			}
			randData, err = PackRegisterData(nodeChain, fn, randPeerID, randListeners, randTs)
			p2pMessagePacked, err := proto.Marshal(&P2PMessage{Data: randData})
			if err != nil {
				listenerSendError(rw, nodeChain, err, "-->radomn node", log)
				break
			}
			_, err = rw.Write(p2pMessagePacked)
			rw.Flush()
			log.Info(fmt.Sprintf("WRITE:\x1b[32mLength:%d\x1b[0m> ", len(p2pMessagePacked)))
		}
	}
}

func listenerSendError(sender *bufio.ReadWriter, chain string, err error, logPrefix string, log *logger.L) {
	errorMessage := [][]byte{[]byte(chain), []byte("E"), []byte(err.Error())}
	packedP2PMessage, err := proto.Marshal(&P2PMessage{Data: errorMessage})
	_, wErr := sender.Write(packedP2PMessage)
	if wErr != nil && log != nil {
		log.Errorf("%s  \x1b[32mError:%v \x1b[0m", logPrefix, wErr)
	}
	if log != nil {
		fmt.Printf("%s  \x1b[32mError:%v \x1b[0m\n", logPrefix, err)
	}
	sender.Flush()
}

func printP2PMessage(msg P2PMessage, l *logger.L) {
	chain := string(msg.Data[0])
	fn := string(msg.Data[1])
	id, _ := peerlib.IDFromBytes(msg.Data[2])
	l.Info(fmt.Sprintf("%s RECIEVE: chain: \x1b[32m%s fn:\x1b[0m%s> id:%s", "listener", chain, fn, id.String()))
}

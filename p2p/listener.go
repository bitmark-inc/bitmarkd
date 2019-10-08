package p2p

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/logger"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

const maxBytesRecieve = 2000

//ListenHandler is a host Listening  handler
type ListenHandler struct {
	ID   peerlib.ID
	log  *logger.L
	node *Node
}

type serverInfo struct {
	Version string `json:"version"`
	Chain   string `json:"chain"`
	Normal  bool   `json:"normal"`
	Height  uint64 `json:"height"`
}

//NewListenHandler return a NewListenerHandler
func NewListenHandler(ID peerlib.ID, node *Node, log *logger.L) ListenHandler {
	return ListenHandler{ID: ID, log: log, node: node}
}

func (l *ListenHandler) handleStream(stream network.Stream) {
	defer stream.Close()
	log := l.log
	log.Info("--- Start A New stream --")
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	//nodeChain := mode.ChainName()
	nodeChain := "local"
	log.Infof("chain Name:%v", nodeChain)
	req := make([]byte, maxBytesRecieve)
	reqLen, err := rw.Read(req)

	if err != nil {
		listenerSendError(rw, nodeChain, err, "-->READ", log)
		return
	}
	if reqLen < 1 {
		listenerSendError(rw, nodeChain, errors.New("length of byte recieved is less than 1"), "-->READ", log)
		return
	}
	reqChain, fn, parameters, err := UnPackP2PMessage(req[:reqLen])

	if err != nil {
		listenerSendError(rw, nodeChain, err, "-->Unpack", log)
		return
	}
	if len(reqChain) < 2 {
		listenerSendError(rw, nodeChain, errors.New("Invalid Chain"), "-->Unpack", log)
		return
	}
	if reqChain != nodeChain {
		listenerSendError(rw, nodeChain, errors.New("Different Chain"), "-->Chain", log)
		return
	}

	log.Info(fmt.Sprintf("%s RECIEVED:\x1b[32mfn:%s\x1b[0m> ", fn))

	switch fn {
	case "I": // server information
		info := serverInfo{
			Version: l.node.Version,
			Chain:   nodeChain,
			Normal:  mode.Is(mode.Normal),
			Height:  blockheader.Height(),
		}
		// chain, fn, info
		result, err := json.Marshal(info)
		if err != nil {
			listenerSendError(rw, nodeChain, err, "-->Query Server  Information", log)
			return
		}
		respParams := [][]byte{result}
		packed, err := PackP2PMessage(nodeChain, "I", respParams)
		if err != nil {
			listenerSendError(rw, nodeChain, err, "-->Query Server Information", log)
			return
		}
		rw.Write(packed)
		rw.Flush()
	case "N": // get block number
		blockNumber := blockheader.Height()
		result := make([]byte, 8)
		binary.BigEndian.PutUint64(result, blockNumber)
		respParams := [][]byte{result}
		packed, err := PackP2PMessage(nodeChain, "I", respParams)
		if err != nil {
			listenerSendError(rw, nodeChain, err, "-->Query Server Information", log)
			return
		}
		rw.Write(packed)
		rw.Flush()
	case "B": // get packed block
		if 1 != len(parameters) {
			err = fault.ErrMissingParameters
		} else if 6 == len(parameters[0]) { //it 8 or 6 ??
			result := storage.Pool.Blocks.Get(parameters[0])
			if nil == result {
				err = fault.ErrBlockNotFound
				listenerSendError(rw, nodeChain, err, "-->Query Block: block not found", log)
			}
			respParams := [][]byte{result}
			packed, err := PackP2PMessage(nodeChain, "B", respParams)
			if err != nil {
				listenerSendError(rw, nodeChain, err, "-->Query Block  Information", log)
				return
			}
			rw.Write(packed)
			rw.Flush()
		} else {
			err = fault.ErrBlockNotFound
			listenerSendError(rw, nodeChain, err, "-->Query Block: invalid parameter", log)
		}
	case "H": // get block hash
		if 1 != len(parameters) {
			err = fault.ErrMissingParameters
		} else if 6 == len(parameters[0]) { //it 8 or 6 ??
			number := binary.BigEndian.Uint64(parameters[0])
			d, e := blockheader.DigestForBlock(number)
			var result []byte
			if nil == e {
				result = d[:]
			} else {
				err = e
			}
			respParams := [][]byte{result}
			packed, err := PackP2PMessage(nodeChain, "B", respParams)
			if err != nil {
				listenerSendError(rw, nodeChain, err, "-->Query Block  Information", log)
				return
			}
			rw.Write(packed)
			rw.Flush()
		} else {
			err = fault.ErrBlockNotFound
			listenerSendError(rw, nodeChain, err, "-->Query Block: invalid parameter", log)
		}
	case "R":
		nType, reqID, reqMaAddrs, timestamp, err := UnPackRegisterData(parameters)
		if err != nil {
			listenerSendError(rw, nodeChain, err, "-->RegData", log)
			return
		}
		if nType != "client" {
			log.Info(fmt.Sprintf("register:\x1b[32mClient registered\x1b[0m>"))
			announce.AddPeer(reqID, reqMaAddrs, timestamp) // id, listeners, timestam
		}
		randPeerID, randListeners, randTs, err := announce.GetRandom(reqID)
		var randData [][]byte
		if nil != err { // No Random Node sendback this Node
			randData, err = PackRegisterData(nodeChain, fn, nType, reqID, reqMaAddrs, time.Now())
		} else {
			randData, err = PackRegisterData(nodeChain, fn, "servant", randPeerID, randListeners, randTs)
		}

		p2pMessagePacked, err := proto.Marshal(&P2PMessage{Data: randData})
		if err != nil {
			listenerSendError(rw, nodeChain, err, "-><- Radom node", log)
			return
		}
		l.node.addToRegister(reqID, stream)
		_, err = rw.Write(p2pMessagePacked)
		rw.Flush()
		log.Info(fmt.Sprintf("<--WRITE:\x1b[32mLength:%d\x1b[0m> ", len(p2pMessagePacked)))
	default: // other commands as subscription-type commands // this will move to pubsub
		listenerSendError(rw, nodeChain, errors.New("subscription-type command"), "-> Subscription type command , should send through pubsub", log)
		//processSubscription(log, fn, parameters)
		//result = []byte{'A'}
		return
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

package p2p

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	"github.com/gogo/protobuf/proto"
	p2phelp "github.com/libp2p/go-libp2p-core/helpers"
	mux "github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/network"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

const maxBytesRecieve = 3000

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
	defer p2phelp.FullClose(stream)
	log := l.log
	//log.Info("--- Start A New stream --")
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	//nodeChain := mode.ChainName()
	for {
		nodeChain := "local"
		req := make([]byte, maxBytesRecieve)
		reqLen, err := rw.Read(req)
		if err != nil {
			if err == io.EOF || err.Error() == mux.ErrReset.Error() {
				break
			}
			listenerSendError(rw, nodeChain, err, "-->READ", log)
			break
		}
		if reqLen < 1 {
			listenerSendError(rw, nodeChain, errors.New("length of byte recieved is less than 1"), "-->READ", log)
			break
		}
		reqChain, fn, parameters, err := UnPackP2PMessage(req[:reqLen])
		if err != nil {
			listenerSendError(rw, nodeChain, err, "-->Unpack", log)
			break
		}
		if len(reqChain) < 2 {
			listenerSendError(rw, nodeChain, errors.New("Invalid Chain"), "-->Unpack", log)
			break
		}
		if reqChain != nodeChain {
			listenerSendError(rw, nodeChain, errors.New("Different Chain"), "-->Chain", log)
			break
		}

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
				break
			}
			respParams := [][]byte{result}
			packed, err := PackP2PMessage(nodeChain, "I", respParams)
			if err != nil {
				listenerSendError(rw, nodeChain, err, "-->Query Server Information", log)
				break
			}
			rw.Write(packed)
			rw.Flush()
		case "N": // get block number
			blockNumber := blockheader.Height()
			result := make([]byte, 8)
			binary.BigEndian.PutUint64(result, blockNumber)
			respParams := [][]byte{result}
			packed, err := PackP2PMessage(nodeChain, "N", respParams)
			if err != nil {
				listenerSendError(rw, nodeChain, err, "-->Query Server Information", log)
				break
			}
			rw.Write(packed)
			rw.Flush()
		case "B": // get packed block
			if 1 != len(parameters) {
				err = fault.ErrMissingParameters
			} else if 8 == len(parameters[0]) { //it 8 or 6 ??
				result := storage.Pool.Blocks.Get(parameters[0])
				if nil == result {
					err = fault.ErrBlockNotFound
					listenerSendError(rw, nodeChain, err, "-->Query Block: block not found", log)
				}
				respParams := [][]byte{result}
				packed, err := PackP2PMessage(nodeChain, "B", respParams)
				if err != nil {
					listenerSendError(rw, nodeChain, err, "-->Query Block  Information", log)
					break
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
			} else if 8 == len(parameters[0]) {
				number := binary.BigEndian.Uint64(parameters[0])
				d, e := blockheader.DigestForBlock(number)
				var result []byte
				if nil == e {
					result = d[:]
				} else {
					err = e
				}
				if err != nil {
					listenerSendError(rw, nodeChain, err, "-->Query Blockhash  Information", log)
					return
				}
				respParams := [][]byte{result}
				packed, err := PackP2PMessage(nodeChain, "H", respParams)
				if err != nil {
					listenerSendError(rw, nodeChain, err, "-->Query Blockhash  Information", log)
					return
				}
				rw.Write(packed)
				rw.Flush()
			} else {
				err = fault.ErrBlockNotFound
				listenerSendError(rw, nodeChain, err, "-->Query Blockhash: invalid parameter", log)
			}
		case "R":
			nType, reqID, reqMaAddrs, timestamp, err := UnPackRegisterData(parameters)
			if err != nil {
				listenerSendError(rw, nodeChain, err, "-->RegData", log)
				return
			}
			if nType != "client" {
				log.Info(fmt.Sprintf("register:\x1b[32m Servant registered:%s\x1b[0m>", reqID.String()))
				announce.AddPeer(reqID, reqMaAddrs, timestamp) // id, listeners, timestam
			} else {
				log.Info(fmt.Sprintf("register:\x1b[32m Client registered:%s\x1b[0m>", reqID.String()))
			}
			log.Debug(fmt.Sprintf("register:\x1b[32m requestID  :%s\x1b[0m>", reqID.ShortString()))
			randPeerID, randListeners, randTs, err := announce.GetRandom(reqID)
			var randData [][]byte

			if nil != err || util.IDEqual(reqID, randPeerID) { // No Random Node sendback this Node
				randData, err = PackRegisterData(nodeChain, fn, nType, reqID, reqMaAddrs, time.Now())
				log.Debug(fmt.Sprintf("register:\x1b[32m No Random Request PeerID:%s\x1b[0m>", reqID.ShortString()))
			} else {
				randData, err = PackRegisterData(nodeChain, fn, "servant", randPeerID, randListeners, randTs)
				log.Debug(fmt.Sprintf("register:\x1b[32m Random  PeerID:%s\x1b[0m>", randPeerID.ShortString()))
			}

			p2pMessagePacked, err := proto.Marshal(&P2PMessage{Data: randData})
			if err != nil {
				listenerSendError(rw, nodeChain, err, "-><- Radom node", log)
				break
			}
			l.node.addRegister(reqID)
			_, err = rw.Write(p2pMessagePacked)
			rw.Flush()
			log.Debug(fmt.Sprintf("<--WRITE:\x1b[32mLength:%d\x1b[0m> ", len(p2pMessagePacked)))
		default: // other commands as subscription-type commands // this will move to pubsub
			listenerSendError(rw, nodeChain, errors.New("subscription-type command"), "-> Subscription type command , should send through pubsub", log)
			//processSubscription(log, fn, parameters)
			//result = []byte{'A'}
			break
		}
	}
}

func listenerSendError(sender *bufio.ReadWriter, chain string, err error, logPrefix string, log *logger.L) {
	errorMessage := [][]byte{[]byte(chain), []byte("E"), []byte(err.Error())}
	packedP2PMessage, err := proto.Marshal(&P2PMessage{Data: errorMessage})
	_, wErr := sender.Write(packedP2PMessage)
	if wErr != nil && log != nil {
		util.LogWarn(log, util.CoMagenta, fmt.Sprintf("%s Error:%v", logPrefix, wErr))
	}
	if log == nil {
		fmt.Printf("%s  \x1b[31mError:%v \x1b[0m\n", logPrefix, err)
	}
	sender.Flush()
}

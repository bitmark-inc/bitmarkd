package p2p

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
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

const maxBytesRecieve = 1024 * 100 //TODO: MaxBlock Size

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
	nodeChain := mode.ChainName()
	for {
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
			listenerSendError(rw, nodeChain, fault.DataLengthLessThanOne, "-->READ", log)
			break
		}
		reqChain, fn, parameters, err := UnPackP2PMessage(req[:reqLen])
		if err != nil {
			listenerSendError(rw, nodeChain, err, "-->Unpack", log)
			break
		}
		if len(reqChain) < 2 {
			listenerSendError(rw, nodeChain, fault.InvalidChain, "-->Unpack", log)
			break
		}
		if reqChain != nodeChain {
			listenerSendError(rw, nodeChain, fault.DifferentChain, "-->Chain", log)
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
			if 1 == len(parameters) {
				if 8 == len(parameters[0]) {
					result := storage.Pool.Blocks.Get(parameters[0])
					if nil == result {
						err = fault.BlockNotFound
						listenerSendError(rw, nodeChain, err, "-->Query Block:", log)
						break
					}
					respParams := [][]byte{result}
					packed, err := PackP2PMessage(nodeChain, "B", respParams)
					if err != nil {
						listenerSendError(rw, nodeChain, err, "-->Query Block:", log)
						break
					}
					rw.Write(packed)
					rw.Flush()
				} else {
					listenerSendError(rw, nodeChain, fault.BlockNotFound, "-->Query Block: ", log)
					break
				}
			} else {
				listenerSendError(rw, nodeChain, fault.MissingParameters, "-->Query Block:", log)
				break
			}
		case "H": // get block hash
			if 1 != len(parameters) {
				listenerSendError(rw, nodeChain, fault.MissingParameters, "-->Query Blockhash  Information", log)
				break
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
					break
				}
				respParams := [][]byte{result}
				packed, err := PackP2PMessage(nodeChain, "H", respParams)
				if err != nil {
					listenerSendError(rw, nodeChain, err, "-->Query Blockhash  Information", log)
					break
				}
				rw.Write(packed)
				rw.Flush()
			} else {
				err = fault.BlockNotFound
				listenerSendError(rw, nodeChain, err, "-->Query Blockhash: invalid parameter", log)
				break
			}
		case "R":
			nType, reqID, reqMaAddrs, timestamp, err := UnPackRegisterData(parameters)
			if err != nil {
				listenerSendError(rw, nodeChain, err, "-->RegData", log)
				break
			}
			if nType != "client" {
				announce.AddPeer(reqID, reqMaAddrs, timestamp) // id, listeners, timestam
			}
			randPeerID, randListeners, randTs, err := announce.GetRandom(reqID)
			var randData [][]byte
			var packError error
			if nil != err || util.IDEqual(reqID, randPeerID) { // No Random Node sendback this Node
				randData, packError = PackRegisterData(nodeChain, fn, nType, reqID, reqMaAddrs, time.Now())
				if packError != nil {
					listenerSendError(rw, nodeChain, packError, "-->Radom node", log)
					break
				}
			} else { //Get a Random Node
				randData, packError = PackRegisterData(nodeChain, fn, nType, randPeerID, randListeners, randTs)
				if packError != nil {
					listenerSendError(rw, nodeChain, packError, "-->Radom node", log)
					break
				}
			}
			p2pMessagePacked, err := proto.Marshal(&P2PMessage{Data: randData})
			if err != nil {
				listenerSendError(rw, nodeChain, err, "-><- Radom node", log)
				break
			}
			l.node.addRegister(reqID)
			_, err = rw.Write(p2pMessagePacked)
			util.LogError(log, util.CoReset, fmt.Sprintf("Register ID:%s Write Error:%v", reqID.ShortString(), err))
			rw.Flush()

		default: // other commands as subscription-type commands // this will move to pubsub
			listenerSendError(rw, nodeChain, fault.NotP2PCommand, "-> Subscription type command , should send through pubsub", log)
			//processSubscription(log, fn, parameters)
			//result = []byte{'A'}
		}
	}
}

func listenerSendError(sender *bufio.ReadWriter, chain string, err error, logPrefix string, log *logger.L) {
	errorMessage := [][]byte{[]byte(chain), []byte("E"), []byte(err.Error())}
	packedP2PMessage, err := proto.Marshal(&P2PMessage{Data: errorMessage})
	_, wErr := sender.Write(packedP2PMessage)
	if wErr != nil && log != nil {
		util.LogWarn(log, util.CoRed, fmt.Sprintf("%s Error:%v", logPrefix, wErr))
	}
	if log == nil {
		fmt.Printf("%s  \x1b[31mError:%v \x1b[0m\n", logPrefix, err)
	}
	sender.Flush()
}

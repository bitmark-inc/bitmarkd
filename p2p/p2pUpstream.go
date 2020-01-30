package p2p

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	"github.com/prometheus/common/log"
)

const (
	waitingRespTime    = 30 * time.Second
	readDefaultTimeout = 5 * time.Second
)

//UpdateVotingMetrics Register first and get info for voting metrics. This is an  efficient way to get data without create a new stream
func (n *Node) UpdateVotingMetrics(id peerlib.ID, metrics *MetricsPeersVoting) error {
	cctx, cancel := context.WithTimeout(context.Background(), waitingRespTime)
	defer cancel()
	s, err := n.Host.NewStream(cctx, id, "p2pstream")
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("UpdateVotingMetrics: Create new stream for ID:%v Error:%v", id.ShortString(), err))
		return err
	}
	defer s.Reset()
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	_, err = n.RequestRegister(id, s, rw)
	if err != nil {
		return err
	}
	height, err := n.QueryBlockHeight(id, s, rw)
	if err != nil {
		return err
	}
	digest, err := n.RemoteDigestOfHeight(id, height, s, rw)
	if err != nil {
		return err
	}
	metrics.SetMetrics(id, height, digest)
	return nil
}

//determineStreamRWerHelper
// to detemine stream and readwriter to use in request.
//streamCreated tells if newStream is created in the function.
//If it does , return stream may need to be reset in defer
func (n *Node) determineStreamRWerHelper(id peerlib.ID, s network.Stream, rw *bufio.ReadWriter) (stream network.Stream, readwriter *bufio.ReadWriter, streamCreated bool) {
	cctx, cancel := context.WithTimeout(context.Background(), waitingRespTime)
	defer cancel()
	if nil == s {
		newStream, newErr := n.Host.NewStream(cctx, id, "p2pstream")
		if newErr != nil {
			util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("fail to create a new stream: Error %v", newErr))
			return nil, nil, false
		}
		stream = newStream
		streamCreated = true
		readwriter = bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	} else {
		stream = s
		if readwriter != nil {
			readwriter = rw
		} else {
			readwriter = bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		}
	}
	//util.LogDebug(n.Log, util.CoGreen, fmt.Sprintf("determineStreamRWerHelper:  ID:%s", id.ShortString()))
	return
}

func (n *Node) readWithTimeout(readwriter *bufio.ReadWriter, buf []byte, timeout time.Duration) (size int, err error) {
	ch := make(chan bool)
	if nil == readwriter {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("readWithTimeout:readwriter is nil"))
		return size, errors.New("readwriter is nil")
	}
	if nil == buf {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("readWithTimeout:buf is nil"))
		return size, errors.New("buf is nil")
	}
	go func(reader *bufio.ReadWriter, readbuf []byte) {
		size, err = reader.Read(readbuf)
		ch <- true
	}(readwriter, buf)
	select {
	case <-ch:
		return
	case <-time.After(timeout):
		return 0, fault.ReadTimeout
	}
}

//RequestRegister this node register itself to the peer node. If stream is  nil, the function will create a new stream
func (n *Node) RequestRegister(id peerlib.ID, stream network.Stream, readwriter *bufio.ReadWriter) (network.Stream, error) {
	s, rw, created := n.determineStreamRWerHelper(id, stream, readwriter)
	if nil == s || nil == rw {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf(" RequestRegister: ID:%v No Useful Stream and ReadWriter", id.ShortString()))
		return nil, fault.StreamReadWriter
	}
	if created && s != nil {
		defer s.Reset()
	}
	nodeChain := mode.ChainName()
	p2pData, packError := PackRegisterData(nodeChain, "R", n.NodeType, n.Host.ID(), n.Announce, time.Now())
	if packError != nil {
		return nil, packError
	}

	p2pMsgPacked, err := proto.Marshal(&P2PMessage{Data: p2pData})
	if err != nil {
		return nil, err
	}
	_, err = rw.Write(p2pMsgPacked)
	if err != nil {
		n.Log.Error(err.Error())
		return nil, err
	}
	rw.Flush()
	// Wait for response
	resp := make([]byte, maxBytesRecieve)
	respLen, err := n.readWithTimeout(rw, resp, readDefaultTimeout)
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf(" RequestRegister: Read Error:%v ID:%v", err, id.ShortString()))
		return nil, err
	}
	if respLen < 1 {
		return nil, fault.DataLengthLessThanOne
	}
	chain, fn, parameters, err := UnPackP2PMessage(resp[:respLen])
	if err != nil {
		return nil, err
	}
	if chain != nodeChain {
		return nil, fault.DifferentChain
	}
	switch fn {
	case "E": //Register error
		errMessage := UnpackListenError(parameters)
		n.unRegister(id)
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf(" RequestRegister: Receive  E errorMessage:%v", errMessage))
		return nil, err
	case "R":
		nType, randID, randAddrs, randTs, err := UnPackRegisterData(parameters)
		if err != nil {
			n.unRegister(id)
			return nil, err
		}
		n.addRegister(id)
		if n.dnsPeerOnly == DnsOnly { // Do not add a random node when the only dns peer  is needed
			return s, nil
		}
		if !util.IDEqual(randID, id) {
			// peer return the info, register send. don't add into peer tree
			if nType != "client" { // client does not in the peer Tree
				announce.AddPeer(randID, randAddrs, randTs) // id, listeners, timestam
			}
			util.LogInfo(n.Log, util.CoGreen, fmt.Sprintf("<<--RequestRegister to  Successful:%v", id.ShortString()))
		}
	}
	return s, nil
}

//QueryBlockHeight query the  block height of peer node with given peerID. Put nil when you don't want to reuse stream
func (n *Node) QueryBlockHeight(id peerlib.ID, stream network.Stream, readwriter *bufio.ReadWriter) (uint64, error) {
	fn := "N"
	s, rw, created := n.determineStreamRWerHelper(id, stream, readwriter)
	if nil == s || nil == rw {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf(" Register:No Useful Stream and ReadWrite ID:%v", id.ShortString()))
		return 0, fault.StreamReadWriter
	}
	if created && s != nil {
		defer s.Reset()
	}
	if !n.IsRegister(id) { // stream has registered {
		_, regErr := n.RequestRegister(id, stream, readwriter)
		if regErr != nil {
			return 0, regErr
		}
	}
	packedP2PMsg, err := PackP2PMessage(mode.ChainName(), fn, [][]byte{})
	if err != nil {
		return 0, err
	}
	_, err = rw.Write(packedP2PMsg)
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf(" QueryBlockHeight:PeerID Write Error:%v", err))
		return 0, err
	}
	rw.Flush()
	respPacked := make([]byte, maxBytesRecieve)
	respLen, err := n.readWithTimeout(rw, respPacked, readDefaultTimeout)
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf(" QueryBlockHeight: Read Error:%v ID:%v", err, id.ShortString()))
		return 0, err
	}
	chain, fn, parameters, err := UnPackP2PMessage(respPacked[:respLen])
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight:UnPackP2PMessage  Error:%v", err))
		return 0, fmt.Errorf("invalid message response: %v", err)
	}

	if mode.ChainName() != chain {
		util.LogWarn(n.Log, util.CoRed, "QueryBlockHeight:Different Chain  Error")
		return 0, fault.DifferentChain
	}

	if fn == "" || len(parameters[0]) < 1 { // not enough  data return
		util.LogWarn(n.Log, util.CoRed, "QueryBlockHeight:Not valid parameters  Error")
		return 0, fmt.Errorf("Not valid parameters")
	}

	switch fn {
	case "E":
		errMessage := UnpackListenError(parameters)
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight Response Fail,  E msg:%v", errMessage))
		return 0, errMessage
	case "N":
		if 8 != len(parameters[0]) {
			util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight Response Success, but invalid response  param:%q", parameters[0]))
			return 0, fmt.Errorf("highestBlock:  invalid response: %q", parameters[0])
		}
		height := binary.BigEndian.Uint64(parameters[0])
		util.LogDebug(n.Log, util.CoGreen, fmt.Sprintf("<<--QueryBlockHeight ID:%v Success,", id.ShortString()))
		return height, nil
	default:
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight unexpected response:%v", fn))
		return 0, fmt.Errorf("unexpected response: %v", fn)
	}
}

//RemoteDigestOfHeight - fetch block digest from a specific block number
func (n *Node) RemoteDigestOfHeight(id peerlib.ID, blockNumber uint64, stream network.Stream, readwriter *bufio.ReadWriter) (blockdigest.Digest, error) {
	s, rw, created := n.determineStreamRWerHelper(id, stream, readwriter)
	if nil == s || nil == rw {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("Register:No Useful Stream and ReadWrite ID:%v", id.ShortString()))
		return blockdigest.Digest{}, fault.StreamReadWriter
	}
	if created && s != nil {
		defer s.Reset()
	}
	nodeChain := mode.ChainName()
	if !n.IsRegister(id) { // stream has registered {
		_, regErr := n.RequestRegister(id, stream, readwriter)
		if regErr != nil {
			return blockdigest.Digest{}, regErr
		}
	}
	packedData, packError := PackQueryDigestData(nodeChain, blockNumber)
	if packError != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf(" RemoteDigestOfHeight: Read Error:%v ID:%v stream:%v readriter:%v", packError, id.ShortString(), s, rw))
		return blockdigest.Digest{}, packError
	}
	p2pMsgPacked, err := proto.Marshal(&P2PMessage{Data: packedData})
	if nil != err {
		return blockdigest.Digest{}, err
	}
	_, err = rw.Write(p2pMsgPacked)
	if nil != err {
		return blockdigest.Digest{}, err
	}
	rw.Flush()

	respPacked := make([]byte, maxBytesRecieve)
	respLen, err := n.readWithTimeout(rw, respPacked, readDefaultTimeout)
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf(" RemoteDigestOfHeight: Read Error:%v ID:%v stream:%v readriter:%v", err, id.ShortString(), s, rw))
		return blockdigest.Digest{}, err
	}
	chain, fn, parameters, err := UnPackP2PMessage(respPacked[:respLen])
	if mode.ChainName() != chain {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RemoteDigestOfHeight:Different Chain ID:%v", id.ShortString()))
		return blockdigest.Digest{}, err
	}

	if fn == "" || len(parameters) != 1 { // not enough  data return
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RemoteDigestOfHeight:Invalid parameters ID:%v", id.ShortString()))
		return blockdigest.Digest{}, err
	}

	switch string(fn) {
	case "E":
		errMessage := UnpackListenError(parameters)
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RemoteDigestOfHeight: Response Fail,  E msg:%v ID:%v", errMessage, id.ShortString()))
		return blockdigest.Digest{}, fault.BlockNotFound
	case "H":
		d := blockdigest.Digest{}
		if blockdigest.Length == len(parameters[0]) {
			err := blockdigest.DigestFromBytes(&d, parameters[0])
			if err != nil {
				util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RemoteDigestOfHeight: Response Success but error:%v digest ID:%v  hash:%q ", err, id.ShortString(), d))
			}
			util.LogDebug(n.Log, util.CoGreen, fmt.Sprintf("<<--RemoteDigestOfHeight: Success ID:%v", id.ShortString()))

			return d, err
		}
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RemoteDigestOfHeight:  ID:%v Digest length does not match", id.ShortString()))

	default:
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RemoteDigestOfHeight:  unexpected response:%v  ID%v", fn, id.ShortString()))
	}
	return blockdigest.Digest{}, fault.InvalidPeerResponse
}

// GetBlockData - fetch block data from a specific block number
func (n *Node) GetBlockData(id peerlib.ID, blockNumber uint64, stream network.Stream, readwriter *bufio.ReadWriter) ([]byte, error) {
	s, rw, created := n.determineStreamRWerHelper(id, stream, readwriter)
	if nil == s || nil == rw {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("Register:No Useful Stream and ReadWrite ID:%v", id.ShortString()))
		return nil, fault.StreamReadWriter
	}
	if created && s != nil {
		defer s.Reset()
	}
	if !n.IsRegister(id) { // stream has registered {
		_, regErr := n.RequestRegister(id, stream, readwriter)
		if regErr != nil {
			return nil, regErr
		}
	}
	nodeChain := mode.ChainName()
	packedData, packError := PackQueryBlockData(nodeChain, blockNumber)
	if packError != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: PackQueryBlockData  Error:%v ID:%v", packError, id.ShortString()))
		return nil, packError
	}
	p2pMsgPacked, err := proto.Marshal(&P2PMessage{Data: packedData})
	if nil != err {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: Marshal  Error:%v ID:%v", err, id.ShortString()))
		return nil, err
	}
	rw.Write(p2pMsgPacked)
	rw.Flush()
	respPacked := make([]byte, maxBytesRecieve)
	respLen, err := n.readWithTimeout(rw, respPacked, readDefaultTimeout)
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf(" GetBlockData: Read Error:%v ID:%v", err, id.ShortString()))
		return nil, err
	}
	chain, fn, parameters, err := UnPackP2PMessage(respPacked[:respLen])
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: UnPackP2PMessage  Error:%v ID:%v", err, id.ShortString()))
		return nil, fmt.Errorf("invalid message response: %v", err)
	}
	if mode.ChainName() != chain {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: Different Chain ErrorID:%v", id.ShortString()))
		return nil, fault.DifferentChain
	}

	if 1 != len(parameters) {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData:   Error:%v  ID:%v parameter length=%d NOT equal 1", fault.InvalidPeerResponse, id.ShortString(), len(parameters)))
		return nil, fault.InvalidPeerResponse
	}

	switch string(fn) {
	case "E":
		errMessage := UnpackListenError(parameters)
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: Response Fail,  E msg:%v ID:%v", errMessage, id.ShortString()))
		return nil, fault.BlockNotFound
	case "B":
		util.LogDebug(n.Log, util.CoGreen, fmt.Sprintf("<<--GetBlockData Success! ID:%v", id.ShortString()))
		return parameters[0], nil
	default:
	}
	return nil, fault.InvalidPeerResponse
}

//PushMessageBus  send the CommandBus Message to the peer with given ID
func (n *Node) PushMessageBus(item BusMessage, id peerlib.ID, stream network.Stream, readwriter *bufio.ReadWriter) error {
	s, rw, created := n.determineStreamRWerHelper(id, stream, readwriter)
	if nil == s || nil == rw {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("Register:No Useful Stream and ReadWrite ID:%v", id.ShortString()))
		return fault.StreamReadWriter
	}
	if created && s != nil {
		defer s.Reset()
	}
	if !n.IsRegister(id) { // stream has registered {
		_, regErr := n.RequestRegister(id, stream, readwriter)
		if regErr != nil {
			return regErr
		}
	}
	packedP2PMsg, err := PackP2PMessage(mode.ChainName(), item.Command, item.Parameters)
	if err != nil {
		return err
	}
	_, err = rw.Write(packedP2PMsg)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	rw.Flush()
	respPacked := make([]byte, maxBytesRecieve)
	_, err = n.readWithTimeout(rw, respPacked, readDefaultTimeout)
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf(" PushMessageBus: Read Error:%v ID:%v", err, id.ShortString()))
		return err
	}
	chain, command, parameters, err := UnPackP2PMessage(respPacked)
	if err != nil {
		return fmt.Errorf("invalid message response: %v", err)
	}

	if mode.ChainName() != chain {
		return fault.DifferentChain
	}

	if command == "" || len(parameters[0]) < 1 { // not enough  data return
		return fmt.Errorf("Not enough data")
	}
	switch command {
	case "E":
		return fmt.Errorf("rpc error response: %q", parameters[0])
	case item.Command:
		util.LogDebug(n.Log, util.CoGreen, fmt.Sprintf("<<--push: client: %s complete: %q", id.ShortString(), parameters[0]))
		return nil
	default:
		return fmt.Errorf("rpc unexpected response: %q", parameters[0])
	}
}

//QueryPeerInfo query peer info
func (n *Node) QueryPeerInfo(id peerlib.ID, stream *network.Stream) error {
	var s network.Stream
	nodeChain := mode.ChainName()
	cctx, cancel := context.WithTimeout(context.Background(), waitingRespTime)
	defer cancel()
	if nil == stream {
		createStream, newErr := n.Host.NewStream(cctx, id, "p2pstream")
		if newErr != nil {
			return newErr
		}
		s = createStream
		defer s.Reset()
	} else {
		s = *stream
	}
	p2pMsgPacked, err := PackP2PMessage(nodeChain, "I", [][]byte{})
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryPeerInfo: PackP2PMessageError:%v ID:%v", err, id.ShortString()))
		return err
	}
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	rw.Write([]byte(p2pMsgPacked))
	rw.Flush()
	return nil
}

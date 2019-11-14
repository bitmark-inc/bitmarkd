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

//UpdateVotingMetrics Register first and get info for voting metrics. This is an  efficient way to get data without create a new stream
func (n *Node) UpdateVotingMetrics(id peerlib.ID, metrics *MetricsPeersVoting) error {
	s, err := n.Host.NewStream(context.Background(), id, "p2pstream")
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("UpdateVotingMetrics: Create new stream error Error:%v", err))
		n.Log.Warn(err.Error())
		return err
	}
	defer s.Reset()
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	_, err = n.RequestRegister(id, &s, rw)
	if err != nil {
		return err
	}
	height, err := n.QueryBlockHeight(id, &s, rw)
	if err != nil {
		return err
	}
	digest, err := n.RemoteDigestOfHeight(id, height, &s, rw)
	if err != nil {
		return err
	}
	metrics.setMetrics(id, height, digest)
	return nil
}

//determineStreamRWerHelper  to detemine stream and readwriter to use in request. streamCreated tells if newStream is created in the function. If it does , return stream may need to be reset in defer
func (n *Node) determineStreamRWerHelper(id peerlib.ID, s *network.Stream, rw *bufio.ReadWriter) (stream *network.Stream, readwriter *bufio.ReadWriter, streamCreated bool) {
	if nil == s {
		createStream, newErr := n.Host.NewStream(context.Background(), id, "p2pstream")
		if newErr != nil {
			return nil, nil, false
		}
		stream = &createStream
		streamCreated = true
		readwriter = bufio.NewReadWriter(bufio.NewReader(*stream), bufio.NewWriter(*stream))
	} else {
		stream = s
		if readwriter != nil {
			readwriter = rw
		} else {
			readwriter = bufio.NewReadWriter(bufio.NewReader(*stream), bufio.NewWriter(*stream))
		}
	}
	return
}

//RequestRegister this node register itself to the peer node. If stream is  nil, the function will create a new stream
func (n *Node) RequestRegister(id peerlib.ID, stream *network.Stream, readwriter *bufio.ReadWriter) (*network.Stream, error) {
	s, rw, created := n.determineStreamRWerHelper(id, stream, readwriter)
	if nil == s && nil == rw {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RequestRegister:No Useful Stream and ReadWrite ID:%v", id.ShortString()))
		return nil, errors.New("No Useful Stream and ReadWriter")
	}
	if created && s != nil {
		defer (*s).Reset()
	}
	//nodeChain := mode.ChainName()
	nodeChain := "local"
	p2pData, err := PackRegisterData(nodeChain, "R", n.NodeType, n.Host.ID(), n.Announce, time.Now())
	if err != nil {
		return nil, err
	}
	p2pMsgPacked, err := proto.Marshal(&P2PMessage{Data: p2pData})
	if err != nil {
		return nil, err
	}
	n.Lock()
	_, err = rw.Write(p2pMsgPacked)
	if err != nil {
		n.Unlock()
		n.Log.Error(err.Error())
		return nil, err
	}
	rw.Flush()
	// Wait for response
	resp := make([]byte, maxBytesRecieve)
	respLen, err := rw.Read(resp)
	n.Unlock()
	if err != nil {
		return nil, err
	}
	if respLen < 1 {
		return nil, errors.New("length of byte recieved is less than 1")
	}
	chain, fn, parameters, err := UnPackP2PMessage(resp[:respLen])
	if err != nil {
		return nil, err
	}
	if chain != nodeChain {
		return nil, errors.New("Different chain")
	}
	switch fn {
	case "E": //Register error
		errMessage, _ := UnpackListenError(parameters)
		n.delRegister(id)
		n.Log.Warn(fmt.Sprintf("\x1b[31mRequestRegister Error:%s\x1b[0m>", errMessage))
		return nil, err
	case "R":
		nType, randID, randAddrs, randTs, err := UnPackRegisterData(parameters)
		if err != nil {
			return nil, err
		}
		if !util.IDEqual(randID, id) {
			// peer return the info, register send. don't add into peer tree
			n.addRegister(id)
			if nType != "client" { // client does not in the peer Tree
				announce.AddPeer(randID, randAddrs, randTs) // id, listeners, timestam
			}
			util.LogInfo(n.Log, util.CoGreen, fmt.Sprintf("Register Successful:%v", id.ShortString()))
		}
	}
	return s, nil
}

//QueryBlockHeight query the  block height of peer node with given peerID. Put nil when you don't want to reuse stream
func (n *Node) QueryBlockHeight(id peerlib.ID, stream *network.Stream, readwriter *bufio.ReadWriter) (uint64, error) {
	fn := "N"
	s, rw, created := n.determineStreamRWerHelper(id, stream, readwriter)
	if nil == s && nil == rw {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("Register:No Useful Stream and ReadWrite ID:%v", id.ShortString()))
		return 0, errors.New("No Useful Stream and ReadWriter")
	}
	if created && s != nil {
		defer (*s).Reset()
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
	n.Lock()
	_, err = rw.Write(packedP2PMsg)
	if err != nil {
		n.Unlock()
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight:PeerID Write Error:%v", err))
		return 0, err
	}
	rw.Flush()
	respPacked := make([]byte, maxBytesRecieve)
	_, err = rw.Read(respPacked) //Expected data :  chain, fn, block-height
	if err != nil {
		n.Unlock()
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight:Response  Error:%v", err))
		return 0, err
	}
	n.Unlock()
	chain, fn, parameters, err := UnPackP2PMessage(respPacked)
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight:UnPackP2PMessage  Error:%v", err))
		return 0, fmt.Errorf("invalid message response: %v", err)
	}

	if mode.ChainName() != chain {
		util.LogWarn(n.Log, util.CoRed, "QueryBlockHeight:Different Chain  Error")
		return 0, fmt.Errorf("different chain")
	}

	if fn == "" || len(parameters[0]) < 1 { // not enough  data return
		util.LogWarn(n.Log, util.CoRed, "QueryBlockHeight:Not valid parameters  Error")
		return 0, fmt.Errorf("Not valid parameters")
	}

	switch fn {
	case "E":
		errMessage, _ := UnpackListenError(parameters)
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight Response Fail,  E msg:%v", errMessage))
		return 0, errMessage
	case "N":
		if 8 != len(parameters[0]) {
			util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight Response Success, but invalid response  param:%q", parameters[0]))
			return 0, fmt.Errorf("highestBlock:  invalid response: %q", parameters[0])
		}
		height := binary.BigEndian.Uint64(parameters[0])
		util.LogInfo(n.Log, util.CoGreen, fmt.Sprintf("QueryBlockHeight ID:%v Success,", id.ShortString()))
		return height, nil
	default:
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight unexpected response:%v", fn))
		return 0, fmt.Errorf("unexpected response: %v", fn)
	}

	return 0, errors.New("Peer :" + id.Pretty() + " is not registered")
}

//RemoteDigestOfHeight - fetch block digest from a specific block number
func (n *Node) RemoteDigestOfHeight(id peerlib.ID, blockNumber uint64, stream *network.Stream, readwriter *bufio.ReadWriter) (blockdigest.Digest, error) {
	s, rw, created := n.determineStreamRWerHelper(id, stream, readwriter)
	if nil == s && nil == rw {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("Register:No Useful Stream and ReadWrite ID:%v", id.ShortString()))
		return blockdigest.Digest{}, errors.New("No Useful Stream and ReadWriter")
	}
	if created && s != nil {
		defer (*s).Reset()
	}
	//nodeChain := mode.ChainName()
	nodeChain := "local"
	if !n.IsRegister(id) { // stream has registered {
		_, regErr := n.RequestRegister(id, stream, readwriter)
		if regErr != nil {
			return blockdigest.Digest{}, regErr
		}
	}
	packedData, err := PackQueryDigestData(nodeChain, blockNumber)
	if err != nil {
		n.Log.Warn(err.Error())
		return blockdigest.Digest{}, err
	}
	p2pMsgPacked, err := proto.Marshal(&P2PMessage{Data: packedData})
	if nil != err {
		return blockdigest.Digest{}, err
	}
	n.Lock()
	_, err = rw.Write(p2pMsgPacked)
	if nil != err {
		n.Unlock()
		return blockdigest.Digest{}, err
	}
	rw.Flush()

	respPacked := make([]byte, maxBytesRecieve)
	_, err = rw.Read(respPacked)
	n.Unlock()
	chain, fn, parameters, err := UnPackP2PMessage(respPacked)
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RemoteDigestOfHeight:UnPackP2PMessage Error:%v", err))
		return blockdigest.Digest{}, err
	}

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
		errMessage, _ := UnpackListenError(parameters)
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RemoteDigestOfHeight: Response Fail,  E msg:%v ID:%v", errMessage, id.ShortString()))
		return blockdigest.Digest{}, fault.ErrorFromRunes(parameters[0])
	case "H":
		d := blockdigest.Digest{}
		if blockdigest.Length == len(parameters[0]) {
			err := blockdigest.DigestFromBytes(&d, parameters[0])
			if err != nil {
				util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RemoteDigestOfHeight: Response Success but error:%v digest ID:%v  hash:%q ", err, id.ShortString(), d))
			}
			util.LogInfo(n.Log, util.CoGreen, fmt.Sprintf("RemoteDigestOfHeight: Success! digest ID:%v  hash:%q ", id.ShortString(), d))
			return d, err
		}
		util.LogInfo(n.Log, util.CoGreen, fmt.Sprintf("RemoteDigestOfHeight: Success ID:%v", id.ShortString()))
	default:
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("RemoteDigestOfHeight:  unexpected response:%v  ID%v", fn, id.ShortString()))
	}
	return blockdigest.Digest{}, fault.ErrInvalidPeerResponse
}

// GetBlockData - fetch block data from a specific block number
func (n *Node) GetBlockData(id peerlib.ID, blockNumber uint64, stream *network.Stream, readwriter *bufio.ReadWriter) ([]byte, error) {
	s, rw, created := n.determineStreamRWerHelper(id, stream, readwriter)
	if nil == s && nil == rw {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("Register:No Useful Stream and ReadWrite ID:%v", id.ShortString()))
		return nil, errors.New("No Useful Stream and ReadWriter")
	}
	if created && s != nil {
		defer (*s).Reset()
	}
	if !n.IsRegister(id) { // stream has registered {
		_, regErr := n.RequestRegister(id, stream, readwriter)
		if regErr != nil {
			return nil, regErr
		}
	}
	//nodeChain := mode.ChainName()
	nodeChain := "local"

	packedData, err := PackQueryBlockData(nodeChain, blockNumber)
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: PackQueryBlockData  Error:%v ID:%v", err, id.ShortString()))
		return nil, err
	}
	p2pMsgPacked, err := proto.Marshal(&P2PMessage{Data: packedData})
	if nil != err {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: Marshal  Error:%v ID:%v", err, id.ShortString()))
		return nil, err
	}
	n.Lock()
	rw.Write(p2pMsgPacked)
	rw.Flush()
	respPacked := make([]byte, maxBytesRecieve)
	_, err = rw.Read(respPacked) //Expected data :  chain, fn, block
	n.Unlock()
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: Read  Error:%v ID:%v", err, id.ShortString()))
		return nil, err
	}
	chain, fn, parameters, err := UnPackP2PMessage(respPacked)
	if err != nil {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: UnPackP2PMessage  Error:%v ID:%v", err, id.ShortString()))
		return nil, fmt.Errorf("invalid message response: %v", err)
	}
	if mode.ChainName() != chain {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: Different Chain ErrorID:%v", id.ShortString()))
		return nil, fmt.Errorf("different chain")
	}

	if 1 != len(parameters) {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData:   Error:%v ID:%v", fault.ErrInvalidPeerResponse, id.ShortString()))
		return nil, fault.ErrInvalidPeerResponse
	}

	switch string(fn) {
	case "E":
		errMessage, _ := UnpackListenError(parameters)
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("GetBlockData: Response Fail,  E msg:%v ID:%v", errMessage, id.ShortString()))
		return nil, fault.ErrorFromRunes(parameters[0])
	case "B":
		util.LogInfo(n.Log, util.CoGreen, fmt.Sprintf("GetBlockData Success! ID:%v", id.ShortString()))
		return parameters[0], nil
	default:
	}
	return nil, fault.ErrInvalidPeerResponse
}

//PushMessageBus  send the CommandBus Message to the peer with given ID
func (n *Node) PushMessageBus(item BusMessage, id peerlib.ID, stream *network.Stream, readwriter *bufio.ReadWriter) error {
	s, rw, created := n.determineStreamRWerHelper(id, stream, readwriter)
	if nil == s && nil == rw {
		util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("Register:No Useful Stream and ReadWrite ID:%v", id.ShortString()))
		return errors.New("No Useful Stream and ReadWriter")
	}
	if created && s != nil {
		defer (*s).Reset()
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
	n.Lock()
	sendLen, err := rw.Write(packedP2PMsg)
	if err != nil {
		n.Unlock()
		log.Error(err.Error())
		return err
	}
	log.Debug(fmt.Sprintf("WRITE:\x1b[32mLength:%d\x1b[0m> ", sendLen))
	rw.Flush()
	respPacked := make([]byte, maxBytesRecieve)
	respLen, err := rw.Read(respPacked) //Expected data chain, fn, block-height
	n.Unlock()
	log.Info(fmt.Sprintf("%s RECIEVED:\x1b[32m%d\x1b[0m> ", "listener", respLen))
	if err != nil {
		return err
	}
	chain, command, parameters, err := UnPackP2PMessage(respPacked)
	if err != nil {
		return fmt.Errorf("invalid message response: %v", err)
	}

	if mode.ChainName() != chain {
		return fmt.Errorf("different chain")
	}

	if command == "" || len(parameters[0]) < 1 { // not enough  data return
		return fmt.Errorf("Not enough data")
	}
	switch command {
	case "E":
		return fmt.Errorf("rpc error response: %q", parameters[0])
	case item.Command:
		log.Debugf("push: client: %s complete: %q", id.Pretty(), parameters[0])
		return nil
	default:
		return fmt.Errorf("rpc unexpected response: %q", parameters[0])
	}
	return errors.New("no register stream")
}

//QueryPeerInfo query peer info
func (n *Node) QueryPeerInfo(id peerlib.ID, stream *network.Stream) error {
	//nodeChain := mode.ChainName()
	var s network.Stream
	nodeChain := "local"
	if nil == stream {
		createStream, newErr := n.Host.NewStream(context.Background(), id, "p2pstream")
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

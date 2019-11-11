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

//Register this node register itself to the peerInfo node and  add the stream to RegisterStream
func (n *Node) Register(peerInfo *peerlib.AddrInfo) (network.Stream, error) {
	s, err := n.Host.NewStream(context.Background(), peerInfo.ID, "p2pstream")
	if err != nil {
		n.Log.Warn(err.Error())
		return nil, err
	}
	defer s.Reset()
	//nodeChain := mode.ChainName()
	nodeChain := "local"
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
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
	n.Log.Debug(fmt.Sprintf("-->Register RECIEVED:\x1b[32m%d\x1b[0m> ", respLen))
	if err != nil {
		return nil, err
	}
	if respLen < 1 {
		return nil, errors.New("length of byte recieved is less than 1")
	}
	chain, fn, parameters, err := UnPackP2PMessage(resp[:respLen])
	n.Log.Debug(fmt.Sprintf("-->>Register RECIEVED:\x1b[32mLength:%d\x1b[0m> ", respLen))
	if err != nil {
		return nil, err
	}
	if chain != nodeChain {
		return nil, errors.New("Different chain")
	}
	switch fn {
	case "E": //Register error
		errMessage, _ := UnpackListenError(parameters)
		n.Log.Warn(fmt.Sprintf("\x1b[31mRegister Error:%s\x1b[0m>", errMessage))
		s.Close()
		return nil, err
	case "R":
		nType, randID, randAddrs, randTs, err := UnPackRegisterData(parameters)
		if err != nil {
			s.Close()
			return nil, err
		}
		if !util.IDEqual(randID, peerInfo.ID) {
			// peer return the info, register send. don't add into peer tree
			n.addRegister(peerInfo.ID)
			if nType != "client" { // client does not in the peer Tree
				announce.AddPeer(randID, randAddrs, randTs) // id, listeners, timestam
			}
			n.Log.Infof("--> \x1b[32mRegister Successful:%v\x1b[0m", peerInfo.ID.String())
		}
	}
	return s, nil
}

//QueryBlockHeight query the  block height of peer node with given peerID
func (n *Node) QueryBlockHeight(peerID peerlib.ID) (uint64, error) {
	log := n.Log
	s, err := n.Host.NewStream(context.Background(), peerID, "p2pstream")
	if err != nil {
		return 0, err
	}
	defer s.Reset()
	fn := "N"
	if n.IsRegister(peerID) { // stream has registered
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		packedP2PMsg, err := PackP2PMessage(mode.ChainName(), fn, [][]byte{})
		if err != nil {
			return 0, err
		}
		n.Lock()
		sendLen, err := rw.Write(packedP2PMsg)
		if err != nil {
			n.Unlock()
			util.LogWarn(n.Log, util.CoRed, fmt.Sprintf("QueryBlockHeight:PeerID Write Error:%v", err))
			return 0, err
		}
		rw.Flush()
		log.Debug(fmt.Sprintf("<<-- QueryBlockHeight WRITE:\x1b[32mLength:%d\x1b[0m> ", sendLen))
		respPacked := make([]byte, maxBytesRecieve)
		respLen, err := rw.Read(respPacked) //Expected data :  chain, fn, block-height
		if err != nil {
			n.Unlock()
			n.Log.Warnf("\x1b[30m QueryBlockHeight:Response Error:%s\x1b[0m> ", err.Error())
			return 0, err
		}
		n.Unlock()
		log.Debug(fmt.Sprintf("-->>QueryBlockHeight RECIEVED:\x1b[32m%d\x1b[0m> ", respLen))

		chain, fn, parameters, err := UnPackP2PMessage(respPacked)
		if err != nil {
			return 0, fmt.Errorf("invalid message response: %v", err)
		}

		if mode.ChainName() != chain {
			return 0, fmt.Errorf("different chain")
		}

		if fn == "" || len(parameters[0]) < 1 { // not enough  data return
			return 0, fmt.Errorf("Not valid parameters")
		}

		switch fn {
		case "E":
			errMessage, _ := UnpackListenError(parameters)
			log.Warn(fmt.Sprintf("\x1b[30m queryBlockHeight Error:%s\x1b[0m>", errMessage))
			return 0, errMessage
		case "N":
			if 8 != len(parameters[0]) {
				return 0, fmt.Errorf("highestBlock:  invalid response: %q", parameters[0])
			}
			height := binary.BigEndian.Uint64(parameters[0])
			return height, nil
		default:
			return 0, fmt.Errorf("unexpected response: %q", fn)
		}
	}
	return 0, errors.New("Peer :" + peerID.Pretty() + " is not registered")
}

//RemoteDigestOfHeight - fetch block digest from a specific block number
func (n *Node) RemoteDigestOfHeight(peerID peerlib.ID, blockNumber uint64) (blockdigest.Digest, error) {
	s, err := n.Host.NewStream(context.Background(), peerID, "p2pstream")
	if err != nil {
		n.Log.Warn(err.Error())
		return blockdigest.Digest{}, err
	}
	defer s.Reset()

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	//nodeChain := mode.ChainName()
	nodeChain := "local"

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
		log.Warn(fmt.Sprintf("\x1b[30m RemoteDigestOfHeight UnPackP2PMessage Error:%s\x1b[0m>", err))
		return blockdigest.Digest{}, err
	}

	if mode.ChainName() != chain {
		log.Warn(fmt.Sprintf("\x1b[30m RemoteDigestOfHeight chain different Error:%s\x1b[0m>", err))
		return blockdigest.Digest{}, err
	}

	if fn == "" || len(parameters) != 1 { // not enough  data return
		n.Log.Warn(fmt.Sprintf("\x1b[31mRemoteDigestOfHeight fn !empty || len(parameters[0]) != 1  len:%d\x1b[0m>", len(parameters[0])))
		return blockdigest.Digest{}, err
	}

	switch string(fn) {
	case "E":
		n.Log.Warn(fmt.Sprintf("\x1b[31mRemoteDigestOfHeight fn !empty || len(parameters[0]) != 1  len:%d\x1b[0m>", len(parameters[0])))
		return blockdigest.Digest{}, fault.ErrorFromRunes(parameters[0])
	case "H":
		d := blockdigest.Digest{}
		if blockdigest.Length == len(parameters[0]) {
			err := blockdigest.DigestFromBytes(&d, parameters[0])
			return d, err
		}
	default:
		n.Log.Warn(fmt.Sprintf("\x1b[31mdefaul: fn :%s\x1b[0m>", fn))
	}
	return blockdigest.Digest{}, fault.ErrInvalidPeerResponse
}

// GetBlockData - fetch block data from a specific block number
func (n *Node) GetBlockData(peerID peerlib.ID, blockNumber uint64) ([]byte, error) {
	s, err := n.Host.NewStream(context.Background(), peerID, "p2pstream")
	if err != nil {
		n.Log.Warn(err.Error())
		return nil, err
	}
	defer s.Reset()
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	//nodeChain := mode.ChainName()
	nodeChain := "local"

	packedData, err := PackQueryBlockData(nodeChain, blockNumber)
	if err != nil {
		return nil, err
	}
	p2pMsgPacked, err := proto.Marshal(&P2PMessage{Data: packedData})
	if nil != err {
		return nil, err
	}
	n.Lock()
	rw.Write(p2pMsgPacked)
	rw.Flush()
	respPacked := make([]byte, maxBytesRecieve)
	_, err = rw.Read(respPacked) //Expected data :  chain, fn, block
	n.Unlock()
	if err != nil {
		return nil, err
	}
	chain, fn, parameters, err := UnPackP2PMessage(respPacked)
	if err != nil {
		return nil, fmt.Errorf("invalid message response: %v", err)
	}
	if mode.ChainName() != chain {
		return nil, fmt.Errorf("different chain")
	}

	if 1 != len(parameters) {
		return nil, fault.ErrInvalidPeerResponse
	}

	switch string(fn) {
	case "E":
		return nil, fault.ErrorFromRunes(parameters[0])
	case "B":
		return parameters[0], nil
	default:
	}
	return nil, fault.ErrInvalidPeerResponse
}

//PushMessageBus  send the CommandBus Message to the peer with given ID
func (n *Node) PushMessageBus(item BusMessage, peerID peerlib.ID) error {
	log := n.Log
	s, err := n.Host.NewStream(context.Background(), peerID, "p2pstream")
	if err != nil {
		n.Log.Warn(err.Error())
		return err
	}
	defer s.Reset()
	if n.IsRegister(peerID) { // stream has registered
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
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
			log.Debugf("push: client: %s complete: %q", peerID.Pretty(), parameters[0])
			return nil
		default:
			return fmt.Errorf("rpc unexpected response: %q", parameters[0])
		}

	}
	return errors.New("no register stream")
}

//QueryPeerInfo query peer info
func (n *Node) QueryPeerInfo(id peerlib.ID) error {
	//nodeChain := mode.ChainName()
	nodeChain := "local"
	n.Log.Info(fmt.Sprintf("Enter\x1b[32mQueryPeerInfo:%s\x1b[0m>", nodeChain))
	s, err := n.Host.NewStream(context.Background(), n.Host.ID(), "p2pstream")
	if err != nil {
		n.Log.Error(err.Error())
		return err
	}
	defer s.Reset()
	if s != nil {
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		//p2pMsgPacked, err := PackP2PMessage(nodeChain, "I", [][]byte{})

		rw.Write([]byte("p2pMsgPacked"))
		rw.Flush()
	}
	return nil
}

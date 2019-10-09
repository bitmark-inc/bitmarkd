package p2p

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

//Register this node register itself to the peerInfo node and  add the stream to RegisterStream
func (n *Node) Register(peerInfo *peerlib.AddrInfo) (network.Stream, error) {
	s, err := n.Host.NewStream(context.Background(), peerInfo.ID, "p2pstream")
	//nodeChain := mode.ChainName()
	nodeChain := "local"
	n.Log.Info("---- start to register ---")
	if err != nil {
		n.Log.Warn(err.Error())
		return nil, err
	}
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	p2pData, err := PackRegisterData(nodeChain, "R", n.NodeType, n.Host.ID(), n.Announce, time.Now())
	if err != nil {
		return nil, err
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
	n.Log.Debug(fmt.Sprintf("WRITE:\x1b[32mLength:%d\x1b[0m> ", len(p2pMsgPacked)))
	rw.Flush()
	// Wait for response
	resp := make([]byte, maxBytesRecieve)
	respLen, err := rw.Read(resp)
	n.Log.Debug(fmt.Sprintf("%s RECIEVED:\x1b[32m%d\x1b[0m> ", "listener", respLen))
	if err != nil {
		return nil, err
	}
	if respLen < 1 {
		return nil, errors.New("length of byte recieved is less than 1")
	}
	chain, fn, parameters, err := UnPackP2PMessage(resp[:respLen])
	n.Log.Debug(fmt.Sprintf("RECIEVED:\x1b[32mLength:%d\x1b[0m> ", respLen))
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
			n.addToRegister(peerInfo.ID, s)
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
	n.RLock()
	fn := "N"
	if s, ok := n.RegisterStream[peerID.Pretty()]; ok { // stream has registered
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		packedP2PMsg, err := PackP2PMessage(mode.ChainName(), fn, [][]byte{})
		if err != nil {
			return 0, err
		}
		sendLen, err := rw.Write(packedP2PMsg)
		if err != nil {
			n.Log.Error(err.Error())
			return 0, err
		}
		log.Debug(fmt.Sprintf("WRITE:\x1b[32mLength:%d\x1b[0m> ", sendLen))
		rw.Flush()
		respPacked := make([]byte, maxBytesRecieve)
		respLen, err := rw.Read(respPacked) //Expected data :  chain, fn, block-height
		log.Info(fmt.Sprintf("%s RECIEVED:\x1b[32m%d\x1b[0m> ", "listener", respLen))
		if err != nil {
			return 0, err
		}

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
			log.Warn(fmt.Sprintf("\x1b[31m queryBlockHeight Error:%s\x1b[0m>", errMessage))
			return 0, errMessage
		case "N":
			if 8 != len(parameters[0]) {
				return 0, fmt.Errorf("highestBlock:  invalid response: %q", parameters[0])
			}
			height := binary.BigEndian.Uint64(parameters[0])
			log.Infof("height: %d", height)
			return height, nil
		default:
			return 0, fmt.Errorf("unexpected response: %q", fn)
		}
	}
	return 0, errors.New("Peer :" + peerID.Pretty() + " is not registered")
}

//PushMessageBus  send the CommandBus Message to the peer with given ID
func (n *Node) PushMessageBus(item BusMessage, peerID peerlib.ID) error {
	log := n.Log
	n.RLock()
	if s, ok := n.RegisterStream[peerID.Pretty()]; ok { // stream has registered
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		packedP2PMsg, err := PackP2PMessage(mode.ChainName(), item.Command, item.Parameters)
		if err != nil {
			return err
		}
		sendLen, err := rw.Write(packedP2PMsg)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		log.Debug(fmt.Sprintf("WRITE:\x1b[32mLength:%d\x1b[0m> ", sendLen))
		rw.Flush()
		respPacked := make([]byte, maxBytesRecieve)
		respLen, err := rw.Read(respPacked) //Expected data chain, fn, block-height
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
		n.RUnlock()
	}
	return errors.New("no register stream")
}

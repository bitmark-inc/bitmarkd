package p2p

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

func (n *Node) register(peerInfo *peerlib.AddrInfo) (*network.Stream, error) {
	s, err := n.Host.NewStream(context.Background(), peerInfo.ID, "p2pstream")
	//nodeChain := mode.ChainName()
	nodeChain := "local"
	n.log.Info("---- start to register ---")
	if err != nil {
		n.log.Warn(err.Error())
		return nil, err
	}
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	p2pData, err := PackRegisterData(nodeChain, "R", peerInfo.ID, peerInfo.Addrs, time.Now())
	if err != nil {
		return nil, err
	}
	p2pMsgPacked, err := proto.Marshal(&P2PMessage{Data: p2pData})
	if err != nil {
		return nil, err
	}
	_, err = rw.Write(p2pMsgPacked)
	if err != nil {
		n.log.Error(err.Error())
		return nil, err
	}
	n.log.Info(fmt.Sprintf("WRITE:\x1b[32mLength:%d\x1b[0m> ", len(p2pMsgPacked)))
	rw.Flush()
	// Wait for response
	resp := make([]byte, maxBytesRecieve)
	respLen, err := rw.Read(resp)
	n.log.Info(fmt.Sprintf("%s RECIEVED:\x1b[32m%d\x1b[0m> ", "listener", respLen))
	if err != nil {
		return nil, err
	}
	if respLen < 1 {
		return nil, errors.New("length of byte recieved is less than 1")
	}
	chain, fn, parameters, err := UnPackP2PMessage(resp[:respLen])
	n.log.Info(fmt.Sprintf("RECIEVE:\x1b[32mLength:%d\x1b[0m> ", respLen))
	if err != nil {
		return nil, err
	}
	if chain != nodeChain {
		return nil, errors.New("Different chain")
	}
	switch fn {
	case "E": //Register error
		errMessage, _ := UnpackListenError(parameters)
		n.log.Warn(fmt.Sprintf("\x1b[31mRegister Error:%s\x1b[0m>", errMessage))
		s.Close()
		return nil, err
	case "R":
		randID, randAddrs, randTs, err := UnPackRegisterData(parameters)
		if err != nil {
			s.Close()
			return nil, err
		}
		if !util.IDEqual(randID, peerInfo.ID) {
			// peer return the info, register send. don't add into peer tree
			announce.AddPeer(randID, randAddrs, randTs) // id, listeners, timestam
			n.log.Info("--> \x1b[32mRegister Successful\x1b[0m")
		}
	}
	return &s, nil
}

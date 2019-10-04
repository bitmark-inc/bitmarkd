package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/p2p"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/gogo/protobuf/proto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

func runInfo(info *peerlib.AddrInfo) {
	s, err := globalData.Host.NewStream(context.Background(), info.ID, "p2pstream")
	nodeChain := "local"
	globalData.Log.Info("---- start to query server info---")
	if err != nil {
		globalData.Log.Warn(err.Error())
		return
	}
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	p2pMsgPacked, err := p2p.PackP2PMessage(nodeChain, "I", [][]byte{})
	_, err = rw.Write(p2pMsgPacked)
	if err != nil {
		globalData.Log.Error(err.Error())
		return
	}
	rw.Flush()
	resp := make([]byte, 10000)
	respLen, err := rw.Read(resp)
	globalData.Log.Info(fmt.Sprintf("--> Read:\x1b[32m%d\x1b[0m> ", respLen))
	if err != nil {
		globalData.Log.Infof("Error:%v", err)
		return
	}
	if respLen < 1 {
		globalData.Log.Info(errors.New("length of byte recieved is less than 1").Error())
	}
	_, _, parameters, err := p2p.UnPackP2PMessage(resp[:respLen])
	if err != nil {
		globalData.Log.Infof("Error:%v", err)
		return
	}
	var servInfo serverInfo
	err = json.Unmarshal(parameters[0], &servInfo)
	if err != nil {
		globalData.Log.Infof("Error:%v", err)
		return
	}

	globalData.Log.Infof("--> ServerInfo \x1b[32mver:%s chain:%s noraml=%v height:%d\x1b[0m", servInfo.Version, servInfo.Chain, servInfo.Normal, servInfo.Height)
}

func runReg(info *peerlib.AddrInfo) {
	s, err := globalData.Host.NewStream(context.Background(), info.ID, "p2pstream")
	//nodeChain := mode.ChainName()
	nodeChain := "local"
	globalData.Log.Info("---- start to register ---")
	if err != nil {
		globalData.Log.Warn(err.Error())
		return
	}
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	p2pData, err := p2p.PackRegisterData(nodeChain, "R", globalData.NodeType, info.ID, info.Addrs, time.Now())
	if err != nil {
		return
	}
	p2pMsgPacked, err := proto.Marshal(&p2p.P2PMessage{Data: p2pData})
	if err != nil {
		return
	}
	_, err = rw.Write(p2pMsgPacked)
	if err != nil {
		globalData.Log.Error(err.Error())
		return
	}
	globalData.Log.Info(fmt.Sprintf("-->WRITE:\x1b[32mLength:%d\x1b[0m> ", len(p2pMsgPacked)))
	rw.Flush()
	// Wait for response
	resp := make([]byte, 10000)
	respLen, err := rw.Read(resp)
	if err != nil {
		globalData.Log.Infof("Error:%v", err)
		return
	}
	if respLen < 1 {
		globalData.Log.Info(errors.New("length of byte recieved is less than 1").Error())
	}
	chain, fn, parameters, err := p2p.UnPackP2PMessage(resp[:respLen])
	globalData.Log.Info(fmt.Sprintf("-->RECIEVE:\x1b[32mLength:%d\x1b[0m> ", respLen))
	if err != nil {
		globalData.Log.Infof("Error:%v", err)
		return
	}
	if chain != nodeChain {
		return
	}
	switch fn {
	case "E": //Register error
		errMessage, _ := p2p.UnpackListenError(parameters)
		globalData.Log.Warn(fmt.Sprintf("\x1b[31mRegister Error:%s\x1b[0m>", errMessage))
		s.Close()
		globalData.Log.Infof("Error:%v", err)
		return
	case "R":
		nType, randID, randAddrs, randTs, err := p2p.UnPackRegisterData(parameters)
		if err != nil {
			s.Close()
			globalData.Log.Infof("Error:%v", err)
			return
		}
		globalData.Log.Infof("-->\x1b[32mnodeType:%v, ID:%s , addrs:%s , timestamp:%d\x1b[0m", nType, randID.String(), util.PrintMaAddrs(randAddrs), randTs)
		return
	}
}

package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/bitmark-inc/bitmarkd/p2p"
	"github.com/bitmark-inc/logger"
	"github.com/libp2p/go-libp2p"
	p2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	crypto "github.com/libp2p/go-libp2p-crypto"
	tls "github.com/libp2p/go-libp2p-tls"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	runAddr      string
	runFn        string
	runKey       string
	globalData   p2p.Node
	gLog         *logger.L
	nodeProtocol = ma.ProtocolWithCode(ma.P_P2P).Name
)

func main() {

	flag.StringVar(&runAddr, "addr", "", "Connect to Address")
	flag.StringVar(&runFn, "fn", "", "command")
	flag.StringVar(&runKey, "key", "", "key path")
	flag.Parse()
	if err := logger.Initialise(getLogConfig()); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	globalData = p2p.Node{NodeType: "client", Log: logger.New("p2p")}
	client, err := NewClient(runKey)
	if err != nil {
		//	panic(err)
		return
	}
	globalData.Host = *client

	globalData.RegisterStream = make(map[string]network.Stream)
	gLog = globalData.Log
	maAddr, err := ma.NewMultiaddr(runAddr)
	if err != nil {
		panic(err)
	}
	info, err := peerlib.AddrInfoFromP2pAddr(maAddr)
	if err != nil {
		gLog.Errorf("create info error:%v", err)
		panic(err)
	}
	//gLog.Infof("my ID:%s", info.ID.Pretty())
	fmt.Println("my ID-:", (*client).ID())
	globalData.DirectConnect(*info)

	sendFn(runFn, info)
	for {
		time.Sleep(10 * time.Second)
	}

}
func sendFn(fn string, info *peerlib.AddrInfo) {
	gLog.Infof("type:%s  id:%s peerAddrs:%s", globalData.NodeType, globalData.Host.ID().String())
	for _, a := range globalData.Host.Addrs() {
		gLog.Info(fmt.Sprintf("Host Address: %s/%v/%s\n", a, nodeProtocol, globalData.Host.ID()))
		fmt.Println(fmt.Sprintf("Host Address-: %s/%v/%s\n", a, nodeProtocol, globalData.Host.ID()))
	}
	switch fn {
	case "R":
		runReg(info)
	case "I":
		runInfo(info)
	default:
	}

}

//NewClient creat a client for query
func NewClient(keypath string) (*p2pcore.Host, error) {
	prvKey, err := loadIdentity(keypath)
	if err != nil {
		fmt.Println("LoadIdentity Error:", err)
		prvKey, _, err = crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			fmt.Println("GenerateEd25519Key error:", err)
			//panic(err)
			return nil, err
		}
		options := []libp2p.Option{libp2p.Identity(prvKey), libp2p.Security(tls.ID, tls.New)}

		host, err := libp2p.New(context.Background(),
			options...,
		)
		if err != nil {
			fmt.Println("NewHost error:", err)
			return nil, err
			//panic(err)
		}
		fmt.Println("Create Host successfully")
		return &host, nil
	}
	return nil, err
}

func loadIdentity(filepath string) (crypto.PrivKey, error) {
	keyBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Println("loadIdentity error:", err)
		return nil, err
	}

	prvKey, err := p2p.DecodeHexToPrvKey(keyBytes) //Hex Decoded binaryString
	if err != nil {
		fmt.Println("loadIdentity error:", err)
		return nil, err
	}
	return prvKey, nil
}

func getLogConfig() logger.Configuration {
	curPath := os.Getenv("PWD")
	var logLevel map[string]string
	logLevel = make(map[string]string, 0)
	logLevel["DEFAULT"] = "info"
	var logConfig = logger.Configuration{
		Directory: curPath,
		File:      "raw_test.log",
		Size:      1048576,
		Count:     20,
		Console:   true,
		Levels:    logLevel,
	}
	return logConfig

}

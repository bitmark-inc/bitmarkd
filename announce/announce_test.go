package announce

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/logger"
	proto "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	curPath := os.Getenv("PWD")
	logLevel := make(map[string]string)
	logLevel["DEFAULT"] = "info"
	var logConfig = logger.Configuration{
		Directory: curPath,
		File:      "routing_test.log",
		Size:      1048576,
		Count:     20,
		Console:   true,
		Levels:    logLevel,
	}
	if err := logger.Initialise(logConfig); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	globalData.log = logger.New("nodes")
	os.Exit(m.Run())
}

func TestStorePeers(t *testing.T) {
	f := func(string) ([]string, error) {
		return []string{}, nil
	}
	err := Initialise("random.test.domain", "", DnsOnly, f)

	assert.Nil(t, err, "wrong error")
}

func TestReadPeers(t *testing.T) {
	curPath := os.Getenv("PWD")
	peerFile := path.Join(curPath, "peers")
	var peers PeerList
	readIN, err := ioutil.ReadFile(peerFile)
	assert.NoError(t, err, "TestReadPeers:readFile Error")
	err = proto.Unmarshal(readIN, &peers)
	assert.NoError(t, err, "proto unmarshal error")
	for _, peer := range peers.Peers {
		addrList := util.ByteAddrsToString(peer.Listeners.Address)
		fmt.Printf("ID:%s, listener:%v Timestamp:%d\n", string(peer.PeerID), addrList, peer.Timestamp)
	}
}

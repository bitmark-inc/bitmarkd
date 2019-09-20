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
	var logLevel map[string]string
	logLevel = make(map[string]string, 0)
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
	fmt.Println("TestStorePeers")
	curPath := os.Getenv("PWD")
	peerfile := path.Join(curPath, "peers")
	// domain from bind9
	err := Initialise("nodes.rachael.bitmark", peerfile)
	assert.NoError(t, err, "routing initialized error")
	storePeers(peerfile)
	assert.NoError(t, err, "routing backupPeers error")
}
func TestReadPeers(t *testing.T) {
	curPath := os.Getenv("PWD")
	peerfile := path.Join(curPath, "peers")
	var peers PeerList
	readin, err := ioutil.ReadFile(peerfile)
	assert.NoError(t, err, "TestReadPeers:readFile Error")
	proto.Unmarshal(readin, &peers)
	for _, peer := range peers.Peers {
		addrList := util.ByteAddrsToString(peer.Listeners.Address)
		fmt.Printf("peerID:%s, listener:%v timestamp:%d\n", string(peer.PeerID), addrList, peer.Timestamp)
	}
}

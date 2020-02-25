package p2p

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"

	peerlib "github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

func TestMain(m *testing.M) {
	curPath := os.Getenv("PWD")
	var logConfig = logger.Configuration{
		Directory: curPath,
		File:      "p2p_test.log",
		Size:      1048576,
		Count:     20,
		Console:   true,
		Levels: map[string]string{
			logger.DefaultTag: "trace",
		},
	}
	if err := logger.Initialise(logConfig); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	globalData.Log = logger.New("nodes")
	os.Exit(m.Run())
}
func TestIDMarshalUnmarshal(t *testing.T) {
	conf, err := mockConfiguration("server", 12136)
	assert.NoError(t, err, "generate mock data error")
	prvKey, err := util.DecodePrivKeyFromHex(conf.PrivateKey)
	assert.NoError(t, err, "Decode Hex Key Error")
	id, err := peerlib.IDFromPrivateKey(prvKey)
	assert.NoError(t, err, "IDFromPrivateKey Error")
	mID, err := id.Marshal()
	assert.NoError(t, err, "ID Marshal Error:")
	id2, err := peerlib.IDFromBytes(mID)
	assert.NoError(t, err, "not a valid id bytes")
	assert.Equal(t, id.String(), id2.String(), fmt.Sprintf("Convert ID fail! id:%v", id2.ShortString()))

}

func TestNewP2P(t *testing.T) {
	config, err := mockConfiguration("server", 22136)
	assert.NoError(t, err, "mockdata generate error")
	err = Initialise(config, "v1.0.0", false)
	assert.NoError(t, err, "P2P  initialized error")
	Finalise()
}

func TestListen(t *testing.T) {
	config, err := mockConfiguration("server", 22136)
	assert.NoError(t, err, "mockdata generate error")
	n1 := Node{}
	n1.Log = logger.New("p2p")
	n1.Setup(config, "p2p-v1", false)
	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(config.Port))
	assert.NoError(t, err, fmt.Sprintf("listen error : %v", err))
	conn.Close()
	n1.Host.Close()
}

func TestIsTheSameNode(t *testing.T) {
	config, err := mockConfiguration("server", 22136)
	assert.NoError(t, err, "mockdata generate error")
	n1 := Node{}
	n1.Log = logger.New("p2p")
	n1.Setup(config, "p2p-v1", false)
	n1Info, err := peerlib.AddrInfoFromP2pAddr(n1.Announce[0])
	same := n1.isSameNode(*n1Info)
	assert.Equal(t, true, same, "should be the same node but not")
}

func mockConfiguration(nType string, port int) (*Configuration, error) {
	portString := strconv.Itoa(port)
	hexKey, err := util.MakeEd25519PeerKey()
	if err != nil {
		return nil, err
	}

	return &Configuration{
		NodeType:   nType,
		Port:       port,
		Listen:     []string{"0.0.0.0:" + portString, "[::]:" + portString},
		Announce:   []string{"127.0.0.1:" + portString, "[::1]:" + portString},
		PrivateKey: hexKey,
		Connect:    []StaticConnection{},
	}, nil
}

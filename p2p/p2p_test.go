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

// Note: test will fail very badly if something is using this port
//       lots of "bind: address already in use" will be produced
const unusedPortForTesting = 32136

// this protects against error cascade and resulting panic
// usage: _ = assert.NoError(t, err, "…") || die(t)
func die(t *testing.T) bool {
	t.Fatal("assertNoError must die")
	return true
}

func TestMain(m *testing.M) {
	const theLogFile = "p2p_test.log"
	_ = os.Remove(theLogFile)
	curPath := os.Getenv("PWD")
	var logConfig = logger.Configuration{
		Directory: curPath,
		File:      theLogFile,
		Size:      1048576,
		Count:     20,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "trace",
		},
	}
	if err := logger.Initialise(logConfig); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	globalData.Log = logger.New("nodes")
	rc := m.Run()
	_ = os.Remove(theLogFile)
	os.Exit(rc)
}

func TestIDMarshalUnmarshal(t *testing.T) {
	conf, err := mockConfiguration("server", unusedPortForTesting)
	_ = assert.NoError(t, err, "generate mock data error") || die(t)
	secretKey, err := util.DecodePrivKeyFromHex(conf.SecretKey)
	_ = assert.NoError(t, err, "Decode Hex Key Error") || die(t)
	id, err := peerlib.IDFromPrivateKey(secretKey)
	_ = assert.NoError(t, err, "IDFromSecretKey Error") || die(t)
	mID, err := id.Marshal()
	_ = assert.NoError(t, err, "ID Marshal Error:") || die(t)
	id2, err := peerlib.IDFromBytes(mID)
	_ = assert.NoError(t, err, "not a valid id bytes") || die(t)
	assert.Equal(t, id.String(), id2.String(), fmt.Sprintf("Convert ID fail! id: %v", id2.ShortString()))

}

func TestNewP2P(t *testing.T) {
	config, err := mockConfiguration("server", unusedPortForTesting)
	_ = assert.NoError(t, err, "mockdata generate error") || die(t)
	err = Initialise(config, "v1.0.0", false)
	_ = assert.NoError(t, err, "P2P  initialized error") || die(t)
	Finalise()
}

func TestListen(t *testing.T) {
	config, err := mockConfiguration("server", unusedPortForTesting)
	_ = assert.NoError(t, err, "mockdata generate error") || die(t)
	n1 := Node{}
	n1.Log = logger.New("p2p")
	err = n1.Setup(config, "p2p-v1", false)
	_ = assert.NoError(t, err, fmt.Sprintf("node setup error: %s", err)) || die(t)
	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(config.Port))
	_ = assert.NoError(t, err, fmt.Sprintf("listen error: %s", err)) || die(t)
	conn.Close()
	n1.Host.Close()
}

func TestIsTheSameNode(t *testing.T) {
	config, err := mockConfiguration("server", unusedPortForTesting)
	_ = assert.NoError(t, err, "mockdata generate error") || die(t)
	n1 := Node{}
	n1.Log = logger.New("p2p")
	err = n1.Setup(config, "p2p-v1", false)
	_ = assert.NoError(t, err, fmt.Sprintf("node setup error: %s", err)) || die(t)
	n1Info, err := peerlib.AddrInfoFromP2pAddr(n1.Announce[0])
	_ = assert.NoError(t, err, fmt.Sprintf("convert AddrInfo fail address:%v", n1.Announce[0])) || die(t)
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
		NodeType:  nType,
		Port:      port,
		Listen:    []string{"0.0.0.0:" + portString, "[::]:" + portString},
		Announce:  []string{"127.0.0.1:" + portString, "[::1]:" + portString},
		SecretKey: hexKey,
		Connect:   []StaticConnection{},
	}, nil
}

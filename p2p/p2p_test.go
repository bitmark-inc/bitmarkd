package p2p

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/logger"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	curPath := os.Getenv("PWD")
	var logLevel map[string]string
	logLevel = make(map[string]string, 0)
	logLevel["DEFAULT"] = "info"
	var logConfig = logger.Configuration{
		Directory: curPath,
		File:      "p2p_test.log",
		Size:      1048576,
		Count:     20,
		Console:   true,
		Levels:    logLevel,
	}
	if err := logger.Initialise(logConfig); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	globalData.Log = logger.New("nodes")
	os.Exit(m.Run())
}

func mockConfiguration(nType string, port int) *Configuration {
	return &Configuration{
		NodeType:           nType,
		Port:               port,
		DynamicConnections: true,
		PreferIPv6:         false,
		Listen:             []string{"0.0.0.0:2136", "[::]:2136"},
		Announce:           []string{"118.163.120.180:2136", "[2001:b030:2303:100:699b:a02d:9230:d2cb]:2136"},
		PrivateKey:         "080112406eb84a3845d33c2a389d7fbea425cbf882047a2ab13084562f06875db47b5fdc2e45a298e6cd0472eeb97cd023c723824e157869d81039794864987c05b212a8",
		Connect:            []StaticConnection{},
	}
}

func TestIDMarshalUnmarshal(t *testing.T) {
	conf := mockConfiguration("servant", 12136)
	fmt.Println(conf.PrivateKey)
	prvKey, err := DecodeHexToPrvKey([]byte(conf.PrivateKey))
	assert.NoError(t, err, "Decode Hex Key Error")
	id, err := peer.IDFromPrivateKey(prvKey)
	assert.NoError(t, err, "IDFromPrivateKey Error:")
	fmt.Println("id:", id)
	mID, err := id.Marshal()
	assert.NoError(t, err, "ID Marshal Error:")
	id2, err := peer.IDFromBytes(mID)
	assert.NoError(t, err, "not a valid id bytes")
	fmt.Println("id2:", id2.String(), " shortID:", id2.ShortString())
	assert.Equal(t, id.String(), id2.String(), "Convert ID fail")
}
func TestNewP2P(t *testing.T) {
	err := Initialise(mockConfiguration("servant", 12136), "v1.0.0")
	assert.NoError(t, err, "P2P  initialized error")
	time.Sleep(8 * time.Second)
	defer announce.Finalise()
}

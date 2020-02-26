package p2p

import (
	"crypto/rand"
	"testing"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

type mockRegisterData struct {
	chain   string
	fn      string
	role    nodeType
	id      peerlib.ID
	addrs   []ma.Multiaddr
	curTime time.Time
}

func TestPackUnPackP2PMessage(t *testing.T) {
	type m struct {
		chain  string
		fn     string
		params [][]byte
	}

	d1 := m{chain: "live", fn: "R", params: [][]byte{[]byte("testing"), []byte("message")}}
	d2 := m{chain: "live", fn: "R", params: [][]byte{}} // empty case
	d3 := m{chain: "live", fn: "R"}                     // nil case

	// Test set 1
	packed, err := PackP2PMessage(d1.chain, d1.fn, d1.params)
	assert.NoError(t, err, "PackP2PMessage error")
	unpackChain, unpackFn, unpackParam, err := UnPackP2PMessage(packed)
	assert.NoError(t, err, "UnPackP2PMessage error")
	assert.Equal(t, d1.chain, unpackChain, "UnPackP2PMessage chain error")
	assert.Equal(t, d1.fn, unpackFn, "UnPackP2PMessage fn error")
	assert.Equal(t, d1.params, unpackParam, "UnPackP2PMessage params error")
	// Test set 2
	packed, err = PackP2PMessage(d2.chain, d2.fn, d2.params)
	assert.NoError(t, err, "PackP2PMessage error")
	_, _, unpackParam, err = UnPackP2PMessage(packed)
	assert.NoError(t, err, "UnPackP2PMessage error")
	assert.Equal(t, [][]byte{}, unpackParam, "UnPackP2PMessage params error")
	// Test set 3
	packed, err = PackP2PMessage(d3.chain, d3.fn, d3.params)
	assert.NoError(t, err, "PackP2PMessage error")
	_, _, unpackParam, err = UnPackP2PMessage(packed)
	assert.NoError(t, err, "UnPackP2PMessage error")
	assert.Equal(t, [][]byte{}, unpackParam, "UnPackP2PMessage params error")
}

func TestPackUnPackRegisterDataMessage(t *testing.T) {
	mockdata, err := genRegisterData()
	assert.NotEqual(t, nil, err, "generate register data fail")
	for idx, data := range mockdata {
		packedParam, err := PackRegisterParameter(data.role, data.id, data.addrs, data.curTime)
		if 2 == idx { // case 3 addrs is nil , addrs can empty but not nil
			assert.Error(t, err, "PackRegisterParameter does not check addrs nil case")
			return
		}
		assert.NoError(t, err, "PackRegisterParameter fail")

		packedP2PMsg, err := PackP2PMessage(data.chain, data.fn, packedParam)
		assert.NoError(t, err, "pack p2p header data error")

		unPackChain, unPackFn, unPackParams, err := UnPackP2PMessage(packedP2PMsg)
		assert.NoError(t, err, "unpack p2p header data error")
		assert.NotEqual(t, unPackChain, data.chain)
		assert.NotEqual(t, unPackFn, data.fn)

		unpackType, unpackID, unPackAddrs, unpackTs, err := UnPackRegisterParameter(unPackParams)
		assert.NoError(t, err, "unpack registerer param data error")
		assert.NotEqual(t, unpackType, data.role, "unpack type error")
		assert.NotEqual(t, unpackID, data.id, "unpack id error")
		assert.NotEqual(t, unPackAddrs, data.addrs, "unpack addrs error")
		assert.NotEqual(t, unpackTs, data.curTime.UnixNano(), "unpack ts error")
	}
}

func genRegisterData() ([]mockRegisterData, error) {
	data := []mockRegisterData{}
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, rand.Reader)
	if err != nil {
		return nil, err
	}
	libp2pID, err := peerlib.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, err
	}
	addrStrings := []string{
		"/ip4/127.0.0.1/tcp/1234",
		"/ip6/::1/tcp/1234",
		"ipv6/2001:b030:2314:200:2c0d:42b5:471:e95c/tcp/1234",
	}
	addrs := []ma.Multiaddr{}
	for _, maAddr := range addrStrings {
		addr, err := ma.NewMultiaddr(maAddr)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}
	ts := time.Now()
	data = append(data, mockRegisterData{chain: "test", fn: "R", role: ServerNode, id: libp2pID, addrs: addrs, curTime: ts})
	data = append(data, mockRegisterData{chain: "test", fn: "R", role: ClientNode, id: libp2pID, addrs: []ma.Multiaddr{}, curTime: ts})
	data = append(data, mockRegisterData{chain: "test", fn: "R", role: ClientNode, id: libp2pID, addrs: nil, curTime: ts})
	return data, nil
}

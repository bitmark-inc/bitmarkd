package avl_test

import (
	"fmt"
	"testing"

	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/bitmarkd/avl"
	p2pPeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

type peerIDkey p2pPeer.ID

// Compare - public key comparison for AVL interface
func (p peerIDkey) Compare(q interface{}) int {
	return util.IDCompare(p2pPeer.ID(p), p2pPeer.ID(q.(peerIDkey)))
}

// peerIDKey to String
func (p peerIDkey) String() string {
	return p2pPeer.ID(p).String()
}

// TestCompare
func TestCompare(t *testing.T) {
	fmt.Println("Start Comparing")
	IDKeys := []peerIDkey{
		peerIDkey(p2pPeer.ID("1000")),
		peerIDkey(p2pPeer.ID("8133")),
		peerIDkey(p2pPeer.ID("999")),
	}
	lowKey := peerIDkey(p2pPeer.ID("1000"))
	res := lowKey.Compare(IDKeys[0])
	assert.Equal(t, res, 0, "Not Equal")
	res = lowKey.Compare(IDKeys[1])
	assert.Greater(t, 0, res, "Input is not greater")
	res = lowKey.Compare(IDKeys[2])
	assert.Greater(t, 0, res, "Input is not lesser")
}

func TestGetKey(t *testing.T) {
	IDKeys := []peerIDkey{
		peerIDkey(p2pPeer.ID("1000")),
		peerIDkey(p2pPeer.ID("8133")),
		peerIDkey(p2pPeer.ID("999")),
	}
	tree := avl.New()
	for _, key := range IDKeys {
		tree.Insert(key, "data:"+key.String())
	}
	tree.Print(true)
	for i := 0; i < tree.Count(); i++ {
		node := tree.Get(i)
		IDKey := (node.Key()).(peerIDkey)
		fmt.Println("[", i, "]", IDKey)
	}
}

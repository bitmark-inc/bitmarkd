package avl_test

import (
	"bitmark-network/util"
	"fmt"
	"testing"

	"github.com/bitmark-inc/bitmarkd/avl"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

type peerIDkey peerlib.ID

// Compare - public key comparison for AVL interface
func (p peerIDkey) Compare(q interface{}) int {
	return util.IDCompare(peerlib.ID(p), peerlib.ID(q.(peerIDkey)))
}

// peerIDKey to String
func (p peerIDkey) String() string {
	return peerlib.ID(p).String()
}

// TestCompare
func TestCompare(t *testing.T) {
	fmt.Println("Start Comparing")
	idkeys := []peerIDkey{
		peerIDkey(peerlib.ID("1000")),
		peerIDkey(peerlib.ID("8133")),
		peerIDkey(peerlib.ID("999")),
	}
	lowKey := peerIDkey(peerlib.ID("1000"))
	res := lowKey.Compare(idkeys[0])
	assert.Equal(t, res, 0, "Not Equal")
	res = lowKey.Compare(idkeys[1])
	assert.Greater(t, 0, res, "Input is not greater")
	res = lowKey.Compare(idkeys[2])
	assert.Greater(t, 0, res, "Input is not lesser")
}

func TestGetKey(t *testing.T) {
	idkeys := []peerIDkey{
		peerIDkey(peerlib.ID("1000")),
		peerIDkey(peerlib.ID("8133")),
		peerIDkey(peerlib.ID("999")),
	}
	tree := avl.New()
	for _, key := range idkeys {
		//t.Logf("add item: %q", key)
		tree.Insert(key, "data:"+key.String())
	}
	tree.Print(true)
	for i := 0; i < tree.Count(); i++ {
		node := tree.Get(i)
		idkey := (node.Key()).(peerIDkey)
		fmt.Println("[", i, "]", idkey)
	}
}

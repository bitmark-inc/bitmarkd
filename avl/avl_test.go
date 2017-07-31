// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package avl_test

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/avl"
	"sort"
	"strings"
	"testing"
)

//type stringItem string
type stringItem struct {
	s string
}

func (s stringItem) String() string {
	return s.s
}
func (s stringItem) Compare(x interface{}) int {
	return strings.Compare(s.s, x.(stringItem).s)
}

func TestListOne(t *testing.T) {

	addList := []stringItem{
		{"4201"}, {"1254"}, {"8608"}, {"1639"}, {"8950"},
		{"9564"}, {"1963"}, {"3975"}, {"1710"}, {"2861"},
		{"2928"}, {"7694"}, {"5263"}, {"5632"}, {"8158"},
		{"1247"}, {"3004"}, {"0194"}, {"6644"}, {"3169"},
	}
	doList(t, addList)
	doTraverse(t, addList)
}

func TestListTwo(t *testing.T) {

	addList := []stringItem{
		{"8133"}, {"2136"}, {"9651"}, {"4079"}, {"1042"},
		{"3579"}, {"3630"}, {"1427"}, {"5843"}, {"9549"},
		{"5433"}, {"1274"}, {"9034"}, {"4724"}, {"6179"},
		{"5072"}, {"9272"}, {"4030"}, {"4205"}, {"3363"},
		{"8582"}, {"1720"}, {"0506"}, {"8382"}, {"6774"},
		{"3088"}, {"2329"}, {"9039"}, {"6703"}, {"1027"},
		{"7297"}, {"6063"}, {"4156"}, {"1005"}, {"0982"},
		{"3065"}, {"2553"}, {"0795"}, {"8426"}, {"2377"},
		{"0877"}, {"9085"}, {"5918"}, {"2581"}, {"7797"},
		{"3028"}, {"5880"}, {"3061"}, {"5212"}, {"6539"},
		{"1320"}, {"3581"}, {"3334"}, {"4348"}, {"2934"},
		{"8342"}, {"8814"}, {"8736"}, {"1353"}, {"3082"},
		{"9620"}, {"0056"}, {"5063"}, {"1245"}, {"7066"},
		{"7435"}, {"2999"}, {"7803"}, {"1303"}, {"1697"},
		{"0017"}, {"4314"}, {"9926"}, {"7587"}, {"2531"},
		{"8123"}, {"5693"}, {"7495"}, {"9975"}, {"5465"},
		{"4342"}, {"7958"}, {"7138"}, {"9382"}, {"0672"},
		{"5402"}, {"0204"}, {"2397"}, {"2712"}, {"0938"},
		{"9610"}, {"3611"}, {"2140"}, {"4289"}, {"9271"},
		{"4786"}, {"4145"}, {"1066"}, {"4366"}, {"6716"},
		{"8579"}, {"1012"}, {"5935"}, {"8278"}, {"5761"},
		{"1871"}, {"6257"}, {"2649"}, {"8643"}, {"1239"},
		{"3416"}, {"6146"}, {"7127"}, {"9517"}, {"5788"},
		{"9025"}, {"6880"}, {"9064"}, {"4849"}, {"4503"},
		{"4898"}, {"6815"}, {"8811"}, {"6745"}, {"6907"},
		{"7503"}, {"9869"}, {"5491"}, {"9940"}, {"5955"},
		{"3764"}, {"3254"}, {"8048"}, {"5339"}, {"2406"},
		{"3137"}, {"0251"}, {"0486"}, {"4202"}, {"1844"},
		{"1741"}, {"7154"}, {"4286"}, {"5160"}, {"9472"},
		{"2998"}, {"1935"}, {"4758"}, {"6478"}, {"9572"},
		{"9254"}, {"6848"}, {"3126"}, {"1848"}, {"7692"},
		{"2791"}, {"1504"}, {"3469"}, {"9701"}, {"5077"},
		{"7928"}, {"7978"}, {"5383"}, {"4319"}, {"8197"},
		{"9227"}, {"1166"}, {"4216"}, {"0866"}, {"1791"},
		{"5395"}, {"4310"}, {"4452"}, {"6140"}, {"1494"},
		{"8859"}, {"3394"}, {"5507"}, {"7295"}, {"5408"},
		{"7789"}, {"8237"}, {"6990"}, {"6882"}, {"8243"},
		{"8894"}, {"4352"}, {"6727"}, {"7019"}, {"3126"},
		{"3102"}, {"2948"}, {"8242"}, {"5027"}, {"8892"},
		{"3492"}, {"1323"}, {"1101"}, {"4526"}, {"5177"},
		{"6175"}, {"6664"}, {"2742"}, {"6094"}, {"9877"},
		{"2534"}, {"2105"}, {"6588"}, {"9982"}, {"3696"},
		{"3480"}, {"2244"}, {"7487"}, {"2844"}, {"3199"},
		{"5829"}, {"6952"}, {"6915"}, {"0905"}, {"7615"},
	}

	doList(t, addList)
	doTraverse(t, addList)
}

func doList(t *testing.T, addList []stringItem) {

	for i := 0; i < len(addList)+1; i += 1 {

		t.Logf("delete size: %d", i)
		alreadyDeleted := make(map[stringItem]struct{})

		tree := avl.New()
		for _, key := range addList {
			//t.Logf("add item: %v", key)
			tree.Insert(key, "data:"+key.String())
		}

		if !tree.CheckUp() {
			t.Errorf("add: inconsistant tree")
			depth := tree.Print(true)
			t.Logf("depth: %v", depth)
			t.Fatal("inconsistant tree")
		}

	delete_items:
		for _, key := range addList[:i] {
			//t.Logf("delete item: %v", key)
			if _, ok := alreadyDeleted[key]; ok {
				continue delete_items
			}
			alreadyDeleted[key] = struct{}{}
			dv := tree.Delete(key)
			ev := "data:" + key.String()
			if dv != ev {
				t.Fatalf("delete returned: %v  expected: %v", dv, ev)
			}
		}

		if !tree.CheckUp() {
			t.Errorf("delete: inconsistant tree")
			depth := tree.Print(true)
			t.Logf("depth: %v", depth)
			t.Fatal("inconsistant tree")
		}

	delete_remainder:
		for _, key := range addList[i:] {
			//t.Logf("delete item: %v", key)
			if _, ok := alreadyDeleted[key]; ok {
				continue delete_remainder
			}
			alreadyDeleted[key] = struct{}{}
			dv := tree.Delete(key)
			ev := "data:" + key.String()
			if dv != ev {
				t.Fatalf("delete returned: %v  expected: %v", dv, ev)
			}
		}
		if !tree.IsEmpty() {
			t.Errorf("remainder:remaining nodes")
			depth := tree.Print(true)
			t.Logf("depth: %v", depth)
			t.Fatal("remaining nodes")
		}
	}
}

// traverse the tree forwards and backwards to check iterators
func doTraverse(t *testing.T, addList []stringItem) {

	unique := make(map[string]struct{})
	tree := avl.New()
	for _, key := range addList {
		unique[key.String()] = struct{}{}
		tree.Insert(key, "data:"+key.String())
	}

	p := tree.First()
	if nil == p {
		t.Fatalf("no first item")
	}

	expected := make([]string, 0, len(unique))
	for key := range unique {
		expected = append(expected, key)
	}
	sort.Strings(expected)

	n := 0
	for i := 0; nil != p; i += 1 {
		if 0 != p.Key().Compare(stringItem{expected[i]}) {
			t.Fatalf("next item: actual: %v  expected: %v", p.Key(), expected[i])
		}
		n += 1
		p = p.Next()
	}

	if n != len(expected) {
		t.Fatalf("item count: actual: %v  expected: %v", n, len(addList))
	}

	p = tree.Last()
	if nil == p {
		t.Fatalf("no last item")
	}

	n = 0
	for i := len(expected) - 1; nil != p; i -= 1 {
		if 0 != p.Key().Compare(stringItem{expected[i]}) {
			t.Fatalf("prev item: actual: %v  expected: %v", p.Key(), expected[i])
		}
		n += 1
		p = p.Prev()
	}

	if n != len(expected) {
		t.Fatalf("item count: actual: %v  expected: %v", n, len(addList))
	}
	if n != tree.Count() {
		t.Fatalf("tree count: actual: %v  expected: %v", tree.Count(), len(addList))
	}

	// delete remainder
	for _, key := range expected {
		//t.Logf("delete item: %v", key)
		tree.Delete(stringItem{key})
	}

	if !tree.IsEmpty() {
		t.Errorf("remainder:remaining nodes")
		depth := tree.Print(true)
		t.Logf("depth: %v", depth)
		t.Fatalf("remaining nodes")
	}
	if 0 != tree.Count() {
		t.Fatalf("remaining count not zero: %d", tree.Count())
	}

}

func makeKey() stringItem {

	b := make([]byte, 4)
	_, err := rand.Read(b)
	if nil != err {
		panic("rand failed")
	}
	n := int(binary.BigEndian.Uint32(b))
	return stringItem{fmt.Sprintf("%04d", n%10000)}
}

func TestRandomTree(t *testing.T) {
	tree := avl.New()

	const (
		toDelete = 2000
		total    = 2200
	)
	d := make([]stringItem, toDelete)

	for i := 0; i < total; i += 1 {
		key := makeKey()
		if i < len(d) {
			d[i] = key
		}
		//t.Logf("add item: %v", key)
		tree.Insert(key, "data:"+key.String())
	}

	if !tree.CheckUp() {
		depth := tree.Print(true)
		t.Logf("depth: %v", depth)
		t.Fatalf("inconsistant tree")
	}

	for _, key := range d {
		//t.Logf("delete item: %v", key)
		tree.Delete(key)
		if !tree.CheckUp() {
			depth := tree.Print(true)
			t.Logf("depth: %d", depth)

			t.Fatalf("inconsistant tree")
		}
	}

	// add back the test value
	testKey := stringItem{"500"}
	const testValue = "just testing data: test 500 value"
	tree.Insert(testKey, testValue)

	if !tree.CheckUp() {
		depth := tree.Print(true)
		t.Logf("depth: %d", depth)

		t.Fatalf("inconsistant tree")
	}

	doTraverse(t, d)

	// check that test value is searchable
	tv := tree.Search(testKey)
	if nil == tv {
		t.Fatalf("could not find test key: %v", testKey)
	}
	if testKey != tv.Key() {
		t.Fatalf("test key mismatch: actual: %v  expected: %v", tv.Key(), testKey)
	}
	if testValue != tv.Value() {
		t.Fatalf("test value mismatch: actual: %v  expected: %v", tv.Value(), testValue)
	}

	// check iterators
	n := tv.Next()
	p := tv.Prev()
	if nil == n {
		t.Fatal("could not find next")
	}
	if nil == p {
		t.Fatal("could not find prev")
	}

	t.Logf("test: %v", tv.Value())
	t.Logf("next: %v", n.Value())
	t.Logf("prev: %v", p.Value())

	// delete the test value, and check it return the correct
	// value and is no longer in the tree
	value := tree.Delete(testKey)
	if value != testValue {
		t.Fatalf("delete value mismatch: actual: %v  expected: %v", value, testValue)
	}
	tv = tree.Search(testKey)
	if nil != tv {
		t.Fatalf("test key not deleted and contains: %v", tv.Value())
	}

}

// check that inserted nodes can be overwritten
// and that nodes keep constant address when tree is re-balanced
func TestOverwriteAndNodeStability(t *testing.T) {
	addList := []stringItem{
		{"01"}, {"02"}, {"03"}, {"04"}, {"05"},
		{"06"}, {"07"}, {"08"}, {"09"}, {"10"},
	}

	tree := avl.New()
	for _, key := range addList {
		//t.Logf("add item: %v", key)
		tree.Insert(key, "data:"+key.String())
	}

	if !tree.CheckUp() {
		t.Errorf("add: inconsistant tree")
		depth := tree.Print(true)
		t.Logf("depth: %v", depth)
		t.Fatalf("inconsistant tree")
	}

	// overwrite a key
	oKey := stringItem{"05"}
	const newData = "new content for 05"
	tree.Insert(oKey, newData)

	if !tree.CheckUp() {
		t.Errorf("add: inconsistant tree")
		depth := tree.Print(true)
		t.Logf("depth: %v", depth)
		t.Fatalf("inconsistant tree")
	}

	// check overwrite
	node1 := tree.Search(oKey)
	t.Logf("v:%p → %v", node1, node1)

	if newData != node1.Value() {
		t.Fatalf("node data actual: %v  expected: %v", node1.Value(), newData)
	}

	// delete a node so the oKey node moves
	dKey := stringItem{"06"}
	t.Logf("delete item: %v", dKey)
	tree.Delete(dKey)

	// ensure node did not move
	node2 := tree.Search(oKey)
	t.Logf("v:%p → %v", node2, node2)
	if node1 != node2 {
		t.Fatalf("node moved from: %p → %p", node1, node2)
	}
	if !tree.CheckUp() {
		t.Errorf("delete: inconsistant tree")
		depth := tree.Print(true)
		t.Logf("depth: %v", depth)
		t.Fatalf("inconsistant tree")
	}
}

func TestGetDepthInTree(t *testing.T) {
	addList := []stringItem{
		{"01"}, {"02"}, {"03"}, {"04"}, {"05"},
		{"06"}, {"07"},
	}

	tree := avl.New()
	for _, key := range addList {
		tree.Insert(key, "data:"+key.String())
	}

	if d := tree.First().Next().Depth(); d != 1 {
		t.Fatalf("incorrect node depth: %d", d)
	}

	if d := tree.First().Next().Next().Depth(); d != 2 {
		t.Fatalf("incorrect node depth: %d", d)
	}
}

func TestGetChildrenByDepth(t *testing.T) {
	addList := []stringItem{
		{"01"}, {"02"}, {"03"}, {"04"}, {"05"},
		{"06"}, {"07"},
	}

	tree := avl.New()
	for _, key := range addList {
		tree.Insert(key, "data:"+key.String())
	}

	if len(tree.Root().GetChildrenByDepth(1)) != 2 {
		t.Fatalf("incorrect children numner in depth 1")

	}

	if len(tree.Root().GetChildrenByDepth(2)) != 4 {
		t.Fatalf("incorrect children numner in depth 2")
	}
}

func TestGetOrderInTree(t *testing.T) {
	addList := []stringItem{
		{"01"}, {"02"}, {"03"}, {"04"}, {"05"},
		{"06"}, {"07"},
	}

	tree := avl.New()
	for _, key := range addList {
		tree.Insert(key, "data:"+key.String())
	}

	nodeOrder := tree.Root().GetOrder(stringItem{"03"})
	if nodeOrder != 2 {
		t.Fatalf("incorrect node order: %d", nodeOrder)
	}
	nodeOrder = tree.Root().GetOrder(stringItem{"05"})
	if nodeOrder != 4 {
		t.Fatalf("incorrect node order: %d", nodeOrder)
	}
}

func TestGetNodeByOrderInTree(t *testing.T) {
	addList := []stringItem{
		{"01"}, {"02"}, {"03"}, {"04"}, {"05"},
		{"06"}, {"07"},
	}

	tree := avl.New()
	for _, key := range addList {
		tree.Insert(key, "data:"+key.String())
	}

	node := tree.Root().GetNodeByOrder(4)
	if node.Key().Compare(stringItem{"05"}) != 0 {
		t.Fatalf("incorrect node get: %+v", node)
	}

	node = tree.Root().GetNodeByOrder(3)
	if node.Key().Compare(stringItem{"04"}) != 0 {
		t.Fatalf("incorrect node get: %+v", node)
	}
}

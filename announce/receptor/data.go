// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receptor

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/bitmark-inc/bitmarkd/avl"
	"github.com/gogo/protobuf/proto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type Data struct {
	ID        peerlib.ID
	Listeners []ma.Multiaddr
	Timestamp time.Time // last seen time
}

// string - conversion from fmt package
func (d Data) String() []string {
	allAddress := make([]string, 0)
	for _, listener := range d.Listeners {
		allAddress = append(allAddress, listener.String())
	}
	return allAddress
}

func pb(r *Data) *DataPB {
	if r == nil {
		return nil
	}
	var pbAddrs [][]byte
	for _, listener := range r.Listeners {
		pbAddrs = append(pbAddrs, listener.Bytes())
	}
	binaryID, _ := r.ID.Marshal()
	return &DataPB{
		ID:        binaryID,
		Listeners: &Addrs{Address: pbAddrs},
		Timestamp: uint64(r.Timestamp.Unix()),
	}
}

// Backup - backup all receptor data into file
func Backup(backupFile string, tree *avl.Tree) error {
	if tree.Count() <= 2 {
		return nil
	}

	list := List{
		Receptors: make([]*DataPB, 0),
	}

	node := tree.First()
	if node != nil {
		for ; node != nil; node = node.Next() {
			data, ok := node.Value().(*Data)
			if ok {
				r := pb(data)
				list.Receptors = append(list.Receptors, r)
			}
		}
	}

	out, err := proto.Marshal(&list)
	if nil != err {
		return fmt.Errorf("failed to marshal receptor")
	}

	if err := ioutil.WriteFile(backupFile, out, 0600); err != nil {
		return fmt.Errorf("failed writing receptors to file: %s", err)
	}
	return nil
}

// Restore - restore receptors from file
func Restore(backupFile string) (List, error) {
	var list List
	data, err := ioutil.ReadFile(backupFile)
	if err != nil {
		if os.IsNotExist(err) {
			return List{}, nil
		}
		return List{}, err
	}
	err = proto.Unmarshal(data, &list)
	if nil != err {
		return List{}, err
	}
	return list, nil
}

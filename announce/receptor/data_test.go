// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receptor_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/announce/fixtures"
	"github.com/bitmark-inc/bitmarkd/announce/id"
	"github.com/bitmark-inc/bitmarkd/announce/mocks"
	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/avl"
)

const (
	backupFile = "peers"
)

func removeBackupFile() {
	if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
		_ = os.Remove(backupFile)
	}
}

func TestBackup(t *testing.T) {
	removeBackupFile()
	defer removeBackupFile()

	tree := avl.New()
	now := time.Now()

	p1 := &receptor.Entity{
		PublicKey: fixtures.PublicKey1,
		Listeners: fixtures.Listener1,
		Timestamp: now,
	}

	p2 := &receptor.Entity{
		PublicKey: fixtures.PublicKey2,
		Listeners: fixtures.Listener2,
		Timestamp: now,
	}

	p3 := &receptor.Entity{
		PublicKey: fixtures.PublicKey3,
		Listeners: fixtures.Listener1,
		Timestamp: now,
	}

	tree.Insert(id.ID(fixtures.PublicKey1), p1)
	tree.Insert(id.ID(fixtures.PublicKey2), p2)
	tree.Insert(id.ID(fixtures.PublicKey3), p3)

	err := receptor.Backup(backupFile, tree)
	assert.Nil(t, err, "wrong store")

	f, err := os.OpenFile(backupFile, os.O_RDONLY, 0o600)
	assert.Nil(t, err, "peer file read error")
	defer f.Close()

	var list []receptor.StoreEntity
	d := json.NewDecoder(f)
	err = d.Decode(&list)

	assert.Nil(t, err, "wrong unmarshal")
	assert.Equal(t, 3, len(list), "wrong entity count")

	for _, l := range list {
		if id.ID(l.PublicKey).Compare(id.ID(p1.PublicKey)) != 0 && id.ID(l.PublicKey).Compare(id.ID(p2.PublicKey)) != 0 && id.ID(l.PublicKey).Compare(id.ID(p3.PublicKey)) != 0 {
			t.Error("wrong public key")
			t.FailNow()
		}
	}
}

func TestBackupWhenCountLessOrEqualThanTwo(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	tree := avl.New()
	now := time.Now()
	p1 := &receptor.Entity{
		PublicKey: fixtures.PublicKey1,
		Listeners: fixtures.Listener1,
		Timestamp: now,
	}

	p2 := &receptor.Entity{
		PublicKey: fixtures.PublicKey2,
		Listeners: fixtures.Listener2,
		Timestamp: now,
	}

	tree.Insert(id.ID(p1.PublicKey), p1)
	tree.Insert(id.ID(p2.PublicKey), p2)

	err := receptor.Backup(backupFile, tree)
	assert.Nil(t, err, "wrong store")
	_, err = os.Stat(backupFile)
	assert.NotNil(t, err, "peer file should not be stored")
}

func TestRestore(t *testing.T) {
	removeBackupFile()
	defer removeBackupFile()

	tree := avl.New()
	now := time.Now()

	p1 := &receptor.Entity{
		PublicKey: fixtures.PublicKey1,
		Listeners: fixtures.Listener1,
		Timestamp: now,
	}

	p2 := &receptor.Entity{
		PublicKey: fixtures.PublicKey2,
		Listeners: fixtures.Listener2,
		Timestamp: now,
	}

	p3 := &receptor.Entity{
		PublicKey: fixtures.PublicKey3,
		Listeners: fixtures.Listener1,
		Timestamp: now,
	}

	tree.Insert(id.ID(p1.PublicKey), p1)
	tree.Insert(id.ID(p2.PublicKey), p2)
	tree.Insert(id.ID(p3.PublicKey), p3)

	_ = receptor.Backup(backupFile, tree)

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	r := mocks.NewMockReceptor(ctl)
	r.EXPECT().Add(gomock.Any(), gomock.Any(), gomock.Any()).Return(true).Times(3)

	err := receptor.Restore(backupFile, r)
	assert.Nil(t, err, "wrong restore")
}

//
//func TestRestoreWhenFileNotExist(t *testing.T) {
//	_, err := receptor.Restore("not_exist_file")
//	assert.Nil(t, err, "wrong file not exist error")
//}

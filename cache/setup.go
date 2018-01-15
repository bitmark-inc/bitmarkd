// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package cache

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/background"
)

type item struct {
	object    interface{}
	expiresAt time.Time
}

type poolData struct {
	sync.RWMutex
	items        map[string]item
	expiresAfter time.Duration
}

type pools struct {
	UnverifiedTxIndex   *poolData `exp:"72h"`
	UnverifiedTxEntries *poolData `exp:"72h"`
	ProofFilters        *poolData `exp:"72h"`
	VerifiedTx          *poolData
	PendingTransfer     *poolData `exp:"72h"`
	OrphanPayment       *poolData `exp:"72h"`
	TestA               *poolData `exp:"3s"`
	TestB               *poolData
}

type globalDataType struct {
	background *background.T
}

// Pool is the interface to perform CRUD operations on objects stored in memory
var Pool pools
var globalData globalDataType

// Initialise must be called before any operations to MemPool
func Initialise() error {
	poolType := reflect.TypeOf(Pool)
	poolValue := reflect.ValueOf(&Pool).Elem()

	for i := 0; i < poolType.NumField(); i++ {
		var exp time.Duration

		fieldInfo := poolType.Field(i)
		expTag := fieldInfo.Tag.Get("exp")
		if len(expTag) > 0 {
			d, err := time.ParseDuration(expTag)
			if err != nil {
				return fmt.Errorf("invalid time duration: %s", expTag)
			}
			exp = d
		}

		p := &poolData{items: make(map[string]item), expiresAfter: exp}
		newPool := reflect.ValueOf(p)
		poolValue.Field(i).Set(newPool)
	}

	processes := background.Processes{
		&cleaner{},
	}
	globalData.background = background.Start(processes, nil)

	return nil
}

// Finalise stops the expiration check process
func Finalise() {
	globalData.background.Stop()
}

func (p *poolData) Put(key string, value interface{}) {
	p.Lock()
	defer p.Unlock()

	val := item{object: value}
	if p.expiresAfter > 0 {
		val.expiresAt = time.Now().Add(p.expiresAfter)
	}
	p.items[key] = val
}

func (p *poolData) Get(key string) (interface{}, bool) {
	p.RLock()
	defer p.RUnlock()

	item, ok := p.items[key]
	if !ok {
		return nil, false
	}
	return item.object, true
}

func (p *poolData) Delete(key string) {
	p.Lock()
	defer p.Unlock()

	delete(p.items, key)
}

func (p *poolData) Items() map[string]interface{} {
	p.RLock()
	defer p.RUnlock()

	m := make(map[string]interface{}, len(p.items))
	for k, v := range p.items {
		m[k] = v.object
	}
	return m
}

func (p *poolData) Size() int {
	p.RLock()
	defer p.RUnlock()

	return len(p.items)
}

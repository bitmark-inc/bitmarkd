// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/storage"
	"os"
	"reflect"
	"strconv"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("usage: dumpdb database tag count\n")

		// this will be a struct type
		poolType := reflect.TypeOf(storage.Pool)

		// print all avalable tags
		fmt.Printf(" tags:\n")
		for i := 0; i < poolType.NumField(); i += 1 {
			fieldInfo := poolType.Field(i)
			prefixTag := fieldInfo.Tag.Get("prefix")
			fmt.Printf("       %s → %s\n", prefixTag, fieldInfo.Name)
		}
		return
	}

	filename := os.Args[1]

	tag := os.Args[2]

	count, err := strconv.Atoi(os.Args[3])
	if nil != err {
		fmt.Printf("Error on Fetch: %v\n", err)
		return
	}

	storage.Initialise(filename)
	defer storage.Finalise()

	// this will be a struct type
	poolType := reflect.TypeOf(storage.Pool)

	// read-only access
	poolValue := reflect.ValueOf(storage.Pool)

	// the handle
	p := (*storage.PoolHandle)(nil)
	// write access to p as a Value
	pvalue := reflect.ValueOf(&p).Elem()

	// scan each field to locate tag
	for i := 0; i < poolType.NumField(); i += 1 {
		fieldInfo := poolType.Field(i)
		prefixTag := fieldInfo.Tag.Get("prefix")
		if tag == prefixTag {
			pvalue.Set(poolValue.Field(i))
		}

	}
	if nil == p {
		fmt.Printf("no pool corresponding to: %q\n", tag)
		return
	}

	// dump the items as hex
	cursor := p.NewFetchCursor()
	data, err := cursor.Fetch(count)
	if nil != err {
		fmt.Printf("Error on Fetch: %v\n", err)
		return
	}
	for i, e := range data {
		fmt.Printf("%d: Key: %x\n", i, e.Key)
		fmt.Printf("%d: Val: %x\n", i, e.Value)
	}
}

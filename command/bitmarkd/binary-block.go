// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/storage"
)

// save blocks above genesis block to a file
// record format:
//   big endian record length (n)   8 bytes
//   block data                     n bytes
func saveBinaryBlocks(filename string) error {

	fh, err := os.Create(filename)
	if nil != err {
		return err
	}
	defer fh.Close()

	started := false

	if blockheader.Height() <= genesis.BlockNumber {
		return fmt.Errorf("nothing to save")
	}

loop:
	for n := genesis.BlockNumber + 1; true; n += 1 {
		if n%100 == 0 {
			fmt.Printf("%d", n)
		} else {
			fmt.Printf(".")
		}
		buffer, err := getBinaryBlock(n)
		if nil != err {
			if started && fault.BlockNotFound == err {
				break loop
			}
			return err
		}
		started = true
		l := make([]byte, 8)
		binary.BigEndian.PutUint64(l, uint64(len(buffer)))
		err = writeRecord(fh, l)
		if nil != err {
			return err
		}
		err = writeRecord(fh, buffer)
		if nil != err {
			return err
		}
	}
	return nil
}

func writeRecord(fh *os.File, buffer []byte) error {
	l := len(buffer)
	k, err := fh.Write(buffer)
	if nil != err {
		return err
	}
	if l != k {
		return fmt.Errorf("only wrote: %d of %d", k, l)
	}
	return nil
}

// get a binary record for a block
func getBinaryBlock(number uint64) ([]byte, error) {

	// fetch block and compute digest
	n := make([]byte, 8)
	binary.BigEndian.PutUint64(n, number)

	packed := storage.Pool.Blocks.Get(n)
	if nil == packed {
		return nil, fault.BlockNotFound
	}
	return packed, nil
}

// restore binary blocks from a file
// record format: (as save above)
func restoreBinaryBlocks(filename string) error {
	fh, err := os.Open(filename)
	if nil != err {
		return err
	}
	defer fh.Close()

	if blockheader.Height() > genesis.BlockNumber {
		return fmt.Errorf("not overwriting existing data")
	}

loop:
	for {
		n := blockheader.Height()
		if n%100 == 0 {
			fmt.Printf("%d", n)
		} else {
			fmt.Printf(".")
		}

		l := make([]byte, 8)
		err := readRecord(fh, l)
		if err == io.EOF {
			break loop
		} else if nil != err {
			return err
		}
		size := binary.BigEndian.Uint64(l)

		buffer := make([]byte, size)
		err = readRecord(fh, buffer)
		if nil != err {
			return err
		}
		block.StoreIncoming(buffer, nil, block.NoRescanVerified)
	}
	return nil
}

func readRecord(fh *os.File, buffer []byte) error {
	l := len(buffer)
	k, err := fh.Read(buffer)
	if nil != err {
		return err
	}
	if k != l {
		return fmt.Errorf("only read: %d of %d", k, l)
	}
	return nil
}

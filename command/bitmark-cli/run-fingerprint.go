// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/urfave/cli"
	"golang.org/x/crypto/sha3"
)

// version byte prefix for fingerprint file
const (
	fingerprintVersion byte = 0x01
)

func runFingerprint(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	fileName, err := checkFileName(c.String("file"))
	if err != nil {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "checksumming file: %s\n", fileName)
	}

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	fingerprint := sha3.Sum512(data)
	strFP := fmt.Sprintf("%02x%x", fingerprintVersion, fingerprint)

	if m.verbose {
		fmt.Fprintf(m.e, "fingerprint: %s\n", strFP)
	} else {
		out := struct {
			FileName    string `json:"file_name"`
			Fingerprint string `json:"fingerprint"`
		}{
			FileName:    fileName,
			Fingerprint: strFP,
		}
		printJson(m.w, out)
	}
	return nil
}

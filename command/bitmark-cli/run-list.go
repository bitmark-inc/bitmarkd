// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"sort"

	"github.com/urfave/cli"
)

func runList(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	identities := m.config.Identities

	names := make([]string, len(identities))
	i := 0
	for name := range identities {
		names[i] = name
		i += 1
	}
	sort.Strings(names)

	for _, name := range names {
		flag := "--"
		if len(identities[name].Salt) > 0 {
			flag = "SK"
		}
		fmt.Printf("%s %-20s  %s  %q\n", flag, name, identities[name].Account, identities[name].Description)
	}

	// printJson(m.w, infoConfig)

	return nil
}

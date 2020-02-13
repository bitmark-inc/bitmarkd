// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/urfave/cli"
)

func runList(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	if c.Bool("connections") {
		return listConnections(m.w, m.config.Connections, c.Bool("json"))
	}

	// normal list identities
	identities := m.config.Identities

	names := make([]string, len(identities))
	i := 0
	for name := range identities {
		names[i] = name
		i += 1
	}
	sort.Strings(names)

	if c.Bool("json") {
		type item struct {
			HasSecret   bool   `json:"hasSecretKey"`
			Name        string `json:"name"`
			Account     string `json:"account"`
			Description string `json:"description"`
		}
		jsonData := make([]item, len(names))

		for i, name := range names {
			jsonData[i].HasSecret = len(identities[name].Salt) > 0
			jsonData[i].Name = name
			jsonData[i].Account = identities[name].Account
			jsonData[i].Description = identities[name].Description
		}

		printJson(m.w, jsonData)

	} else {
		for _, name := range names {
			flag := "--"
			if len(identities[name].Salt) > 0 {
				flag = "SK"
			}
			fmt.Fprintf(m.w, "%s %-20s  %s  %q\n", flag, name, identities[name].Account, identities[name].Description)
		}
	}

	return nil
}

func listConnections(handle io.Writer, connections []string, printJSON bool) error {
	if printJSON {
		printJson(handle, connections)
	} else {
		for i, conn := range connections {
			fmt.Fprintf(handle, "%4d %s\n", i, conn)
		}
	}

	return nil
}

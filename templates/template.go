// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package templates

const (
	/**** Configuration template ****/
	ConfigurationTemplate = `
# bitmark-cli.conf -*- mode: libucl -*-

default_identity = "{{.Default_identity}}"
network = "{{.Network}}"
connect = "{{.Connect}}"

# identities
identities = {{.Identities}}

`

	/**** Identity template ****/
	IdentityTemplate = `
  {
    name = "{{.Name}}"
    description = "{{.Description}}"
    public_key = "{{.Public_key}}"
    private_key = "{{.Private_key}}"
    private_key_config = {
      iter = {{.Private_key_config.Iter}}
      salt = "{{.Private_key_config.Salt}}"
    }
  }
`
)

// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package getoptions

import (
	"os"
	"path/filepath"
	"strings"
)

// aliases
type AliasMap map[string]string

// returned options
type OptionsMap map[string][]string

// from OS command-line
func GetOS(aliases AliasMap) (program string, options OptionsMap, arguments []string) {
	options, arguments = Get(os.Args[1:], aliases)
	program = filepath.Base(os.Args[0])
	return
}

// get options from array
func Get(inputs []string, aliases AliasMap) (options OptionsMap, arguments []string) {

	options = make(OptionsMap)
	arguments = make([]string, 0, 10)

	n := 0
loop:
	for i, item := range inputs {

		if 0 == len(item) {
			arguments = append(arguments, "") // empty argument
			continue loop
		}

		// check for end of options
		if "--" == item {
			n = i + 1
			break loop
		}

		// check for option
		if '-' == item[0] {
			for len(item) > 0 && '-' == item[0] {
				item = item[1:]
			}
			if 0 == len(item) {
				continue loop // ignore null option
			}
			name := item
			value := ""
			s := strings.SplitN(item, "=", 2)
			if 2 == len(s) {
				name = s[0]
				value = s[1]
			}

			if newName, ok := aliases[name]; ok {
				name = newName
			}
			options[name] = append(options[name], value)
		} else {
			arguments = append(arguments, item)
		}

	}
	if 0 != n {
		arguments = append(arguments, inputs[n:]...)
	}
	return
}

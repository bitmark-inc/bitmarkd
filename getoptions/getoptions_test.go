// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package getoptions_test

import (
	"github.com/bitmark-inc/bitmarkd/getoptions"
	"reflect"
	"testing"
)

type testItem struct {
	in []string
	op getoptions.OptionsMap
	ar []string
}

func TestGetOptions(t *testing.T) {

	aliases := getoptions.AliasMap{"v":"verbose"}

	tests := []testItem{
		{
			in: []string{"-v", "-x", "-v", "-hello=yes", "--test", "argon", "999", "-verbose"},
			op: getoptions.OptionsMap{"test":[]string{""}, "verbose":[]string{"", "", ""}, "x":[]string{""}, "hello":[]string{"yes"}},
			ar: []string{"argon", "999"},
		},
		{
			in: []string{"-v", "-x", "-hello=yes", "--", "multi-word", "999", "-verbose"},
			op: getoptions.OptionsMap{"verbose":[]string{""}, "x":[]string{""}, "hello":[]string{"yes"}},
			ar: []string{"multi-word", "999", "-verbose"},
		},
		{
			in: []string{"-say=hello", "--say=there", "--say=world", "--", "hello", "earth"},
			op: getoptions.OptionsMap{"say":[]string{"hello", "there", "world"}},
			ar: []string{"hello", "earth"},
		},
	}

	for i, s := range tests {
		options, arguments := getoptions.Get(s.in, aliases)
		if !reflect.DeepEqual(options, s.op) {
			t.Errorf("%d: options: %#v  expected: %#v", i, options, s.op)
		}
		if !reflect.DeepEqual(arguments, s.ar) {
			t.Errorf("%d: arguments: %#v expected: %#v", i, arguments, s.ar)
		}
	}

}

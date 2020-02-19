// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/fault"
)

func TestValidTag(t *testing.T) {

	type testItem struct {
		id  int
		txt string
		err error
	}

	testData := []testItem{
		{
			id:  0,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},
		{
			id:  1,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},
		{
			id:  2,
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},

		// corrupt record
		{
			id:  3,
			txt: "bitmark=v3 a=",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  4,
			txt: "bitmark=v3 a= i=",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  5,
			txt: "bitmark=v3 a",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  6,
			txt: "bitmark=v3 a p",
			err: fault.InvalidDnsTxtRecord,
		},

		// check for missing items
		{
			id:  7,
			txt: "bitmark=v3 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  8,
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  9,
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  10,
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  11,
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136",
			err: fault.InvalidDnsTxtRecord,
		},

		// check for incorrect items
		{
			id:  12,
			txt: "bitmark=v3 a=300.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidIpAddress,
		},
		{
			id:  13,
			txt: "bitmark=v3 a=118.163.120.178;2001:x030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidIpAddress,
		},
		{
			id:  14,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=335669 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 s=32135 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			id:  15,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=0 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			id:  16,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=-12 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			id:  17,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=335x669 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			id:  18,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A761934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidFingerprint,
		},
		{
			id:  19,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=461934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED04 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidFingerprint,
		},
		{
			id:  20,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CZFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED04 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidFingerprint,
		},
		{
			id:  21,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=321369 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			id:  22,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPAQQQ",
			err: fault.InvalidIdentityName,
		},
		{
			id:  23,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=abc123",
			err: fault.InvalidIdentityName,
		},

		// old V2 tags still ok
		{
			id:  24,
			txt: "bitmark=v2 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 s=32135 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},

		// invalid tags
		{
			id:  25,
			txt: "bitmark=v0 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 s=32135 c=32136 i=202c1pec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  26,
			txt: "hello world",
			err: fault.InvalidDnsTxtRecord,
		},
	}

	for i, item := range testData {
		_, err := parseTag(item.txt)

		if item.err != err {
			t.Fatalf("parseTag[%d]: %q  error: %s  expected: %v  id:%d", i, item.txt, err, item.err, item.id)
		}
	}
}

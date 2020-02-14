// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domain_test

import (
	"testing"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/announce/domain"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/stretchr/testify/assert"
)

func TestValidTag(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	type testItem struct {
		txt string
		err error
	}

	testData := []testItem{
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},

		// corrupt record
		{
			txt: "bitmark=v3 a=",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			txt: "bitmark=v3 a= i=",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			txt: "bitmark=v3 a",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			txt: "bitmark=v3 a p",
			err: fault.InvalidDnsTxtRecord,
		},

		// check for missing items
		{
			txt: "bitmark=v3 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136",
			err: fault.InvalidDnsTxtRecord,
		},

		// check for incorrect items
		{
			txt: "bitmark=v3 a=300.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidIpAddress,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:x030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidIpAddress,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=335669 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 s=32135 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=0 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=-12 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=335x669 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A761934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidFingerprint,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=461934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED04 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidFingerprint,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CZFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED04 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidFingerprint,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=321369 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPAQQQ",
			err: fault.InvalidIdentityName,
		},
		{
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=abc123",
			err: fault.InvalidIdentityName,
		},

		// old V2 tags still ok
		{
			txt: "bitmark=v2 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 s=32135 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},

		// invalid tags
		{
			txt: "bitmark=v0 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 s=32135 c=32136 i=202c1pec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			txt: "hello world",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			txt: "",
			err: fault.InvalidNodeDomain,
		},
	}

	for _, item := range testData {
		l := domain.NewLookuper(item.txt, logger.New(logCategory))
		f := func(s string) ([]string, error) {
			return []string{item.txt}, nil
		}
		_, err := l.Lookup(f)
		assert.Equal(t, item.err, err, "wrong error")
	}
}

// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domain_test

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/announce/domain"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

func TestValidTag(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	type testItem struct {
		id  int
		txt string
		err error
	}

	testData := []testItem{
		{
			id:  1,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},
		{
			id:  2,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},
		{
			id:  3,
			txt: "bitmark-p2p=v1 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: nil,
		},

		// corrupt record
		{
			id:  4,
			txt: "bitmark-p2p=v1 a=",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  5,
			txt: "bitmark-p2p=v1 a= i=",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  6,
			txt: "bitmark-p2p=v1 a",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  7,
			txt: "bitmark-p2p=v1 a p",
			err: fault.InvalidDnsTxtRecord,
		},

		// check for missing items
		{
			id:  8,
			txt: "bitmark-p2p=v1 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  9,
			txt: "bitmark-p2p=v1 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  10,
			txt: "bitmark-p2p=v1 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  11,
			txt: "bitmark-p2p=v1 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  12,
			txt: "bitmark-p2p=v1 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136",
			err: fault.InvalidDnsTxtRecord,
		},

		// check for incorrect items
		{
			id:  13,
			txt: "bitmark-p2p=v1 a=300.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidIpAddress,
		},
		{
			id:  14,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:x030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidIpAddress,
		},
		{
			id:  15,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=335669 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 s=32135 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			id:  16,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=0 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			id:  17,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=-12 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			id:  18,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=335x669 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			id:  19,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A761934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidFingerprint,
		},
		{
			id:  20,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=461934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED04 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidFingerprint,
		},
		{
			id:  21,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CZFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED04 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidFingerprint,
		},
		{
			id:  22,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=321369 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPA",
			err: fault.InvalidPortNumber,
		},
		{
			id:  23,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=12D3KooWFuZgAPcKGyaG5HqVKjUWptf4m8BYXk3XFzo8vSzi7vPAQQQ",
			err: fault.InvalidIdentityName,
		},
		{
			id:  24,
			txt: "bitmark-p2p=v1 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 i=abc123",
			err: fault.InvalidIdentityName,
		},

		// ignored items
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
		{
			id:  27,
			txt: "",
			err: fault.InvalidDnsTxtRecord,
		},
	}

	for _, item := range testData {
		_, err := domain.ParseTxt(item.txt)

		if nil == item.err && nil != err {
			t.Errorf("id[%d] error: \"%s\"  expected success", item.id, err)
		} else if item.err != err {
			t.Errorf("id[%d] error: \"%s\"  expected: \"%s\"", item.id, err, item.err)
		}

		l := domain.NewLookuper(item.txt, logger.New(logCategory))
		f := func(s string) ([]string, error) {
			return []string{item.txt}, nil
		}

		r, err := l.Lookup(f)
		if nil == item.err && 1 != len(r) {
			t.Errorf("id[%d] expected 1 record but got: %d", item.id, len(r))
		} else if nil != item.err && 0 != len(r) {
			t.Errorf("id[%d] expected zero records bu got: %d", item.id, len(r))
		}

	}
}

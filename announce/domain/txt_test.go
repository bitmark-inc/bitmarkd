// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package domain_test

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/announce/domain"
	"github.com/bitmark-inc/bitmarkd/announce/fixtures"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

func TestValidTag(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	type testItem struct {
		id  int
		txt string
		err error
	}

	testData := []testItem{
		{
			id:  1,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: nil,
		},
		{
			id:  2,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: nil,
		},
		{
			id:  3,
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: nil,
		},

		// corrupt record
		{
			id:  4,
			txt: "bitmark=v3 a=",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  5,
			txt: "bitmark=v3 a= p=",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  6,
			txt: "bitmark=v3 a",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  7,
			txt: "bitmark=v3 a p",
			err: fault.InvalidDnsTxtRecord,
		},

		// check for missing items
		{
			id:  8,
			txt: "bitmark=v3 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  9,
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  10,
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  11,
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  12,
			txt: "bitmark=v3 a=118.163.120.178;[2001:b030:2314:0200:4649:583d:0001:0120] r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136",
			err: fault.InvalidDnsTxtRecord,
		},

		// check for incorrect items
		{
			id:  13,
			txt: "bitmark=v3 a=300.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidIpAddress,
		},
		{
			id:  14,
			txt: "bitmark=v3 a=118.163.120.178;2001:x030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidIpAddress,
		},
		{
			id:  15,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=335669 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 s=32135 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidPortNumber,
		},
		{
			id:  16,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=0 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidPortNumber,
		},
		{
			id:  17,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=-12 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidPortNumber,
		},
		{
			id:  18,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=335x669 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidPortNumber,
		},
		{
			id:  19,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A761934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidFingerprint,
		},
		{
			id:  20,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=461934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED04 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidFingerprint,
		},
		{
			id:  21,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CZFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED04 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidFingerprint,
		},
		{
			id:  22,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=321369 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidPortNumber,
		},
		{
			id:  23,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=1202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidPublicKey,
		},
		{
			id:  24,
			txt: "bitmark=v3 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 c=32136 p=202c1pec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidPublicKey,
		},

		// old V2 tags still ok
		{
			id:  25,
			txt: "bitmark=v2 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 s=32135 c=32136 p=202c14ec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: nil,
		},

		// invalid tags
		{
			id:  26,
			txt: "bitmark=v0 a=118.163.120.178;2001:b030:2314:0200:4649:583d:0001:0120 r=33566 f=48137A7A76934CAFE7635C9AC05339C20F4C00A724D7FA1DC0DC3875476ED004 s=32135 c=32136 p=202c1pec485c21d0d18e9dfd096bd760a558d5ee1139f8e4b2e15863433e7d51",
			err: fault.InvalidDnsTxtRecord,
		},
		{
			id:  27,
			txt: "hello world",
			err: fault.InvalidDnsTxtRecord,
		},
	}

	for _, item := range testData {
		_, err := domain.Parse(item.txt)

		if item.err == nil && err != nil {
			t.Errorf("id[%d] error: \"%s\"  expected success", item.id, err)
		} else if item.err != err {
			t.Errorf("id[%d] error: \"%s\"  expected: \"%s\"", item.id, err, item.err)
		}

		f := func(s string) ([]string, error) {
			return []string{item.txt}, nil
		}
		l := domain.NewLookuper(logger.New(fixtures.LogCategory), f)

		r, err := l.Lookup(item.txt)

		if err == item.err && len(r) != 1 {
			t.Errorf("id[%d] expected 1 record but got: %d", item.id, len(r))
		} else if err != item.err && len(r) != 0 {
			t.Errorf("id[%d] expected zero records bu got: %d", item.id, len(r))
		}
	}
}

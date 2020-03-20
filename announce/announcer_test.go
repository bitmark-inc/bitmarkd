// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/announce"

	"github.com/bitmark-inc/bitmarkd/announce/fixtures"

	"github.com/bitmark-inc/bitmarkd/announce/mocks"
	"github.com/golang/mock/gomock"
)

func TestSendRegistration(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	f := func(_ string) ([]string, error) { return []string{}, nil }

	_ = announce.Initialise("domain.not.exist", "cache", f)
	defer announce.Finalise()

	// make sure background jobs already finish first round, so
	// no logger will be called
	time.Sleep(20 * time.Millisecond)

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	e := fmt.Errorf("wrong")
	c := mocks.NewMockClient(ctl)
	c.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(e).Times(1)

	err := announce.SendRegistration(c, "")
	assert.Equal(t, e, err, "wrong SendRegistration")
}

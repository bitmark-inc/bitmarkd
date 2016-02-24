// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// type to hold Certificate
type Certificate struct {
	log *logger.L
}

// ------------------------------------------------------------

type GetCertificateArguments struct {
	Fingerprint util.FingerprintBytes
}

type GetCertificateReply struct {
	Certificate []byte
}

func (t *Certificate) Get(arguments *GetCertificateArguments, reply *GetCertificateReply) error {

	certificate, found := announce.GetCertificate(&arguments.Fingerprint)
	if !found {
		return fault.ErrCertificateNotFound
	}
	reply.Certificate = certificate
	return nil
}

// ------------------------------------------------------------

type PutCertificateArguments struct {
	Certificate []byte
}

type PutCertificateReply struct {
}

func (t *Certificate) Put(arguments *PutCertificateArguments, reply *PutCertificateReply) error {

	fingerprint := util.Fingerprint(arguments.Certificate)
	announce.AddCertificate(&fingerprint, arguments.Certificate)
	return nil
}

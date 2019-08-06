// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package announce - network announcements
//
// The DNS TXT record format is a set of space separated key=value pairs
//
//  Key      Value
//  =======  =========
//  bitmark  v3
//  a        Public IP addresses as IPv4;[IPv6]
//  c        Peer-To-Peer port number (decimal)
//  r        RPC port number (decimal)
//  f        SHA3 fingerprint of the certificate used by RPC connection for TLS verification (hex)
//  p        Public key of the P2P connection for ZeroMQ encryption (hex)
//
package announce

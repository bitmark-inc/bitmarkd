// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package cache maintains the memory data store
//
//  ***** Data Structure *****
//
//  Pool                      Key                     Value                       ExpiresAfter
//  |___ PendingTransfer	    merkle.Digest (link)    merkle.Digest (txid)        72h
//  |___ UnverifiedTxIndex	  merkle.Digest (txid)    pay.PayId                   72h
//  |___ UnverifiedTxEntries  pay.PayId               reservoir.unverifiedItem    72h
//  |___ VerifiedTx           merkle.Digest (txid)    reservoir.verifiedItem      never
//  |___ OrphanPayment        pay.PayId               reservoir.orphanPayment     72h
//
//
//  link ---------> txid -------------> payid ---------> unverifiedItem
//   |________________|___________________|_____________________|
//    PendingTransfer   UnverifiedTxIndex   UnverifiedTxEntries
//
//  ***** Purpose *****
//
//  PendingTransfer:
//    indexed by link so that duplicate transfers can be detected
//    data is the tx id so that the same transfer repeated can be distinguished
//    from an invalid duplicate transfer
//
//  UnverifiedTxIndex & UnverifiedTxEntries:
//    unverified transaction (issue and transfer)
//
//  VerifiedTx:
//    verified transaction (issue and transfer)
//
//  OrphanPayment:
//    when possible payment is already found, but no transfer transaction is received yet
//    put the payment in the pool to wait for
package cache

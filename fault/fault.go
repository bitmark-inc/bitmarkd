// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// error instances
//
// Provides a single instance of errors to allow easy comparison
package fault

// error base
type GenericError string

// to allow for different classes of errors
type ExistsError GenericError
type InvalidError GenericError
type NotFoundError GenericError
type ProcessError GenericError

// common errors - keep in alphabetic order
var (
	ErrKeyLength                = InvalidError("key length is invalid")
	ErrPasswordLength           = InvalidError("password length is invalid")
	ErrVerifiedPassword         = InvalidError("verified password is different")
	ErrRequiredConfigDir        = InvalidError("config folder is required")
	ErrConfigDirPath            = InvalidError("config is not a folder")
	ErrProvenanceRequestFail    = InvalidError("provenance request failed")
	ErrRequiredConnect          = InvalidError("connect is required")
	ErrRequiredIdentity         = InvalidError("identity is required")
	ErrRequiredDescription      = InvalidError("description is required")
	ErrRequiredAssetName        = InvalidError("asset name is required")
	ErrRequiredAssetDescription = InvalidError("asset description is required")
	ErrRequiredAssetFingerprint = InvalidError("asset fingerprint is required")
	ErrRequiredPayId            = InvalidError("payment id is required")
	ErrRequiredReceipt          = InvalidError("receipt id is required")
	ErrRequiredTransferTo       = InvalidError("transfer to is required")
	ErrRequiredTransferTxId     = InvalidError("transaction id is required")
	ErrWrongPassword            = InvalidError("wrong password")
	ErrInvalidSignature         = InvalidError("invalid signature")
	ErrNotFoundIdentity         = NotFoundError("identity name is invalid")
	ErrNotFoundConfigFile       = NotFoundError("config file is not found")
	ErrJsonParseFail            = ProcessError("parse to json failed")
	ErrIssueRequestFail         = ProcessError("send issue request failed")
	ErrAssetRequestFail         = ProcessError("send asset request failed")
	ErrTransferRequestFail      = ProcessError("send transfer request failed")
	ErrReceiptRequestFail       = ProcessError("send receipt request failed")
	ErrNodeInfoRequestFail      = ProcessError("send info request failed")
	ErrMakeIssueFail            = ProcessError("make issue failed")
	ErrMakeAssetFail            = ProcessError("make asset failed")
	ErrMakeTransferFail         = ProcessError("make transfer failed")
	ErrUnmarshalTextFail        = ProcessError("unmarshal text failed")
)

// the error interface base method
func (e GenericError) Error() string { return string(e) }

// the error interface methods
func (e ExistsError) Error() string   { return string(e) }
func (e InvalidError) Error() string  { return string(e) }
func (e NotFoundError) Error() string { return string(e) }
func (e ProcessError) Error() string  { return string(e) }

// determine the class of an error
func IsErrExists(e error) bool   { _, ok := e.(ExistsError); return ok }
func IsErrInvalid(e error) bool  { _, ok := e.(InvalidError); return ok }
func IsErrNotFound(e error) bool { _, ok := e.(NotFoundError); return ok }
func IsErrProcess(e error) bool  { _, ok := e.(ProcessError); return ok }

// Copyright (c) 2014-2017 Bitmark Inc.
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
	ErrAssetMetadataMustBeMap   = InvalidError("asset metadata must be map")
	ErrAssetRequestFail         = ProcessError("send asset request failed")
	ErrInvalidPrivateKey        = InvalidError("invalid private key")
	ErrInvalidSignature         = InvalidError("invalid signature")
	ErrKeyLength                = InvalidError("key length is invalid")
	ErrMakeIssueFail            = ProcessError("make issue failed")
	ErrMakeTransferFail         = ProcessError("make transfer failed")
	ErrNotFoundConfigFile       = NotFoundError("config file not found")
	ErrNotFoundIdentity         = NotFoundError("identity name not found")
	ErrPasswordLength           = InvalidError("password length is invalid")
	ErrProvenanceRequestFail    = InvalidError("provenance request failed")
	ErrRequiredAssetMetadata    = InvalidError("asset metadata is required")
	ErrRequiredAssetFingerprint = InvalidError("asset fingerprint is required")
	ErrRequiredAssetName        = InvalidError("asset name is required")
	ErrRequiredConfigFile       = InvalidError("config file is required")
	ErrRequiredConnect          = InvalidError("connect is required")
	ErrRequiredDescription      = InvalidError("description is required")
	ErrRequiredFileName         = InvalidError("file name is required")
	ErrRequiredIdentity         = InvalidError("identity is required")
	ErrRequiredPayId            = InvalidError("payment id is required")
	ErrRequiredPublicKey        = InvalidError("public key is required")
	ErrRequiredReceipt          = InvalidError("receipt id is required")
	ErrRequiredTransferTo       = InvalidError("transfer to is required")
	ErrRequiredTransferTxId     = InvalidError("transaction id is required")
	ErrUnableToRegenerateKeys   = InvalidError("unable to regenerate keys")
	ErrUnmarshalTextFail        = ProcessError("unmarshal text failed")
	ErrVerifiedPassword         = InvalidError("verified password is different")
	ErrWrongPassword            = InvalidError("wrong password")
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

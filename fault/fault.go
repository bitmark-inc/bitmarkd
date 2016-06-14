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
	ErrKeyLength                = InvalidError("Key Length is invalid")
	ErrPasswordLength           = InvalidError("Password Length is invalid")
	ErrVerifiedPassword         = InvalidError("Verified password is different")
	ErrRequiredConfig           = InvalidError("Config file is required")
	ErrConfigDirPath            = InvalidError("Config is not a folder")
	ErrRequiredConnect          = InvalidError("connect is required")
	ErrRequiredIdentity         = InvalidError("identity is required")
	ErrRequiredDescription      = InvalidError("description is required")
	ErrRequiredAssetName        = InvalidError("Asset name is required")
	ErrRequiredAssetDescription = InvalidError("Asset description is required")
	ErrRequiredAssetFingerprint = InvalidError("Asset fingerprint is required")
	ErrRequiredTransferTo       = InvalidError("Transfer to is required")
	ErrRequiredTransferTxId     = InvalidError("Transaction id is required")
	ErrWrongPassword            = InvalidError("Wrong password")
	ErrInvalidSignature         = InvalidError("Invalid signature")
	ErrInvalidPrivateKey        = InvalidError("Invalid privateKey")
	ErrNotFoundIdentity         = NotFoundError("Identity name is invalid")
	ErrNotFoundConfigFile       = NotFoundError("Config file is not found")
	ErrJsonParseFail            = ProcessError("Parse to json failed")
	ErrIssueRequestFail         = ProcessError("Send issue request failed")
	ErrAssetRequestFail         = ProcessError("Send asset request failed")
	ErrTransferRequestFail      = ProcessError("Send transfer request failed")
	ErrNodeInfoRequestFail      = ProcessError("Send info request failed")
	ErrMakeIssueFail            = ProcessError("Make issue failed")
	ErrMakeAssetFail            = ProcessError("Make asset failed")
	ErrMakeTransferFail         = ProcessError("Make transfer failed")
	ErrUnmarshalTextFail        = ProcessError("UnmarshalText failed")
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

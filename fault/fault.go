// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fault

import "errors"

// genericError - error base class
type genericError string

// to allow for different classes of errors
type existsError genericError
type invalidError genericError
type lengthError genericError
type notFoundError genericError
type processError genericError
type recordError genericError
type valueError genericError
type validationError genericError

// common errors - keep in alphabetic order
var (
	ErrAddressIsNil                          = processError("address is nil")
	ErrAlreadyInitialised                    = existsError("already initialised")
	ErrAssetNotFound                         = notFoundError("asset not found")
	ErrAssetsAlreadyRegistered               = invalidError("assets already registered")
	ErrBitcoinAddressForWrongNetwork         = invalidError("bitcoin address for wrong network")
	ErrBitcoinAddressIsNotSupported          = invalidError("bitcoin address is not supported")
	ErrBlockNotFound                         = notFoundError("block not found")
	ErrBlockVersionMustNotDecrease           = invalidError("block version must not decrease")
	ErrBufferCapacityLimit                   = lengthError("buffer capacity limit")
	ErrCannotConvertSharesBackToAssets       = invalidError("cannot convert shares back to assets")
	ErrCannotDecodeAccount                   = recordError("cannot decode account")
	ErrCannotDecodePrivateKey                = recordError("cannot decode private key")
	ErrCannotDecodeSeed                      = recordError("cannot decode seed")
	ErrCanOnlyConvertAssetsToShares          = invalidError("can only convert assets to shares")
	ErrCertificateFileAlreadyExists          = existsError("certificate file already exists")
	ErrCertificateFileNotFound               = notFoundError("cerfificate file not found")
	ErrChecksumMismatch                      = processError("checksum mismatch")
	ErrConnectingToSelfForbidden             = processError("connecting to self forbidden")
	ErrCurrencyIsNotSupportedByProofer       = invalidError("currency is not supported by proofer")
	ErrDoubleTransferAttempt                 = invalidError("double transfer attempt")
	ErrFingerprintTooLong                    = lengthError("fingerprint too long")
	ErrFingerprintTooShort                   = lengthError("fingerprint too short")
	ErrHeightOutOfSequence                   = invalidError("height out of sequence")
	ErrIdentityNameNotFound                  = notFoundError("identity name not found")
	ErrIncorrectChain                        = invalidError("incorrect chain")
	ErrInitialisationFailed                  = invalidError("initialisation failed")
	ErrInsufficientShares                    = invalidError("insufficient shares")
	ErrInvalidBitcoinAddress                 = invalidError("invalid bitcoin address")
	ErrInvalidBlockHeaderDifficulty          = invalidError("invalid block header difficulty")
	ErrInvalidBlockHeaderSize                = invalidError("invalid block header size")
	ErrInvalidBlockHeaderTimestamp           = invalidError("invalid block header timestamp")
	ErrInvalidBlockHeaderVersion             = invalidError("invalid block header version")
	ErrInvalidBuffer                         = invalidError("invalid buffer")
	ErrInvalidChain                          = invalidError("invalid chain")
	ErrInvalidCount                          = invalidError("invalid count")
	ErrInvalidCurrency                       = invalidError("invalid currency")
	ErrInvalidCurrencyAddress                = invalidError("invalid currency address")
	ErrInvalidCursor                         = invalidError("invalid cursor")
	ErrInvalidDnsTxtRecord                   = invalidError("invalid dns txt record")
	ErrInvalidFingerprint                    = invalidError("invalid fingerprint")
	ErrInvalidIPAddress                      = invalidError("invalid IP Address")
	ErrInvalidItem                           = invalidError("invalid item")
	ErrInvalidKeyLength                      = invalidError("invalid key length")
	ErrInvalidKeyType                        = invalidError("invalid key type")
	ErrInvalidLength                         = invalidError("invalid length")
	ErrInvalidLitecoinAddress                = invalidError("invalid litecoin address")
	ErrInvalidNodeDomain                     = invalidError("invalid node domain")
	ErrInvalidNonce                          = invalidError("invalid nonce")
	ErrInvalidOwnerOrRegistrant              = invalidError("invalid owner or registrant")
	ErrInvalidPaymentVersion                 = invalidError("invalid Payment version")
	ErrInvalidPeerResponse                   = invalidError("invalid peer response")
	ErrInvalidPortNumber                     = invalidError("invalid port number")
	ErrInvalidPrivateKey                     = invalidError("invalid private key")
	ErrInvalidPrivateKeyFile                 = invalidError("invalid private key file")
	ErrInvalidProofSigningKey                = invalidError("invalid proof signing key")
	ErrInvalidPublicKey                      = invalidError("invalid public key")
	ErrInvalidPublicKeyFile                  = invalidError("invalid public key file")
	ErrInvalidSeedHeader                     = invalidError("invalid seed header")
	ErrInvalidSeedLength                     = invalidError("invalid seed length")
	ErrInvalidSignature                      = invalidError("invalid signature")
	ErrInvalidStructPointer                  = invalidError("invalid struct pointer")
	ErrInvalidTimestamp                      = invalidError("invalid timestamp")
	ErrInvalidVersion                        = invalidError("invalid version")
	ErrKeyFileAlreadyExists                  = existsError("key file already exists")
	ErrKeyFileNotFound                       = notFoundError("key file not found")
	ErrKeyLengthIsInvalid                    = invalidError("key length is invalid")
	ErrLinkToInvalidOrUnconfirmedTransaction = invalidError("link to invalid or unconfirmed transaction")
	ErrLitecoinAddressForWrongNetwork        = invalidError("litecoin address for wrong network")
	ErrLitecoinAddressIsNotSupported         = invalidError("litecoin address is not supported")
	ErrMerkleRootDoesNotMatch                = invalidError("Merkle Root Does Not Match")
	ErrMetadataIsNotMap                      = invalidError("metadata is not map")
	ErrMetadataTooLong                       = lengthError("metadata too long")
	ErrMissingBlockOwner                     = lengthError("missing block owner")
	ErrMissingOwnerData                      = notFoundError("missing owner data")
	ErrMissingParameters                     = lengthError("missing parameters")
	ErrNameTooLong                           = lengthError("name too long")
	ErrNoConnectionsAvailable                = invalidError("no connections available")
	ErrNoNewTransactions                     = invalidError("no new transactions")
	ErrNotAPayId                             = invalidError("not a pay id")
	ErrNotAPayNonce                          = invalidError("not a pay nonce")
	ErrNotAssetIdentifier                    = recordError("not asset id")
	ErrNotAvailableDuringSynchronise         = invalidError("not available during synchronise")
	ErrNotConnected                          = notFoundError("not connected")
	ErrNotACountersignableRecord             = invalidError("not a countersignable record")
	ErrNotInitialised                        = notFoundError("not initialised")
	ErrNotLink                               = recordError("not link")
	ErrNotOwnedItem                          = invalidError("not owned item")
	ErrNotPrivateKey                         = recordError("not private key")
	ErrNotPublicKey                          = recordError("not public key")
	ErrNotOwnerDataPack                      = recordError("not owner data pack")
	ErrNotTransactionPack                    = recordError("not transaction pack")
	ErrOutOfPlaceBaseData                    = invalidError("out of place base data")
	ErrOutOfPlaceBlockOwnerIssue             = invalidError("out of place block owner issue")
	ErrPayIdAlreadyUsed                      = invalidError("payId already used")
	ErrPaymentAddressTooLong                 = lengthError("payment address too long")
	ErrPreviousBlockDigestDoesNotMatch       = invalidError("previous block digest does not match")
	ErrRateLimiting                          = lengthError("rate limiting")
	ErrReceiptTooLong                        = lengthError("receipt too long")
	ErrRecordHasExpired                      = invalidError("record has expired")
	ErrSignatureTooLong                      = lengthError("signature too long")
	ErrShareIdsCannotBeIdentical             = valueError("share ids cannot be identical")
	ErrShareQuantityTooSmall                 = valueError("share quantity too small")
	ErrTooManyItemsToProcess                 = lengthError("too many items to process")
	ErrTransactionCountOutOfRange            = lengthError("transaction count out of range")
	ErrTransactionAlreadyExists              = existsError("transaction already exists")
	ErrTransactionIsNotATransfer             = invalidError("transaction is not a transfer")
	ErrTransactionIsNotAnAsset               = invalidError("transaction is not an asset")
	ErrTransactionIsNotAnIssue               = invalidError("transaction is not an issue")
	ErrTransactionIsNotAnIssueOrATransfer    = invalidError("transaction is not an issue or a transfer")
	ErrTransactionLinksToSelf                = recordError("transaction links to self")
	ErrWrongNetworkForPrivateKey             = invalidError("wrong network for private key")
	ErrWrongNetworkForPublicKey              = invalidError("wrong network for public key")
	ErrOwnershipIsNotIndexed                 = validationError("ownership is not indexed")
	ErrOwnershipIsNotCleaned                 = validationError("ownership is not cleaned")
	ErrTransactionIsNotCleaned               = validationError("transaction is not cleaned")
	ErrAssetIsNotIndexed                     = validationError("asset is not indexed")
	ErrTransactionIsNotIndexed               = validationError("transaction is not indexed")
	ErrDataInconsistent                      = validationError("data inconsistent")
	ErrUnableToRegenerateKeys                = invalidError("unable to regenerate keys")
	ErrUnexpectedTransaction                 = recordError("unexpected transaction record")
	ErrUnmarshalTextFailed                   = processError("unmarshal text failed")
	ErrWrongPassword                         = invalidError("wrong password")
)

var (
	ErrMakeBlockTransferFailed  = processError("make block transfer failed")
	ErrMakeGrantFailed          = processError("make grant failed")
	ErrMakeIssueFailed          = processError("make issue failed")
	ErrMakeShareFailed          = processError("make share failed")
	ErrMakeSwapFailed           = processError("make swap failed")
	ErrMakeTransferFailed       = processError("make transfer failed")
	ErrInvalidPasswordLength    = invalidError("invalid password length")
	ErrKeyPairCannotBeNil       = invalidError("key pair cannot be nil")
	ErrPasswordMismatch         = invalidError("password mismatch")
	ErrAssetMetadataMustBeMap   = invalidError("asset metadata must be map")
	ErrRequiredAssetFingerprint = invalidError("asset fingerprint is required")
	ErrRequiredAssetMetadata    = invalidError("asset metadata is required")
	ErrRequiredAssetName        = invalidError("asset name is required")
	ErrRequiredConnect          = invalidError("connect is required use: HOST:PORT or IPv4:PORT or [IPv6]:PORT")
	ErrRequiredConnectPort      = invalidError("connect requires :PORT-NUMBER suffix")
	ErrRequiredCurrencyAddress  = invalidError("currency address is required")
	ErrRequiredDescription      = invalidError("description is required")
	ErrRequiredFileName         = invalidError("file name is required")
	ErrRequiredIdentity         = invalidError("identity is required")
	ErrRequiredPayId            = invalidError("payment id is required")
	ErrRequiredPublicKey        = invalidError("public key is required")
	ErrRequiredReceipt          = invalidError("receipt id is required")
	ErrRequiredTransferTo       = invalidError("transfer to is required")
	ErrRequiredTransferTx       = invalidError("transaction hex data is required")
	ErrRequiredTxId             = invalidError("transaction id is required")
)

// the error interface base method
func (e genericError) Error() string { return string(e) }

// the error interface methods
func (e existsError) Error() string     { return string(e) }
func (e invalidError) Error() string    { return string(e) }
func (e lengthError) Error() string     { return string(e) }
func (e notFoundError) Error() string   { return string(e) }
func (e processError) Error() string    { return string(e) }
func (e recordError) Error() string     { return string(e) }
func (e valueError) Error() string      { return string(e) }
func (e validationError) Error() string { return string(e) }

// determine the class of an error
func IsErrExists(e error) bool     { _, ok := e.(existsError); return ok }
func IsErrInvalid(e error) bool    { _, ok := e.(invalidError); return ok }
func IsErrLength(e error) bool     { _, ok := e.(lengthError); return ok }
func IsErrNotFound(e error) bool   { _, ok := e.(notFoundError); return ok }
func IsErrProcess(e error) bool    { _, ok := e.(processError); return ok }
func IsErrRecord(e error) bool     { _, ok := e.(recordError); return ok }
func IsErrValue(e error) bool      { _, ok := e.(valueError); return ok }
func IsErrValidation(e error) bool { _, ok := e.(validationError); return ok }

// ErrorFromRunes - convert a byte slice to a limited length error
func ErrorFromRunes(buffer []byte) error {
	s := []rune(string(buffer))
	if len(s) > 30 {
		s = s[:30]
	}
	return errors.New(string(s))
}

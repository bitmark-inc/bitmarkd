// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fault

// error base
type GenericError string

// to allow for different classes of errors
type ExistsError GenericError
type InvalidError GenericError
type LengthError GenericError
type NotFoundError GenericError
type ProcessError GenericError
type RecordError GenericError

// common errors - keep in alphabetic order
var (
	ErrAlreadyInitialised                    = ExistsError("already initialised")
	ErrAssetNotFound                         = NotFoundError("asset not found")
	ErrAssetsAlreadyRegistered               = InvalidError("assets already registered")
	ErrBitcoinAddressForWrongNetwork         = InvalidError("bitcoin address for wrong network")
	ErrBitcoinAddressIsNotSupported          = InvalidError("bitcoin address is not supported")
	ErrBlockNotFound                         = NotFoundError("block not found")
	ErrCannotDecodeAccount                   = RecordError("cannot decode account")
	ErrCannotDecodePrivateKey                = RecordError("cannot decode private key")
	ErrCannotDecodeSeed                      = RecordError("cannot decode seed")
	ErrCertificateFileAlreadyExists          = ExistsError("certificate file already exists")
	ErrChecksumMismatch                      = ProcessError("checksum mismatch")
	ErrConnectingToSelfForbidden             = ProcessError("connecting to self forbidden")
	ErrCurrencyIsNotSupportedByProofer       = InvalidError("currency is not supported by proofer")
	ErrDoubleTransferAttempt                 = InvalidError("double transfer attempt")
	ErrFingerprintTooLong                    = LengthError("fingerprint too long")
	ErrFingerprintTooShort                   = LengthError("fingerprint too short")
	ErrIncorrectChain                        = InvalidError("incorrect chain")
	ErrInitialisationFailed                  = InvalidError("initialisation failed")
	ErrInvalidBitcoinAddress                 = InvalidError("invalid bitcoin address")
	ErrInvalidBlockHeader                    = InvalidError("invalid block header")
	ErrInvalidChain                          = InvalidError("invalid chain")
	ErrInvalidCount                          = InvalidError("invalid count")
	ErrInvalidCurrency                       = InvalidError("invalid currency")
	ErrInvalidCursor                         = InvalidError("invalid cursor")
	ErrInvalidDnsTxtRecord                   = InvalidError("invalid dns txt record")
	ErrInvalidFingerprint                    = InvalidError("invalid fingerprint")
	ErrInvalidIPAddress                      = InvalidError("invalid IP Address")
	ErrInvalidKeyLength                      = InvalidError("invalid key length")
	ErrInvalidKeyType                        = InvalidError("invalid key type")
	ErrInvalidLength                         = InvalidError("invalid length")
	ErrInvalidLitecoinAddress                = InvalidError("invalid litecoin address")
	ErrInvalidLoggerChannel                  = InvalidError("invalid logger channel")
	ErrInvalidNonce                          = InvalidError("invalid nonce")
	ErrInvalidOwnerOrRegistrant              = InvalidError("invalid owner or registrant")
	ErrInvalidPeerResponse                   = InvalidError("invalid peer response")
	ErrInvalidPortNumber                     = InvalidError("invalid port number")
	ErrInvalidPrivateKey                     = InvalidError("invalid private key")
	ErrInvalidPrivateKeyFile                 = InvalidError("invalid private key file")
	ErrInvalidProofSigningKey                = InvalidError("invalid proof signing key")
	ErrInvalidPublicKey                      = InvalidError("invalid public key")
	ErrInvalidPublicKeyFile                  = InvalidError("invalid public key file")
	ErrInvalidSeedHeader                     = InvalidError("invalid seed header")
	ErrInvalidSeedLength                     = InvalidError("invalid seed length")
	ErrInvalidSignature                      = InvalidError("invalid signature")
	ErrInvalidStructPointer                  = InvalidError("invalid struct pointer")
	ErrInvalidVersion                        = InvalidError("invalid version")
	ErrKeyFileAlreadyExists                  = ExistsError("key file already exists")
	ErrKeyFileNotFound                       = NotFoundError("key file not found")
	ErrLinkToInvalidOrUnconfirmedTransaction = InvalidError("link to invalid or unconfirmed transaction")
	ErrLitecoinAddressForWrongNetwork        = InvalidError("litecoin address for wrong network")
	ErrLitecoinAddressIsNotSupported         = InvalidError("litecoin address is not supported")
	ErrMerkleRootDoesNotMatch                = InvalidError("Merkle Root Does Not Match")
	ErrMetadataIsNotMap                      = InvalidError("metadata is not map")
	ErrMetadataTooLong                       = LengthError("metadata too long")
	ErrMissingParameters                     = LengthError("missing parameters")
	ErrNameTooLong                           = LengthError("name too long")
	ErrNameTooShort                          = LengthError("name too short")
	ErrNoConnectionsAvailable                = InvalidError("no connections available")
	ErrNoNewTransactions                     = InvalidError("no new transactions")
	ErrNotAPayId                             = InvalidError("not a pay id")
	ErrNotAPayNonce                          = InvalidError("not a pay nonce")
	ErrNotAssetIndex                         = RecordError("not asset index")
	ErrNotAvailableDuringSynchronise         = InvalidError("not available during synchronise")
	ErrNotConnected                          = NotFoundError("not connected")
	ErrNotInitialised                        = NotFoundError("not initialised")
	ErrNotLink                               = RecordError("not link")
	ErrNotPrivateKey                         = RecordError("not private key")
	ErrNotPublicKey                          = RecordError("not public key")
	ErrNotTransactionPack                    = RecordError("not transaction pack")
	ErrOutOfPlaceBaseData                    = InvalidError("out of place base data")
	ErrPayIdAlreadyUsed                      = InvalidError("payId already used")
	ErrPaymentAddressTooLong                 = LengthError("payment address too long")
	ErrPreviousBlockDigestDoesNotMatch       = InvalidError("previous block digest does not match")
	ErrReceiptTooLong                        = LengthError("receipt too long")
	ErrSignatureTooLong                      = LengthError("signature too long")
	ErrTooManyItemsToProcess                 = LengthError("too many items to process")
	ErrTransactionAlreadyExists              = ExistsError("transaction already exists")
	ErrTransactionIsNotATransfer             = InvalidError("transaction is not a transfer")
	ErrTransactionIsNotAnAsset               = InvalidError("transaction is not an asset")
	ErrTransactionIsNotAnIssue               = InvalidError("transaction is not an issue")
	ErrTransactionIsNotAnIssueOrATransfer    = InvalidError("transaction is not an issue or a transfer")
	ErrTransactionLinksToSelf                = RecordError("transaction links to self")
	ErrUnexpectedNilPointer                  = ProcessError("unexpected nil pointer")
	ErrWrongNetworkForPrivateKey             = InvalidError("wrong network for private key")
	ErrWrongNetworkForPublicKey              = InvalidError("wrong network for public key")
)

// the error interface base method
func (e GenericError) Error() string { return string(e) }

// the error interface methods
func (e ExistsError) Error() string   { return string(e) }
func (e InvalidError) Error() string  { return string(e) }
func (e LengthError) Error() string   { return string(e) }
func (e NotFoundError) Error() string { return string(e) }
func (e ProcessError) Error() string  { return string(e) }
func (e RecordError) Error() string   { return string(e) }

// determine the class of an error
func IsErrExists(e error) bool   { _, ok := e.(ExistsError); return ok }
func IsErrInvalid(e error) bool  { _, ok := e.(InvalidError); return ok }
func IsErrLength(e error) bool   { _, ok := e.(LengthError); return ok }
func IsErrNotFound(e error) bool { _, ok := e.(NotFoundError); return ok }
func IsErrProcess(e error) bool  { _, ok := e.(ProcessError); return ok }
func IsErrRecord(e error) bool   { _, ok := e.(RecordError); return ok }

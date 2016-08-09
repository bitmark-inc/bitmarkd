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
type LengthError GenericError
type NotFoundError GenericError
type ProcessError GenericError
type RecordError GenericError

// common errors - keep in alphabetic order
var (
	ErrAlreadyInitialised = ExistsError("already initialised")
	ErrAssetNotFound      = NotFoundError("asset not found")
	//ErrBlockHashDoesNotMeetDifficulty = InvalidError("block hash does not meet difficulty")
	ErrBlockNotFound                = NotFoundError("block not found")
	ErrCannotDecodeAccount          = RecordError("cannot decode account")
	ErrCertificateFileAlreadyExists = ExistsError("certificate file already exists")
	//ErrCertificateNotFound            = NotFoundError("certificate not found")
	ErrChecksumMismatch = ProcessError("checksum mismatch")
	//ErrCountMismatch                  = ProcessError("count mismatch")
	//ErrConnectingToSelfForbidden      = ProcessError("connecting to self forbidden")
	//ErrCurrencyTooLong                = LengthError("currency too long")
	ErrDescriptionTooLong = LengthError("description too long")
	ErrFingerprintTooLong = LengthError("fingerprint too long")
	//ErrFirstTransactionIsNotBase      = RecordError("first transaction is not base")
	//ErrInsufficientPayment            = InvalidError("insufficient payment")
	ErrInitialisationFailed = InvalidError("initialisation failed")
	ErrIncorrectChain       = InvalidError("incorrect chain")
	//ErrInvalidBitmarkDataRevision     = InvalidError("invalid bitmark data revision")
	//ErrInvalidBitmarkDataSize         = InvalidError("bitmark data size")
	//ErrInvalidBlock                   = InvalidError("invalid block")
	ErrInvalidBlockHeader    = InvalidError("invalid block header")
	ErrInvalidChain          = InvalidError("invalid chain")
	ErrInvalidCount          = InvalidError("invalid count")
	ErrInvalidCurrency       = InvalidError("invalid currency")
	ErrInvalidCursor         = InvalidError("invalid cursor")
	ErrInvalidIPAddress      = InvalidError("invalid IP Address")
	ErrInvalidFingerprint    = InvalidError("invalid fingerprint")
	ErrInvalidPeerResponse   = InvalidError("invalid peer response")
	ErrInvalidPrivateKeyFile = InvalidError("invalid private key file")
	ErrInvalidPublicKeyFile  = InvalidError("invalid public key file")
	ErrInvalidPublicKey      = InvalidError("invalid public key")
	ErrInvalidKeyLength      = InvalidError("invalid key length")
	ErrInvalidKeyType        = InvalidError("invalid key type")
	ErrInvalidLength         = InvalidError("invalid length")
	ErrInvalidLoggerChannel  = InvalidError("invalid logger channel")

	ErrInvalidPortNumber    = InvalidError("invalid port number")
	ErrInvalidNonce         = InvalidError("invalid nonce")
	ErrInvalidSignature     = InvalidError("invalid signature")
	ErrInvalidStructPointer = InvalidError("invalid struct pointer")
	//ErrInvalidTransactionChain        = InvalidError("invalid transaction chain")
	ErrInvalidDnsTxtRecord = InvalidError("invalid dns txt record")
	//ErrInvalidType                    = InvalidError("invalid type")
	ErrInvalidVersion       = InvalidError("invalid version")
	ErrKeyFileAlreadyExists = ExistsError("key file already exists")
	ErrKeyFileNotFound      = NotFoundError("key file not found")
	//ErrKeyNotFound                    = NotFoundError("key not found")
	//ErrLinkNotFound                   = NotFoundError("link not found")
	//ErrLinksToUnconfirmedTransaction  = InvalidError("links to unconfirmed transaction")
	ErrMerkleRootDoesNotMatch = InvalidError("Merkle Root Does Not Match")
	//ErrMessagingTerminated            = ProcessError("messaging terminated")
	ErrMissingParameters      = LengthError("missing parameters")
	ErrNameTooLong            = LengthError("name too long")
	ErrNoConnectionsAvailable = InvalidError("no connections available")
	//ErrNoPaymentToMiner               = InvalidError("no payment to miner")
	//ErrNotABitmarkPayment             = InvalidError("not a bitmark payment")
	ErrNotAvailableDuringSynchronise = InvalidError("not available during synchronise")
	ErrNotAssetIndex                 = RecordError("not asset index")
	ErrNotConnected                  = NotFoundError("not connected")
	//ErrNotCurrentOwner                = RecordError("not current owner")
	ErrNotInitialised = NotFoundError("not initialised")
	ErrNotLink        = RecordError("not link")
	ErrNotAPayId      = InvalidError("not a pay id")
	ErrNotAPayNonce   = InvalidError("not a pay nonce")
	ErrNotPublicKey   = RecordError("not public key")
	//ErrNotTransactionType             = RecordError("not transaction type")
	ErrNotTransactionPack = RecordError("not transaction pack")
	//ErrPaymentAddressMissing          = NotFoundError("payment address missing")
	ErrPaymentAddressTooLong = LengthError("payment address too long")
	////ErrPeerAlreadyExists              = ExistsError("peer already exists")
	////ErrPeerNotFound                   = NotFoundError("peer not found")
	ErrPreviousBlockDigestDoesNotMatch = InvalidError("previous block digest does not match")
	ErrSignatureTooLong                = LengthError("signature too long")
	ErrTooManyItemsToProcess           = LengthError("too many items to process")
	////ErrTooManyTransactionsInBlock     = LengthError("too many transactions in block")
	ErrTransactionAlreadyExists = ExistsError("transaction already exists")
	ErrWrongNetworkForPublicKey = InvalidError("wrong network for public key")
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

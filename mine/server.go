// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mine

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/gnomon"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
	"io"
	"sync/atomic"
	"time"
)

// global data
var globalMinerCount int64

// type to hold Peer
type Mining struct {
	log          *logger.L
	notifyId     string
	difficultyId string
	extraNonce1  []byte
	//argument *ServerArgument
	//m           sync.Mutex
	//value       int
}

// sizes for nonces
const (
	extraNonce1Size = 4
	extraNonce2Size = 4
	extraNonceSize  = extraNonce1Size + extraNonce2Size
)

// miner registrations
type minerRegistration struct {
	extraNonce1  []byte // this is unique per miner and a random value
	difficultyId string
}

// the active registrations
// indexed by notifyId
var activeRegistrations map[string]minerRegistration

// miner subscription
// ------------------

type SubscribeArguments struct {
	MinerName string `arg:"0"`
	NotifyId  string `arg:"1"`
}

type SubscribeReply []interface{}

func (mining *Mining) Subscribe(arguments SubscribeArguments, reply *SubscribeReply) error {

	// check if there is an existing registration
	if "" != arguments.NotifyId {
		if r, ok := activeRegistrations[arguments.NotifyId]; ok {
			*reply = SubscribeReply{
				[][]string{
					{"mining.set_difficulty", r.difficultyId},
					{"mining.notify", arguments.NotifyId},
				},
				r.extraNonce1,
				extraNonce2Size,
			}
			return nil
		}
	}

	// get some unique values
	difficultyId := gnomon.NewCursor().String()
	notifyId := gnomon.NewCursor().String()

	// random bytes for miner specific nonce
	extraNonce1 := make([]byte, extraNonce1Size)
	_, err := io.ReadFull(rand.Reader, extraNonce1)
	if nil != err {
		return ErrOtherUnknown
	}

	mining.notifyId = notifyId
	mining.difficultyId = difficultyId
	mining.extraNonce1 = extraNonce1

	// ***** FIX THIS: save above in activeRegistrations *****
	// but need to consider a way to expire these
	// do for now just issue a new value

	*reply = SubscribeReply{
		[][]string{
			{"mining.set_difficulty", difficultyId},
			{"mining.notify", notifyId},
		},
		hex.EncodeToString(extraNonce1),
		extraNonce2Size,
	}
	return nil
}

// miner authorisation
// -------------------

type AuthoriseArguments struct {
	Username string `arg:"0"`
	Password string `arg:"1"`
}

// miner log in
func (mining *Mining) Authorize(arguments AuthoriseArguments, reply *bool) error {

	// ***** FIX THIS: need a proper config setting *****
	if arguments.Username == arguments.Password {
		return ErrUnauthorizedWorker
	}

	*reply = true
	return nil
}

// miner submit
// ------------

type SubmitArguments struct {
	Username    string `arg:"0"` // miner identifier
	JobId       string `arg:"1"` // ID of the job from notification
	ExtraNonce2 string `arg:"2"` // ExtraNonce2 used by miner found by miner
	Ntime       string `arg:"3"` // Current ntime
	Nonce       string `arg:"4"` // Nonce fount by miner
}

func fromBE(s string) uint32 {
	return fromAny(s, binary.BigEndian)
}
func fromLE(s string) uint32 {
	return fromAny(s, binary.LittleEndian)
}
func fromAny(s string, endian binary.ByteOrder) uint32 {
	h, err := hex.DecodeString(s)
	if nil != err {
		fault.PanicWithError("mining.fromAny hex decode", err)
	}
	b := bytes.NewBuffer(h)
	n := uint32(0)
	err = binary.Read(b, endian, &n)
	if nil != err {
		fault.PanicWithError("mining.fromAny binary.Read", err)
	}
	return n
}

// miner submit result
func (mining *Mining) Submit(arguments SubmitArguments, reply *bool) error {

	jobId := stringToJobId(arguments.JobId)

	log := mining.log

	*reply = true

	extraNonce2, err := hex.DecodeString(arguments.ExtraNonce2)
	if nil != err {
		return err
	}

	ntime := fromBE(arguments.Ntime)
	nonce := fromBE(arguments.Nonce)

	log.Infof("*SUBMIT* from name = %s", arguments.Username)
	log.Infof("jobId = 0x%x", jobId)
	log.Infof("extraNonce1..2 = %x .. %x", mining.extraNonce1, extraNonce2)
	log.Infof("ntime = %x", ntime)
	log.Infof("nonce = %x", nonce)

	// need some kind of Locking starting here
	// ---------------------------------------

	ids, addresses, timestamp, ok := jobQueue.getIds(jobId)
	if !ok {
		log.Warn("job not found")
		return ErrJobNotFound
	}

	nonce12 := make([]byte, 0, extraNonceSize)
	nonce12 = append(nonce12, mining.extraNonce1...)
	nonce12 = append(nonce12, extraNonce2...)

	digest, blk, ok := block.MinerCheckIn(timestamp, ntime, nonce, nonce12, addresses, ids)
	if !ok {
		log.Warnf("difficulty NOT MET: %s", digest)
		return ErrLowDifficultyShare
	}

	log.Infof("difficulty met: digest: %s", digest)

	// mark the tx as confirmed
	for _, id := range jobQueue.confirm(jobId) {
		txid := transaction.Link(id)
		txid.SetState(transaction.ConfirmedTransaction)
	}

	messagebus.Send(block.Mined(blk))

	return nil
}

// --------------------

// send notifications
//
// The params array sent contains the following, in order:
//    job_id        - ID of the job. Use this ID while submitting share generated from this job.
//    prevhash      - Hash of previous block.
//    coinbase1     - Initial part of coinbase transaction.
//    coinbase2     - Final part of coinbase transaction.
//    merkle_branch - List of hashes, will be used for calculation of merkle root.
//                    This is not a list of all transactions, it only contains prepared hashes of steps of merkle tree algorithm.
//    version       - Bitcoin block version.
//    nbits         - Encoded current network difficulty.
//    ntime         - Current ntime.
//    clean_jobs    - When true, server indicates that submitting shares from previous jobs don't have a sense and such shares will be rejected.
//                    When this flag is set, miner should also drop all previous jobs, so job_ids can be eventually rotated.
func backgroundNotifier(conn Notifier, stop <-chan bool, argument interface{}) {

	serverArgument := argument.(*ServerArgument)
	log := serverArgument.Log

	log.Info("backgroundNotifier: starting…")
	interval := 6 * time.Second // one tenth of a minute

	// send out difficulty
	difficultyValue := difficulty.Current.Pdiff()
	conn.Notify("mining.set_difficulty", []interface{}{difficultyValue})

	currentJobId := jobIdentifierNil

loop:
	for {
		select {
		case <-stop:
			break loop
		case <-time.After(interval):
		}

		difficultySent := false

		// if difficulty changed, re-send
		if d := difficulty.Current.Pdiff(); d != difficultyValue {
			difficultyValue = d
			log.Infof("set difficulty: %v", d)
			conn.Notify("mining.set_difficulty", []interface{}{difficultyValue})
			difficultySent = true
		}
		log.Info("poll for new job")

		// get a job if available
		if jobId, minMerkle, addresses, timestamp, clean, ok := jobQueue.top(); ok && jobId != currentJobId {

			// to prevent duplicate job notifications
			currentJobId = jobId

			log.Infof("job id: %v  minMerkle: %v  addresses: %#v  clean: %v  ok: %v", jobId, minMerkle, addresses, clean, ok)

			if clean && !difficultySent {
				conn.Notify("mining.set_difficulty", []interface{}{difficulty.Current.Pdiff()})
			}

			cb1, cb2 := block.CurrentCoinbase(timestamp, extraNonce1Size+extraNonce2Size, addresses)

			// create minimum merkle tree
			hexMerkleTree := make([]string, len(minMerkle))
			for i, h := range minMerkle {
				hexMerkleTree[i] = h.MinerHex()
			}

			stringBlockVersion := fmt.Sprintf("%08x", block.Version)
			stringTimeNow := fmt.Sprintf("%08x", timestamp.Unix())

			notificationData := []interface{}{
				jobId.String(),                // [0] job_id
				block.PreviousLink().BtcHex(), // [1] previous link
				hex.EncodeToString(cb1),       // [2] coinbase 1
				hex.EncodeToString(cb2),       // [3] coinbase 2
				hexMerkleTree,                 // [4] minimised merkle tree
				stringBlockVersion,            // [5] version
				difficulty.Current.String(),   // [6] bits
				stringTimeNow,                 // [7] time
				clean,                         // [8] clean_jobs
			}

			log.Infof("mining.notify data: %v", notificationData)
			conn.Notify("mining.notify", notificationData)
		}
	}

	log.Info("backgroundNotifier: shutting down…")
}

type AB struct {
	A float64 `arg:"0"`
	B float64 `arg:"1"`
}

type CC struct {
	C float64 `json:"sum"`
}

func (mining *Mining) T(arguments AB, reply *CC) error {

	fmt.Printf("T(%f, %f)\n", arguments.A, arguments.B)
	if 0 == arguments.A {
		return ErrDuplicateShare
	}
	if 0 == arguments.B {
		return ErrJobNotFound
	}

	reply.C = arguments.A + arguments.B

	return nil
}

// the server thread
// -----------------

// the argument passed to the callback
type ServerArgument struct {
	Log *logger.L
}

// listener callback
func Callback(conn io.ReadWriteCloser, argument interface{}) {

	defer conn.Close()

	serverArgument := argument.(*ServerArgument)
	if nil == serverArgument {
		panic("mine: nil serverArgument")
	}
	if nil == serverArgument.Log {
		panic("mine: nil serverArgument.Log")
	}

	log := serverArgument.Log

	mining := &Mining{
		log: log,
	}

	//server := rpc.NewServer()
	server := NewServer()

	server.Register(mining)

	// count miner connections
	atomic.AddInt64(&globalMinerCount, 1)
	defer atomic.AddInt64(&globalMinerCount, -1)

	ServeConnection(conn, server, backgroundNotifier, serverArgument)
}

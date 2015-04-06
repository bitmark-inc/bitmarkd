// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mine

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/gnomon"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/transaction"
	rpc "github.com/bitmark-inc/go-rpc"             // "net/rpc"
	jsonrpc "github.com/bitmark-inc/go-rpc/jsonrpc" // "net/rpc/jsonrpc"
	"github.com/bitmark-inc/listener"
	"github.com/bitmark-inc/logger"
	"io"
	"sync/atomic"
	"time"
)

// error codes
var (
	ErrOtherUnknown       = errors.New("20")
	ErrJobNotFound        = errors.New("21")
	ErrDuplicateShare     = errors.New("22")
	ErrLowDifficultyShare = errors.New("23")
	ErrUnauthorizedWorker = errors.New("24")
	ErrNotSubscribed      = errors.New("25")
)

// global data
var globalMinerCount int64

// type for null arguments
type NoArguments struct{}

// for array of arguments
type GenericArguments []interface{}

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

// generic reply
type Reply []interface{}

// boolean reply
type BoolReply bool

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

// mining subscribe function
func (mining *Mining) Subscribe(arguments GenericArguments, reply *Reply) error {

	// check if there is an existing registration
	if len(arguments) >= 2 {
		notifyId := arguments[1].(string)
		if r, ok := activeRegistrations[notifyId]; ok {
			*reply = Reply{
				[][]string{
					{"mining.set_difficulty", r.difficultyId},
					{"mining.notify", notifyId},
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
	*reply = Reply{
		[][]string{
			{"mining.set_difficulty", difficultyId},
			{"mining.notify", notifyId},
		},
		hex.EncodeToString(extraNonce1),
		extraNonce2Size,
	}
	return nil
}

// miner log in
func (mining *Mining) Authorize(arguments GenericArguments, reply *BoolReply) error {
	if len(arguments) != 2 {
		return ErrOtherUnknown
	}
	name := arguments[0]
	password := arguments[1]

	// ***** FIX THIS: need a proper config setting *****
	if name == password {
		return ErrUnauthorizedWorker
	}

	*reply = true
	return nil
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
//
// params: [name, job_id, extranonce2, ntime, nonce]
func (mining *Mining) Submit(arguments GenericArguments, reply *BoolReply) error {
	if len(arguments) != 5 {
		return ErrOtherUnknown
	}
	name := arguments[0]
	jobId := stringToJobId(arguments[1].(string))

	log := mining.log

	*reply = true

	extraNonce2, err := hex.DecodeString(arguments[2].(string))
	if nil != err {
		return err
	}

	ntime := fromBE(arguments[3].(string))
	nonce := fromBE(arguments[4].(string))

	log.Infof("*SUBMIT* from name = %s", name)
	log.Infof("jobId = 0x%x", jobId)
	log.Infof("extraNonce1 = %x", mining.extraNonce1)
	log.Infof("extraNonce2 = %x", extraNonce2)
	log.Infof("ntime = %x", ntime)
	log.Infof("nonce = %x", nonce)

	// need some kind of Locking starting here
	// ---------------------------------------

	ids, addresses, timestamp, ok := jobQueue.getIds(jobId)
	if !ok {
		log.Warn("job not found")
		//return ErrJobNotFound
		return nil
	}

	nonce12 := make([]byte, 0, extraNonceSize)
	nonce12 = append(nonce12, mining.extraNonce1...)
	nonce12 = append(nonce12, extraNonce2...)

	digest, blk, ok := block.MinerCheckIn(timestamp, ntime, nonce, nonce12, addresses, ids)
	if !ok {
		log.Warnf("difficulty NOT MET")
		//return ErrLowDifficultyShare
		return nil
	}

	log.Infof("difficulty met: digest: %s", digest)

	// mark the tx as mined
	for _, id := range jobQueue.confirm(jobId) {
		txid := transaction.Link(id)
		txid.SetState(transaction.MinedTransaction)
	}

	messagebus.Send(block.Mined(blk))

	return nil
}

// --------------------

// the argument passed to the callback
type ServerArgument struct {
	Log *logger.L
}

// send notifications
func backgroundNotifier(log *logger.L, server *rpc.Server, stop <-chan bool) {

	log.Info("backgroundNotifier: starting…")
	interval := 6 * time.Second // one tenth of a minute

	//    job_id        - ID of the job. Use this ID while submitting share generated from this job.
	//    prevhash      - Hash of previous block.
	//    coinbase1     - Initial part of coinbase transaction.
	//    coinbase2     - Final part of coinbase transaction.
	//    merkle_branch - List of hashes, will be used for calculation of merkle root.
	//                    This is not a list of all transactions, it only contains prepared hashes of steps of merkle tree algorithm.
	//    version       - Bitcoin block version.
	//    nbits         - Encoded current network difficulty
	//    ntime         - Current ntime
	//    clean_jobs    - When true, server indicates that submitting shares from previous jobs don't have a sense and such shares will be rejected.
	//                    When this flag is set, miner should also drop all previous jobs, so job_ids can be eventually rotated.

	// send out difficulty
	difficultyValue := difficulty.Current.Float64()
	server.SendNotification("mining.set_difficulty", []interface{}{difficultyValue})

	currentJobId := jobIdentifierNil

loop:
	for {
		select {
		case <-stop:
			break loop
		case <-time.After(interval):
		}

		// if difficulty changed, re-send
		if d := difficulty.Current.Float64(); d != difficultyValue {
			difficultyValue = d
			log.Infof("set difficulty: %v", d)
			server.SendNotification("mining.set_difficulty", []interface{}{difficultyValue})
		}

		log.Info("poll for new job")

		// get a job if available
		if jobId, minMerkle, addresses, timestamp, clean, ok := jobQueue.top(); ok && jobId != currentJobId {

			// to prevent duplicate job notifications
			currentJobId = jobId

			log.Infof("job id: %v  minMerkle: %v  addresses: %#v  clean: %v  ok: %v", jobId, minMerkle, addresses, clean, ok)

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
			server.SendNotification("mining.notify", notificationData)
		}
	}

	log.Info("backgroundNotifier: shutting down…")
}

// listener callback
func Callback(conn *listener.ClientConnection, argument interface{}) {

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

	shutdown := make(chan bool)
	defer close(shutdown)

	server := rpc.NewServer()

	server.Register(mining)

	shutdownNotifier := make(chan bool)
	defer close(shutdownNotifier)
	go backgroundNotifier(log, server, shutdownNotifier)

	codec := jsonrpc.NewServerCodec(conn)
	defer codec.Close()

	// count miner connections
	atomic.AddInt64(&globalMinerCount, 1)
	defer atomic.AddInt64(&globalMinerCount, -1)

	// allow miner to connect
	server.ServeCodec(codec)
}

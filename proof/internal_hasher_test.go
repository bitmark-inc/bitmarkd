package proof_test

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	zmq "github.com/pebbe/zmq4"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/proof"
)

const (
	endpointRequestStr         = "inproc://internal-hasher-request-test"
	endpointReplyStr           = "inproc://internal-hasher-reply-test"
	testInproc1                = "inproc://test1"
	testInproc2                = "inproc://test2"
	wrongEndpointRequestString = "tcp://wrong-request"
	wrongEndpointReplyString   = "tcp://wrong-reply"
	nonceStart                 = 1
	protocol                   = zmq.PAIR
)

func TestNewInternalHasherForTestWhenInvalidSameString(t *testing.T) {
	_, err := proof.NewInternalHasherForTest(testInproc1, testInproc1)
	assert.NotNil(t, err, "wrong new internal hasher")
}

func TestInternalHasherInitialiseWhenValidString(t *testing.T) {
	h, _ := proof.NewInternalHasherForTest(testInproc1, testInproc2)
	err := h.Initialise()

	assert.Equal(t, nil, err, "wrong initialise")
}

func TestNewInternalHasherInitialiseWhenInvalidString(t *testing.T) {
	h1, _ := proof.NewInternalHasherForTest(wrongEndpointRequestString, testInproc1)
	err := h1.Initialise()

	assert.NotNil(t, err, "wrong initialise")

	h2, _ := proof.NewInternalHasherForTest(testInproc1, wrongEndpointReplyString)
	err = h2.Initialise()

	assert.NotNil(t, err, "wrong initialise")
}

func TestInternalHasherStart(t *testing.T) {
	sender, _ := zmq.NewSocket(protocol)
	_ = sender.Bind(endpointRequestStr)
	receiver, _ := zmq.NewSocket(protocol)
	_ = receiver.Connect(endpointReplyStr)

	h, _ := proof.NewInternalHasherForTest(endpointRequestStr, endpointReplyStr)
	_ = h.Initialise()

	h.Start()

	job := "job"
	msg := &proof.PublishedItem{
		Job: job,
		Header: blockrecord.Header{
			Difficulty: difficulty.New(),
		},
		TxZero: []byte{},
		TxIds:  []merkle.Digest{},
	}
	sendData, _ := json.Marshal(msg)
	_, _ = sender.SendBytes(sendData, 0)

	receivedData, err := receiver.RecvMessageBytes(0)
	assert.Nil(t, err, "non nil error")

	var reply proof.SubmittedItem
	_ = json.Unmarshal([]byte(receivedData[0]), &reply)
	nonce := make([]byte, blockrecord.NonceSize)
	binary.LittleEndian.PutUint64(nonce, uint64(nonceStart))

	assert.Equal(t, job, reply.Job, "wrong job")
	assert.Equal(t, nonce, reply.Packed, "wrong packed")
}

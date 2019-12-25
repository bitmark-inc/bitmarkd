package proof

import (
	"encoding/binary"
	"encoding/json"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"

	zmq "github.com/pebbe/zmq4"
)

const (
	internalHasherProtocol = zmq.PAIR
)

type hashingRequest struct {
	Job    string
	Header blockrecord.Header
}

// InternalHasher - this dummy hasher is for test usage
type InternalHasher interface {
	Initialise() error
	Start()
}

type internalHasher struct {
	endpointRequestStr string
	endpointReplyStr   string
	requestSocket      *zmq.Socket // receive hash request
	replySocket        *zmq.Socket // send hash result reply
}

func (h *internalHasher) Initialise() error {
	requestSocket, err := zmq.NewSocket(internalHasherProtocol)
	if nil != err {
		return err
	}

	err = requestSocket.Connect(h.endpointRequestStr)
	if nil != err {
		return err
	}
	h.requestSocket = requestSocket

	replySocket, err := zmq.NewSocket(internalHasherProtocol)
	if nil != err {
		return err
	}

	err = replySocket.Bind(h.endpointReplyStr)
	if nil != err {
		return err
	}
	h.replySocket = replySocket

	return nil
}

func (h *internalHasher) Start() {
	go func() {
		for i := 1; ; i++ {
			msg, err := h.requestSocket.Recv(0)
			if nil != err {
				continue
			}

			var request hashingRequest
			_ = json.Unmarshal([]byte(msg), &request)
			nonce := make([]byte, blockrecord.NonceSize)
			binary.LittleEndian.PutUint64(nonce, uint64(i))

			reply := struct {
				Request string
				Job     string
				Packed  []byte
			}{
				Request: "block.nonce",
				Job:     request.Job,
				Packed:  nonce,
			}

			replyData, _ := json.Marshal(reply)

			_, err = h.replySocket.SendBytes(replyData, 0)
			if nil != err {
				continue
			}
		}
	}()
}

func NewInternalHasherForTest(request, reply string) (InternalHasher, error) {
	if request == reply {
		return nil, fault.WrongEndpointString
	}

	return &internalHasher{
		endpointRequestStr: request,
		endpointReplyStr:   reply,
	}, nil
}

package p2p

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// PackP2PMessage pack  chain fn and parameters into []byte
func PackP2PMessage(chain, fn string, parameters [][]byte) (packedP2PMessage []byte, err error) {
	data := [][]byte{[]byte(chain), []byte(fn)}
	if len(parameters) != 0 {
		data = append(data, parameters...)
	}
	packedP2PMessage, err = proto.Marshal(&P2PMessage{Data: data})
	return packedP2PMessage, err
}

// UnPackP2PMessage unpack p2pMessage to chain fn and parameters
func UnPackP2PMessage(packed []byte) (chain string, fn string, parameters [][]byte, err error) {
	unpacked := P2PMessage{}
	proto.Unmarshal(packed, &unpacked)
	if len(unpacked.Data) == 0 {
		return "", "", nil, fault.DataFieldEmpty
	}
	chain = string(unpacked.Data[0])
	fn = string(unpacked.Data[1])
	if fn == "B" {
		fmt.Println("\x1b[33m UnPackP2PMessage unpacked BLOCK data length=\x1b[0m", len(unpacked.Data))
	}

	if len(unpacked.Data) > 2 {
		parameters = unpacked.Data[2:]
	}
	return chain, fn, parameters, nil
}

// UnPackRegisterParameter Unpack register binary  data into object information
func UnPackRegisterParameter(parameters [][]byte) (peerType nodeType, id peerlib.ID, addrs []ma.Multiaddr, ts uint64, err error) {
	if len(parameters) < 4 {
		return peerType, id, addrs, ts, fault.ParametersLessThanExpect
	}

	if nodeType(parameters[0]) != ClientNode && nodeType(parameters[0]) != ServerNode {
		return peerType, id, addrs, ts, fault.InvalidNodeType
	}
	//nType := nodeType(parameters[0])
	id, err = peerlib.IDFromBytes(parameters[1])
	if err != nil {
		return "", id, addrs, ts, err
	}
	var announce Addrs
	err = proto.Unmarshal(parameters[2], &announce)
	if err != nil {
		return "", id, addrs, ts, err
	}
	if len(announce.Address) <= 0 {
		return "", id, addrs, ts, fault.NoAnnounceAddress
	}
	addrs = util.GetMultiAddrsFromBytes(announce.Address)
	ts = binary.BigEndian.Uint64(parameters[3])
	return peerType, id, addrs, ts, nil
}

// PackRegisterParameter pack node message into p2pMessage
func PackRegisterParameter(nodeType nodeType, id peerlib.ID, addrs []ma.Multiaddr, ts time.Time) ([][]byte, error) {
	typePacked := []byte(nodeType.String())
	idPacked, err := id.Marshal()
	if err != nil {
		return nil, err
	}
	addrsPackaed, err := proto.Marshal(&Addrs{Address: util.GetBytesFromMultiaddr(addrs)})
	if err != nil {
		return nil, err
	}
	tsPacked := make([]byte, 8)
	binary.BigEndian.PutUint64(tsPacked, uint64(ts.Unix()))
	packedData := [][]byte{typePacked, idPacked, addrsPackaed, tsPacked}
	return packedData, nil
}

// UnpackListenError unpacked ErrorMessage
func UnpackListenError(parameters [][]byte) error {
	return errors.New(string(parameters[0]))
}

// PackQueryDigestParameter pack node message into p2pMessage
func PackQueryDigestParameter(blockheight uint64) ([][]byte, error) {
	heightPacked := make([]byte, 8)
	binary.BigEndian.PutUint64(heightPacked, blockheight)
	packedData := [][]byte{[]byte(heightPacked)}
	return packedData, nil
}

// PackQueryBlockParameter pack node message into p2pMessage
func PackQueryBlockParameter(blockheight uint64) ([][]byte, error) {
	heightPacked := make([]byte, 8)
	binary.BigEndian.PutUint64(heightPacked, blockheight)
	packedData := [][]byte{heightPacked}
	return packedData, nil
}

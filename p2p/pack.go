package p2p

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/gogo/protobuf/proto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

//PackP2PMessage unpack p2pMessage to chain fn and parameters
func PackP2PMessage(chain, fn string, parameters [][]byte) (packedP2PMessage []byte, err error) {
	data := [][]byte{[]byte(chain), []byte(fn)}
	if len(parameters) != 0 {
		data = append(data, parameters...)
	}
	packedP2PMessage, err = proto.Marshal(&P2PMessage{Data: data})
	return packedP2PMessage, err
}

//UnPackP2PMessage unpack p2pMessage to chain fn and parameters
func UnPackP2PMessage(packed []byte) (chain string, fn string, parameters [][]byte, err error) {
	unpacked := P2PMessage{}
	proto.Unmarshal(packed, &unpacked)
	if len(unpacked.Data) == 0 {
		return "", "", nil, errors.New("No Data")
	}
	chain = string(unpacked.Data[0])
	fn = string(unpacked.Data[1])
	parameters = unpacked.Data[2:]
	return chain, fn, parameters, nil
}

//UnPackRegisterData Unpack register binary  data into objectr information
func UnPackRegisterData(parameters [][]byte) (nodeType string, id peerlib.ID, addrs []ma.Multiaddr, ts uint64, err error) {
	if len(parameters) < 4 {
		return nodeType, id, addrs, ts, errors.New("Invalid data")
	}
	nType := string(parameters[0])
	id, err = peerlib.IDFromBytes(parameters[1])
	if err != nil {
		return "", id, addrs, ts, err
	}
	var listeners Addrs
	err = proto.Unmarshal(parameters[2], &listeners)
	if err != nil {
		return "", id, addrs, ts, err
	}
	addrs = util.GetMultiAddrsFromBytes(listeners.Address)
	ts = binary.BigEndian.Uint64(parameters[3])
	return nType, id, addrs, ts, nil
}

//PackRegisterData pack node message into p2pMessage
func PackRegisterData(chain, fn string, nodeType string, id peerlib.ID, addrs []ma.Multiaddr, ts time.Time) ([][]byte, error) {
	typePacked := []byte(nodeType)
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
	packedData := [][]byte{[]byte(chain), []byte(fn), typePacked, idPacked, addrsPackaed, tsPacked}
	return packedData, nil
}

//UnpackListenError unpacked ErrorMessage
func UnpackListenError(parameters [][]byte) (error, error) {
	return errors.New(string(parameters[0])), nil
}

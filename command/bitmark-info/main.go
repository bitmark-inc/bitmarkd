package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"

	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
)

type RPCEmptyArguments struct{}

type ConnClient struct {
	Clients []string `json:"clients"`
}

type RPCClient struct {
	Client *rpc.Client
}

// GetNodeInfo will get the node info of a node from bitmark rpc
func (r *RPCClient) GetNodeInfo() (json.RawMessage, error) {
	args := RPCEmptyArguments{}
	var reply json.RawMessage
	err := r.Client.Call("Node.Info", &args, &reply)
	return reply, err
}

// GetSubscribers will get all its subscribers of a node from bitmark rpc
func (r *RPCClient) GetSubscribers() (ConnClient, error) {
	args := RPCEmptyArguments{}
	var reply ConnClient
	err := r.Client.Call("Node.Subscribers", &args, &reply)
	return reply, err
}

// GetConnectors will get all its connectors of a node from bitmark rpc
func (r *RPCClient) GetConnectors() (ConnClient, error) {
	args := RPCEmptyArguments{}
	var reply ConnClient
	err := r.Client.Call("Node.Connectors", &args, &reply)
	return reply, err
}

func (r *RPCClient) GetAllInfo() (reply map[string]interface{}, err error) {
	node, err := r.GetNodeInfo()
	if err != nil {
		return
	}
	sbsc, err := r.GetSubscribers()
	if err != nil {
		return
	}
	conn, err := r.GetConnectors()
	if err != nil {
		return
	}
	reply = map[string]interface{}{
		"node": node,
		"sbsc": sbsc,
		"conn": conn,
	}
	return
}

func main() {
	defer exitwithstatus.Handler()

	flags := []getoptions.Option{
		{Long: "help", HasArg: getoptions.NO_ARGUMENT, Short: 'h'},
		{Long: "info-type", HasArg: getoptions.OPTIONAL_ARGUMENT, Short: 'i'},
	}

	program, options, arguments, err := getoptions.GetOS(flags)
	if err != nil {
		exitwithstatus.Message("option parse error: %s", err)
	}

	if len(options["help"]) > 0 {
		exitwithstatus.Message("usage: %s [--help] [--info-type=TYPE] [host:port]", program)
	}

	// set the default info type
	infoType := []string{"node"}

	if len(options["info-type"]) != 0 {
		infoType = options["info-type"]
	}

	var hostPort string
	if len(arguments) != 0 {
		hostPort = arguments[0]
	}

	// establish rpc connection over tls
	conn, err := tls.Dial("tcp", hostPort, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		exitwithstatus.Message("dial error: %s", err)
	}
	defer conn.Close()
	client := jsonrpc.NewClient(conn)

	r := RPCClient{client}
	reply := map[string]interface{}{}

	for _, t := range infoType {
		var v interface{}
		switch t {
		case "all":
			reply, err = r.GetAllInfo()
			break
		case "node":
			v, err = r.GetNodeInfo()
			reply["node"] = v
		case "sbsc":
			v, err = r.GetSubscribers()
			reply["sbsc"] = v
		case "conn":
			v, err = r.GetConnectors()
			reply["conn"] = v
		default:
			err = fmt.Errorf("incorrect info type provided: %s", infoType)
		}
	}

	if err != nil {
		exitwithstatus.Message("rpc error: %s", err)
	}

	b, err := json.Marshal(reply)
	if err != nil {
		exitwithstatus.Message("incorrect json marshal: %s", err)
	}

	fmt.Printf("%s", b)
	os.Exit(0)
}

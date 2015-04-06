// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
	flags "github.com/jessevdk/go-flags"
	"os"
	"path/filepath"
	"strings"
)

// basic defaults
const (
	defaultConfigFileName = "bitmarkd.conf"
	defaultDataDirname    = "data"

	defaultLogDirectoryName = "log"
	defaultLogFileName      = "bitmarkd.log"
	defaultLogRotateCount   = 10          //  <logfile>.0 ... <logfile>.<N-1>
	defaultLogSize          = 1024 * 1024 // rotate when <logfile> exceeds this size

	defaultRPCClients = 10
	defaultPeers      = 125
	defaultMines      = 125
	defaultRemotes    = 25
	//defaultBanDuration = time.Hour * 24

	defaultBlockCacheSize       = 100
	defaultTransactionCacheSize = 100
)

// path expanded or calculated defaults
var (
	appHomeDirectory        = util.AppDataDir("bitmarkd", false)
	defaultPidFile          = filepath.Join(appHomeDirectory, "bitmarkd.pid")
	defaultConfigFile       = filepath.Join(appHomeDirectory, defaultConfigFileName)
	defaultDataDirectory    = filepath.Join(appHomeDirectory, defaultDataDirname)
	defaultPublicKeyFile    = filepath.Join(appHomeDirectory, "bitmarkd.private")
	defaultPrivateKeyFile   = filepath.Join(appHomeDirectory, "bitmarkd.public")
	defaultKeyFile          = filepath.Join(appHomeDirectory, "bitmarkd.key")
	defaultCertificateFile  = filepath.Join(appHomeDirectory, "bitmarkd.crt")
	defaultLogDirectory     = filepath.Join(appHomeDirectory, defaultLogDirectoryName)
	defaultLogFile          = filepath.Join(defaultLogDirectory, defaultLogFileName)
	defaultTestDatabaseFile = filepath.Join(defaultDataDirectory, "testing.leveldb")
	defaultLiveDatabaseFile = filepath.Join(defaultDataDirectory, "bitmark.leveldb")

	defaultDebug = map[string]string{
		"main":            "info",
		"config":          "info",
		logger.DefaultTag: "critical",
	}
)

type debugMap map[string]string

// type to hold remote
type Remote struct {
	PublicKey string
	Address   string
}

// all of the possible options
type CommandOptions struct {

	// basic options
	ConfigFile string   `short:"c" long:"config" description:"Path to configuration file (command arguments take precedence or supplement file values)"`
	Version    bool     `short:"V" long:"version" description:"Display version information and exit"`
	Quiet      bool     `short:"q" long:"quiet" description:"Suppress messages to stdout/stderr"`
	Debug      debugMap `short:"D" long:"debug" description:"Set debugging level as module:level where: module=(default,rpc,net,block) and level=(debug,info,warning,error,critcal)"`
	Verbose    bool     `short:"v" long:"verbose" description:"More output independant of log levels"`

	// PID File
	PidFile string `short:"p" long:"PidFile" description:"PID file name"`

	// test mode or production mode
	TestMode bool `long:"TestMode" description:"Set true to enable test mode"`

	// server identification in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
	PublicKey  string `long:"PublicKey" description:"File containing Z85 encoded Curve Public Key"`
	PrivateKey string `long:"PrivateKey" description:"File containing Z85 encoded Curve Private Key"`

	// Peers (incoming from other bitmarkd)
	Peers         int      `long:"Peers" description:"Limit the number of peers that can connect"`
	PeerListeners []string `long:"PeerListen" description:"Add an IP:port to listen for peer connections"`
	PeerAnnounce  []string `long:"PeerAnnounce" description:"Publish a peer IP:port to network (Public/Firewall Forwarded/NAT)"`

	// Connect (outgoing to other bitmarkd)
	Remotes       int      `long:"Remotes" description:"Limit the number outgoing peer connections"`
	RemoteConnect []Remote `long:"RemoteConnect" description:"Add a 'Z85-public-key',IP:port for a connection to a remote peer"`

	// RPC (incoming from clients)
	RPCClients     int      `long:"RpcClients" description:"Limit the number of RPC clients that can connect"`
	RPCListeners   []string `long:"RpcListen" description:"Add an IP:port to listen for RPC connections"`
	RPCCertificate string   `long:"RpcCert" description:"File containing the certificate"`
	RPCKey         string   `long:"RpcKey" description:"File containing the private key"`
	RPCAnnounce    []string `long:"RpcAnnounce" description:"Publish an RPC IP:port to network (Public/Firewall Forwarded/NAT)"`

	// Mines (incoming from stratum+ssl miners)
	Mines           int      `long:"Mines" description:"Limit the number of miners that can connect"`
	MineListeners   []string `long:"MineListen" description:"Add an IP:port to listen for miner connections"`
	MineCertificate string   `long:"MineCert" description:"File containing the certificate"`
	MineKey         string   `long:"MineKey" description:"File containing the private key"`
	//MineAnnounce    []string `long:"MineAnnounce" description:"Publish a mine IP:port to network (Public/Firewall Forwarded/NAT)"`

	// storage
	DatabaseFile         string `long:"database" description:"LevelDB file for all data storage"`
	BlockCacheSize       int    `long:"BlockCache" description:"Memory pool size for caching blocks"`
	TransactionCacheSize int    `long:"TransactionCache" description:"Memory pool size for caching transactions"`

	// logging
	LogFile        string `long:"LogFile" description:"Log file base name"`
	LogSize        int    `long:"LogSize" description:"Maimum size of file before rotating"`
	LogRotateCount int    `long:"LogRotateCount" description:"Maximum number of rotations to keep"`

	// Bitcoin access
	BitcoinUsername string  `long:"BitcoinUsername" description:"Username for Bitcoin RPC access"`
	BitcoinPassword string  `long:"BitcoinPassword" description:"Password for Bitcoin RPC access"`
	BitcoinURL      string  `long:"BitcoinURL" description:"URL for Bitcoin RPC access"`
	BitcoinAddress  string  `long:"BitcoinAddress" description:"Bitcoin Address for miner"`
	BitcoinFee      float64 `long:"BitcoinFee" description:"Bitcoin fee per transaction"`
	BitcoinStart    uint64  `long:"BitcoinStart" description:"Bitcoin start block for transaction dectection"`

	Args struct {
		Command   string   `name:"command" description:"Command: use 'help' to show list of commands"`
		Arguments []string `name:"args" description:"A optional arguments for command"`
	} `positional-args:"yes"`
}

func ParseOptions() CommandOptions {

	options := CommandOptions{
		ConfigFile:           defaultConfigFile,
		Debug:                defaultDebug,
		PidFile:              defaultPidFile,
		TestMode:             false,
		PublicKey:            defaultPublicKeyFile,
		PrivateKey:           defaultPrivateKeyFile,
		RPCClients:           defaultRPCClients,
		RPCCertificate:       defaultCertificateFile,
		RPCKey:               defaultKeyFile,
		Peers:                defaultPeers,
		Remotes:              defaultRemotes,
		Mines:                defaultMines,
		MineCertificate:      defaultCertificateFile,
		MineKey:              defaultKeyFile,
		DatabaseFile:         defaultLiveDatabaseFile,
		BlockCacheSize:       defaultBlockCacheSize,
		TransactionCacheSize: defaultTransactionCacheSize,
		LogFile:              defaultLogFile,
		LogSize:              defaultLogSize,
		LogRotateCount:       defaultLogRotateCount,
	}

	temporaryOptions := options
	temporaryParser := flags.NewParser(&temporaryOptions, flags.None)

	temporaryParser.Parse() // only want to get config file at this point

	// start the real parsing
	parser := flags.NewParser(&options, flags.Default)

	if cfg := temporaryOptions.ConfigFile; cfg != "" {
		// add to or override defaults using configuration file
		err := flags.NewIniParser(parser).ParseFile(cfg)
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				exitwithstatus.Usage("Error: %v parsing configuration from: %s\n", err, cfg)
			}
		}
	}

	// add to or override defaults / configuration file from command arguments
	_, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			exitwithstatus.Usage("Error: %v\n", err)
		}
		exitwithstatus.Exit(1)
	}

	// if test mode and the database file was not specified
	// switch to test file
	if options.TestMode && options.DatabaseFile == defaultLiveDatabaseFile {
		options.DatabaseFile = defaultTestDatabaseFile
	}

	// create the directories if they do not already exist
	for _, d := range []string{appHomeDirectory, defaultDataDirectory, defaultLogDirectory} {
		err = os.MkdirAll(d, 0700)
		if err != nil {
			exitwithstatus.Usage("Directory: %s creation failed with error: %v\n", d, err)
		}
	}

	// done
	return options
}

// resolve a filename
//
// if starts with '/' then it is global
// if not present in current directory try with appHomeDirectory as a prefix
func ResolveFileName(name string) (string, bool) {

	_, err := os.Stat(name)
	if nil == err {
		return name, true
	}

	if filepath.IsAbs(name) {
		return name, false
	}

	path := filepath.Join(appHomeDirectory, name)
	_, err = os.Stat(path)

	return path, nil == err
}

// parse remote
// expect:
//   'z85 encoded publc key',127.0.0.1:1234
//   'z85 encoded publc key',[::1]:1234
func (r *Remote) UnmarshalFlag(value string) error {

	parts := strings.Split(value, "'")
	if 3 != len(parts) || "" != parts[0] || 0 == len(parts[1]) {
		return fault.ErrInvalidRemote
	}

	parts2 := strings.Split(strings.Trim(parts[2], " "), ",")
	if 2 != len(parts2) || 0 != len(parts2[0]) || 0 == len(parts2[1]) {
		return fault.ErrInvalidRemote
	}

	address, err := util.CanonicalIPandPort(parts2[1])
	if nil != err {
		return err
	}

	r.PublicKey = parts[1]
	r.Address = address
	return nil
}

func (r Remote) MarshalFlag() (string, error) {
	return fmt.Sprintf("'%s',%s", r.PublicKey, r.Address), nil
}

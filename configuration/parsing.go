// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"errors"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/chain"
	//"github.com/bitmark-inc/bitmarkd/fault"
	//"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
	"os"
	"path/filepath"
	"strings"
)

// basic defaults (directories and files are relative to the "DataDirectory" from Configuration file)
const (
	defaultDataDirectory = "" // this will error; use "." for the same directory as the config file
	defaultPidFile       = "bitmarkd.pid"

	defaultPublicKeyFile   = "bitmarkd.private"
	defaultPrivateKeyFile  = "bitmarkd.public"
	defaultKeyFile         = "bitmarkd.key"
	defaultCertificateFile = "bitmarkd.crt"

	defaultLevelDBDirectory = "data"
	defaultBitmarkDatabase  = chain.Bitmark + ".leveldb"
	defaultTestingDatabase  = chain.Testing + ".leveldb"
	defaultLocalDatabase    = chain.Local + ".leveldb"

	defaultLogDirectory = "log"
	defaultLogFile      = "bitmarkd.log"
	defaultLogCount     = 10          //  number of log files retained
	defaultLogSize      = 1024 * 1024 // rotate when <logfile> exceeds this size

	defaultRPCClients = 10
	defaultPeers      = 125
	defaultMines      = 125
)

// to hold log levels
type LoglevelMap map[string]string

// path expanded or calculated defaults
var (
	defaultLogLevels = LoglevelMap{
		"main":            "info",
		"config":          "info",
		logger.DefaultTag: "critical",
	}
)

type RPCType struct {
	MaximumConnections int      `libucl:"maximum_connections"`
	Listen             []string `libucl:"listen"`
	Certificate        string   `libucl:"certificate"`
	PrivateKey         string   `libucl:"private_key"`
	Announce           []string `libucl:"announce"`
}

type Connection struct {
	PublicKey string `libucl:"public_key"`
	Address   string `libucl:"address"`
}

// server identification in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
type PeerType struct {
	MaximumConnections int          `libucl:"maximum_connections"`
	Listen             []string     `libucl:"listen"`
	Connect            []Connection `libucl:"connect"`
	PrivateKey         string       `libucl:"private_key"`
	PublicKey          string       `libucl:"public_key"`
	Announce           []string     `libucl:"announce"`
}

type LoggerType struct {
	Directory string            `libucl:"directory"`
	File      string            `libucl:"file"`
	Size      int               `libucl:"size"`
	Count     int               `libucl:"count"`
	Levels    map[string]string `libucl:"levels"`
}

type BitcoinAccess struct {
	Username string `libucl:"Username"`
	Password string `libucl:"Password"`
	URL      string `libucl:"URL"`
	Address  string `libucl:"Address"`
	Fee      string `libucl:"Fee"`
	Start    uint64 `libucl:"Start"`
}

type DatabaseType struct {
	Directory string `libucl:"directory"`
	Name      string `libucl:"name"`
}

type Configuration struct {
	DataDirectory string       `libucl:"data_directory"`
	PidFile       string       `libucl:"pidfile"`
	Chain         string       `libucl:"chain"`
	Database      DatabaseType `libucl:"database"`

	ClientRPC RPCType       `libucl:"client_rpc"`
	Peering   PeerType      `libucl:"peering"`
	Mining    RPCType       `libucl:"mining"`
	Bitcoin   BitcoinAccess `libucl:"bitcoin"`
	Logging   LoggerType    `libucl:"logging"`
}

// // all of the possible options
// type CommandOptions struct {

// 	// basic options
// 	ConfigFile string   `short:"c" long:"config" description:"Path to configuration file (command arguments take precedence or supplement file values)"`
// 	Version    bool     `short:"V" long:"version" description:"Display version information and exit"`
// 	Quiet      bool     `short:"q" long:"quiet" description:"Suppress messages to stdout/stderr"`
// 	Debug      debugMap `short:"D" long:"debug" description:"Set debugging level as module:level where: module=(default,rpc,net,block) and level=(debug,info,warning,error,critcal)"`
// 	Verbose    bool     `short:"v" long:"verbose" description:"More output independant of log levels"`

// 	// PID File
// 	PidFile string `short:"p" long:"PidFile" description:"PID file name"`

// 	// Profile File
// 	ProfileFile string `long:"ProfileFile" description:"Profile file name (blank turns off profiling)"`

// 	// select the chain to connect to
// 	Chain string `long:"Chain" description:"Set to mode string: [bitmark, testing, local]"`

// 	// server identification in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
// 	PublicKey  string `long:"PublicKey" description:"File containing Z85 encoded Curve Public Key"`
// 	PrivateKey string `long:"PrivateKey" description:"File containing Z85 encoded Curve Private Key"`

// 	// Peers (incoming from other bitmarkd)
// 	Peers         int      `long:"Peers" description:"Limit the number of peers that can connect"`
// 	PeerListeners []string `long:"PeerListen" description:"Add an IP:port to listen for peer connections"`
// 	PeerAnnounce  []string `long:"PeerAnnounce" description:"Publish a peer IP:port to network (Public/Firewall Forwarded/NAT)"`

// 	// Connect (outgoing to other bitmarkd)
// 	Remotes       int      `long:"Remotes" description:"Limit the number outgoing peer connections"`
// 	RemoteConnect []Remote `long:"RemoteConnect" description:"Add a 'Z85-public-key',IP:port for a connection to a remote peer"`

// 	// RPC (incoming from clients)
// 	RPCClients     int      `long:"RpcClients" description:"Limit the number of RPC clients that can connect"`
// 	RPCListeners   []string `long:"RpcListen" description:"Add an IP:port to listen for RPC connections"`
// 	RPCCertificate string   `long:"RpcCert" description:"File containing the certificate"`
// 	RPCKey         string   `long:"RpcKey" description:"File containing the private key"`
// 	RPCAnnounce    []string `long:"RpcAnnounce" description:"Publish an RPC IP:port to network (Public/Firewall Forwarded/NAT)"`

// 	// Mines (incoming from stratum+ssl miners)
// 	Mines           int      `long:"Mines" description:"Limit the number of miners that can connect"`
// 	MineListeners   []string `long:"MineListen" description:"Add an IP:port to listen for miner connections"`
// 	MineCertificate string   `long:"MineCert" description:"File containing the certificate"`
// 	MineKey         string   `long:"MineKey" description:"File containing the private key"`
// 	//MineAnnounce    []string `long:"MineAnnounce" description:"Publish a mine IP:port to network (Public/Firewall Forwarded/NAT)"`

// 	// storage
// 	Database string `long:"database" description:"LevelDB file for all data storage"`

// 	// logging
// 	LogFile        string `long:"LogFile" description:"Log file base name"`
// 	LogSize        int    `long:"LogSize" description:"Maximum size of file before rotating"`
// 	LogRotateCount int    `long:"LogRotateCount" description:"Maximum number of rotations to keep"`

// 	// Bitcoin access
// 	BitcoinUsername string `long:"BitcoinUsername" description:"Username for Bitcoin RPC access"`
// 	BitcoinPassword string `long:"BitcoinPassword" description:"Password for Bitcoin RPC access"`
// 	BitcoinURL      string `long:"BitcoinURL" description:"URL for Bitcoin RPC access"`
// 	BitcoinAddress  string `long:"BitcoinAddress" description:"Bitcoin Address for miner"`
// 	BitcoinFee      string `long:"BitcoinFee" description:"Bitcoin fee per transaction in BTC (e.g. 0.0002)"`
// 	BitcoinStart    uint64 `long:"BitcoinStart" description:"Bitcoin start block for transaction dectection"`

// 	Args struct {
// 		Command   string   `name:"command" description:"Command: use 'help' to show list of commands"`
// 		Arguments []string `name:"args" description:"A optional arguments for command"`
// 	} `positional-args:"yes"`
// }

// will read decode and verify the configuration
func GetConfiguration(configurationFileName string) (*Configuration, error) {

	configurationFileName, err := filepath.Abs(filepath.Clean(configurationFileName))
	if nil != err {
		return nil, err
	}

	// absolute path to the main directory
	dataDirectory, _ := filepath.Split(configurationFileName)

	options := &Configuration{

		DataDirectory: defaultDataDirectory,
		PidFile:       defaultPidFile,
		Chain:         chain.Bitmark,

		Database: DatabaseType{
			Directory: defaultLevelDBDirectory,
			Name:      defaultBitmarkDatabase,
		},

		ClientRPC: RPCType{
			MaximumConnections: defaultRPCClients,
			Certificate:        defaultCertificateFile,
			PrivateKey:         defaultKeyFile,
		},

		Peering: PeerType{
			MaximumConnections: defaultPeers,
		},

		Mining: RPCType{
			MaximumConnections: defaultMines,
			Certificate:        defaultCertificateFile,
			PrivateKey:         defaultKeyFile,
		},

		Logging: LoggerType{
			Directory: defaultLogDirectory,
			File:      defaultLogFile,
			Size:      defaultLogSize,
			Count:     defaultLogCount,
			Levels:    defaultLogLevels,
		},
	}

	if err := readConfigurationFile(configurationFileName, options); err != nil {
		return nil, err
	}

	// if any test mode and the database file was not specified
	// switch to appropriate default.  Abort if then chain name is
	// not recognised.
	options.Chain = strings.ToLower(options.Chain)
	if !chain.Valid(options.Chain) {
		return nil, errors.New(fmt.Sprintf("Chain: %q is not supported", options.Chain))
	}

	// if database was not changed from default
	if options.Database.Name == defaultBitmarkDatabase {
		switch options.Chain {
		case chain.Bitmark:
			// already correct default
		case chain.Testing:
			options.Database.Name = defaultTestingDatabase
		case chain.Local:
			options.Database.Name = defaultLocalDatabase
		default:
			return nil, errors.New(fmt.Sprintf("Chain: %s no default databse setting", options.Chain))
		}
	}

	// ensure absolute data directory
	if "" == options.DataDirectory || "~" == options.DataDirectory {
		return nil, errors.New(fmt.Sprintf("Path: %q is not a valid directory", options.DataDirectory))
	} else if "." == options.DataDirectory {
		options.DataDirectory = dataDirectory // same directory as the configuration file
	} else {
		options.DataDirectory = filepath.Clean(options.DataDirectory)
	}

	// this directory must exist - i.e. must be created prior to running
	if fileInfo, err := os.Stat(options.DataDirectory); nil != err {
		return nil, err
	} else if !fileInfo.IsDir() {
		return nil, errors.New(fmt.Sprintf("Path: %q is not a directory", options.DataDirectory))
	}

	// force all relevant items to be absolute paths
	// if not, assign them to the dsts directory
	mustBeAbsolute := []*string{
		&options.PidFile,
		&options.Database.Directory,
		&options.ClientRPC.Certificate,
		&options.ClientRPC.PrivateKey,
		&options.Peering.PublicKey,
		&options.Peering.PrivateKey,
		&options.Mining.Certificate,
		&options.Mining.PrivateKey,
		&options.Logging.Directory,
	}
	for _, f := range mustBeAbsolute {
		*f = ensureAbsolute(options.DataDirectory, *f)
	}

	// fail if any of these are not simple file names i.e. must not contain path seperator
	// then add the correct directory prefix, file item is first and corresponding directory is second
	mustNotBePaths := [][2]*string{
		{&options.Database.Name, &options.Database.Directory},
		{&options.Logging.File, &options.Logging.Directory},
	}
	for _, f := range mustNotBePaths {
		switch filepath.Dir(*f[0]) {
		case "", ".":
			*f[0] = ensureAbsolute(*f[1], *f[0])
		default:
			return nil, errors.New(fmt.Sprintf("Files: %q is not plain name", *f[0]))
		}
	}

	// make absolute and create directories if they do not already exist
	for _, d := range []*string{&options.Database.Directory, &options.Logging.Directory} {
		*d = ensureAbsolute(options.DataDirectory, *d)
		if err := os.MkdirAll(*d, 0700); nil != err {
			return nil, err
		}
	}

	// done
	return options, nil
}

// ensure the path is absolute
func ensureAbsolute(directory string, filePath string) string {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(directory, filePath)
	}
	return filepath.Clean(filePath)
}

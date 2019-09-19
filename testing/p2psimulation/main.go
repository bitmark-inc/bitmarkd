package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/p2p"

	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/logger"
)

// GlobalMockConfiguration Markup global configuration
type GlobalMockConfiguration struct {
	DataDirectory string               `gluamapper:"data_directory" json:"data_directory"`
	Peering       p2p.Configuration    `gluamapper:"peering" json:"peering"`
	Logging       logger.Configuration `gluamapper:"logging" json:"logging"`
	PidFile       string               `gluamapper:"pidfile" json:"pidfile"`
	Chain         string               `gluamapper:"chain" json:"chain"`
	Nodes         string               `gluamapper:"nodes" json:"nodes"`
}

func main() {
	var globalConf GlobalMockConfiguration
	path := filepath.Join(os.Getenv("PWD"), "p2p.conf")
	flag.StringVar(&path, "conf", "", "Specify configuration file")
	flag.Parse()
	fmt.Println("Config File=", path)
	err := configuration.ParseConfigurationFile(path, &globalConf)
	if err != nil {
		fmt.Println("Error:", err)
	}
	// start logging
	if err = logger.Initialise(globalConf.Logging); nil != err {
		panic(err)
	}
	defer logger.Finalise()
	err = announce.Initialise(getDomainName(globalConf), getPeerFile(globalConf.Chain))
	if nil != err {
		panic(fmt.Sprintf("peer initialise error: %s", err))
	}
	defer announce.Finalise()
	err = p2p.Initialise(&globalConf.Peering)
	if nil != err {
		panic(fmt.Sprintf("peer initialise error: %s", err))
	}
	defer p2p.Finalise()
	for {
		time.Sleep(10 * time.Second)
	}
}

func getPeerFile(chain string) string {
	curPath := os.Getenv("PWD")
	switch chain {
	case "bitmark":
		return path.Join(curPath, "peers-bitmark")
	case "testing":
		return path.Join(curPath, "peers-testing")
	case "local":
		return path.Join(curPath, "peers-local")
	default:
		panic("invalid chain name")
	}
}

func getDomainName(masterConfiguration GlobalMockConfiguration) string {
	nodesDomain := ""
	switch masterConfiguration.Nodes {
	case "":
		panic("nodes cannot be blank choose from: none, chain or sub.domain.tld")
	case "none":
		panic("nodes cannot be blank choose from: none, chain or sub.domain.tld")
	case "chain":
		switch cn := masterConfiguration.Chain; cn { // ***** FIX THIS: is there a better way?
		case "local":
			nodesDomain = masterConfiguration.Nodes
		case "testing":
			nodesDomain = "nodes.test.bitmark.com"
		case "bitmark":
			nodesDomain = "nodes.live.bitmark.com"
		default:
			panic(fmt.Sprintf("unexpected chain name: %q", cn))
		}
	default:
		// domain names are complex to validate so just rely on
		// trying to fetch the TXT records for validation
		nodesDomain = masterConfiguration.Nodes // just assume it is a domain name
	}
	fmt.Println(nodesDomain)
	return nodesDomain
}

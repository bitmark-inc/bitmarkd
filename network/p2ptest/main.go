package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
)

// GlobalMockConfiguration Markup global configuration
type GlobalMockConfiguration struct {
	DataDirectory string               `gluamapper:"data_directory" json:"data_directory"`
	Peering       Configuration        `gluamapper:"peering" json:"peering"`
	Logging       logger.Configuration `gluamapper:"logging" json:"logging"`
}

func main() {
	var globalConf GlobalMockConfiguration
	path := filepath.Join(os.Getenv("PWD"), "config.conf")
	err := configuration.ParseConfigurationFile(path, &globalConf)
	if err != nil {
		fmt.Println("Error:", err)
	}
	// start logging
	if err = logger.Initialise(globalConf.Logging); nil != err {
		panic(err)
	}
	defer logger.Finalise()
	err = Initialise(&globalConf.Peering, version)

	if nil != err {
		fmt.Println("peer initialise error: ", err.Error())
		exitwithstatus.Message("peer initialise error: %s", err)
	}

	defer Finalise()
}

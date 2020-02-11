package announce

import (
	"fmt"
	"os"
	"testing"

	"github.com/bitmark-inc/logger"
)

func TestMain(m *testing.M) {
	curPath := os.Getenv("PWD")
	logLevel := make(map[string]string)
	logLevel["DEFAULT"] = "info"
	var logConfig = logger.Configuration{
		Directory: curPath,
		File:      "routing_test.log",
		Size:      1048576,
		Count:     20,
		Console:   true,
		Levels:    logLevel,
	}
	if err := logger.Initialise(logConfig); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	globalData.log = logger.New("nodes")
	os.Exit(m.Run())
}

package main

import (
	"os"
	"path"
	"strconv"
	"testing"

	logger "github.com/bitmark-inc/logger"
)

const (
	logDirectory     = "log"
	logFileName      = "test.log"
	logSizeOfFiles   = 30000
	logNumberOfFiles = 10
	defaultCPUUsage  = 30
)

var logging logger.Configuration

var testLevelMap = map[string]string{
	"main": "debug",
	"aux":  "warn",
}

func setupReader(t *testing.T) *ConfigReaderData {
	removeTestFiles()
	setupLogger(t)
	reader := &ConfigReaderData{
		proofer: &FakeProofer{},
	}
	reader.Initialise("test")
	_ = reader.SetLog(logger.New("test"))

	return reader
}

func teardown() {
	logger.Finalise()
	removeTestFiles()
}

func removeTestFiles() {
	logFilePath := path.Join(logDirectory, logFileName)
	os.Remove(logFilePath)
	for i := 0; i <= logNumberOfFiles; i += 1 {
		os.Remove(logFilePath + "." + strconv.Itoa(i))
	}
	os.Remove(logDirectory)
}

func loggerConfiguration() logger.Configuration {
	return logger.Configuration{
		Directory: logDirectory,
		File:      logFileName,
		Size:      logSizeOfFiles,
		Count:     logNumberOfFiles,
		Levels:    testLevelMap,
	}
}

func setupLogger(t *testing.T) {
	_ = os.Mkdir(logDirectory, 0770)
	logging = loggerConfiguration()
	_ = logger.Initialise(logging)
}

func mockConfiguration(maxCpuUsage int) *Configuration {
	return &Configuration{
		DataDirectory: "test",
		PidFile:       "test",
		Chain:         "test",
		MaxCPUUsage:   maxCpuUsage,
		Peering:       PeerType{},
		Logging:       logging,
	}
}

func TestGetConfig(t *testing.T) {
	reader := setupReader(t)
	defer teardown()

	oldConfig, _, _ := reader.GetConfig()
	if nil != oldConfig {
		t.Errorf("Cannot get configuration")
	}

	newConfig := mockConfiguration(defaultCPUUsage)
	reader.update(newConfig)
	currentConfig, _, _ := reader.GetConfig()
	if currentConfig != newConfig {
		t.Errorf("Get wrong config")
	}
}

func TestUpdateConfiguraion(t *testing.T) {
	reader := setupReader(t)
	defer teardown()

	newConfiguration := mockConfiguration(defaultCPUUsage)
	reader.update(newConfiguration)
	currentConfig, _, _ := reader.GetConfig()
	if currentConfig != newConfiguration {
		t.Errorf("current configuration %v different from expected %v", currentConfig, newConfiguration)
	}
}

func TestUpdateThreadCount(t *testing.T) {
	reader := setupReader(t)
	defer teardown()

	totalCPU := uint32(10)
	newConfiguration := mockConfiguration(defaultCPUUsage)
	reader.updateCpuCount(totalCPU)
	reader.update(newConfiguration)
	threadCount := reader.threadCount
	if threadCount != reader.OptimalThreadCount() {
		t.Errorf("update threadcount fail, expected %d differs %d",
			threadCount, defaultCPUUsage*totalCPU)
	}
}

func TestOptimalThreadCount(t *testing.T) {
	reader := setupReader(t)
	defer teardown()

	expected := []struct {
		totalCPU uint32
		usage    int
		thread   uint32
	}{
		{4, 25, 1},    // 25% of 4 cpu is 1 thread
		{8, 0, 1},     // minimum 1 thread
		{12, 200, 12}, // maximum to # of cpu core
		{16, 30, 4},   // round to integer
	}

	for _, s := range expected {
		mockConfig := mockConfiguration(s.usage)
		reader.update(mockConfig)
		reader.updateCpuCount(s.totalCPU)
		calculatedThreadCount := reader.OptimalThreadCount()
		if s.thread != calculatedThreadCount {
			t.Errorf("expected thread count %d different from calculated %d",
				s.thread, calculatedThreadCount)
		}
	}
}

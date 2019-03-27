// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/bitmark-inc/logger"
)

type ConfigReader interface {
	Initialise(string)
	OptimalThreadCount() uint32
	SetCalendar(JobCalendar)
	Refresh() error
	GetConfig() (*Configuration, string, error)
	SetLog(*logger.L) error
	Start()
	FirstTimeRun()
	SetProofer(Proofer)
}

const (
	oneMinute          = time.Duration(1) * time.Minute
	minThreadCount     = 1
	ReaderLoggerPrefix = "config-reader"
)

var (
	totalCPUCount = uint32(runtime.NumCPU())
)

type ConfigReaderData struct {
	fileName             string
	refreshByMinute      time.Duration
	log                  *logger.L
	currentConfiguration *Configuration
	initialized          bool
	threadCount          uint32
	calendar             JobCalendar
	proofer              Proofer
	watcherChannel       WatcherChannel
}

func newConfigReader(ch WatcherChannel) ConfigReader {
	return &ConfigReaderData{
		log:                  nil,
		currentConfiguration: nil,
		threadCount:          1,
		initialized:          false,
		refreshByMinute:      oneMinute,
		watcherChannel:       ch,
	}
}

// configuration needs read first to know logger file location
func (c *ConfigReaderData) Initialise(fileName string) {
	c.fileName = fileName
}

func (c *ConfigReaderData) SetCalendar(calendar JobCalendar) {
	c.calendar = calendar
}

func (c *ConfigReaderData) SetProofer(proofer Proofer) {
	c.proofer = proofer
}

func (c *ConfigReaderData) FirstTimeRun() {
	err := c.Refresh()
	if nil != err {
		return
	}
	c.notify()
}

func (c *ConfigReaderData) Start() {
	go func() {
		for {
			select {
			case <-c.watcherChannel.change:
				c.log.Debugf("receive file change event, wait for 1 minute to adapt")
				<-time.After(c.refreshByMinute)
				err := c.Refresh()
				if nil != err {
					c.log.Errorf("failed to read configuration from :%s error %s",
						c.fileName, err)
				}
				c.notify()
			case <-c.watcherChannel.remove:
				c.log.Warn("config file removed")
			}
		}

	}()
}

func (c *ConfigReaderData) UpdatePeriodically() {
	c.log.Info("star to update config perioditically")

	go func() {
		for {
			select {
			case <-time.After(c.refreshByMinute):
				err := c.Refresh()
				if nil != err {
					c.log.Errorf("failed to read configuration from :%s error %s",
						c.fileName, err)
				}
				c.notify()
			}
		}
	}()
}

func (c *ConfigReaderData) Refresh() error {
	configuration, err := c.parse()
	if nil != err {
		return err
	}
	c.update(configuration)
	return nil
}

func (c *ConfigReaderData) notify() {
	c.calendar.Refresh(c.currentConfiguration.Calendar)
	c.proofer.Refresh()
}

func (c *ConfigReaderData) parse() (*Configuration, error) {
	configuration, err := getConfiguration(c.fileName)
	if nil != err {
		return nil, err
	}
	return configuration, nil
}

func (c *ConfigReaderData) GetConfig() (*Configuration, string, error) {
	if nil == c.currentConfiguration {
		return nil, "", fmt.Errorf("configuration is empty")
	}
	return c.currentConfiguration, c.fileName, nil
}

func (c *ConfigReaderData) SetLog(log *logger.L) error {
	if nil == log {
		return fmt.Errorf("logger %v is nil", log)
	}
	c.log = log
	c.initialized = true
	return nil
}

func (c *ConfigReaderData) update(newConfiguration *Configuration) {
	c.currentConfiguration = newConfiguration
	c.threadCount = c.OptimalThreadCount()
	if c.initialized {
		c.log.Debugf("Updating configuration, target thread count %d, working: %t",
			c.threadCount,
			c.proofer.IsWorking(),
		)
	}
}

func (c *ConfigReaderData) updateCpuCount(count uint32) {
	if count > 0 {
		totalCPUCount = count
	}
}

func (c *ConfigReaderData) cpuCount() uint32 {
	return totalCPUCount
}

func (c *ConfigReaderData) OptimalThreadCount() uint32 {
	if !c.initialized {
		return uint32(minThreadCount)
	}
	percentage := float32(c.currentConfiguration.maxCPUUsage()) / 100
	threadCount := uint32(float32(totalCPUCount) * percentage)

	if threadCount <= minThreadCount {
		return minThreadCount
	}

	if threadCount > totalCPUCount {
		return totalCPUCount
	}

	return threadCount
}

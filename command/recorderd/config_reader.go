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

const (
	oneMinute          = time.Duration(1) * time.Minute
	minThreadCount     = 1
	ReaderLoggerPrefix = "config-reader"
)

var (
	totalCPUCount = uint32(runtime.NumCPU())
)

type ConfigReader struct {
	fileName             string
	refreshByMinute      time.Duration
	log                  *logger.L
	currentConfiguration *Configuration
	initialized          bool
	threadCount          uint32
	calendar             JobCalendar
	proofer              Proofer
}

func newConfigReader() *ConfigReader {
	return &ConfigReader{
		log:                  nil,
		currentConfiguration: nil,
		threadCount:          1,
		initialized:          false,
		refreshByMinute:      oneMinute,
	}
}

// configuration needs read first to know logger file location
func (c *ConfigReader) initialise(fileName string) {
	c.fileName = fileName
}

func (c *ConfigReader) setCalendar(calendar JobCalendar) {
	c.calendar = calendar
}

func (c *ConfigReader) setProofer(proofer Proofer) {
	c.proofer = proofer
}

func (c *ConfigReader) updatePeriodic() {
	c.log.Info("star to update config perioditically")
	go func() {
		for {
			select {
			case <-time.After(c.refreshByMinute):
				err := c.refresh()
				if nil != err {
					c.log.Errorf("failed to read configuration from :%s error %s",
						c.fileName, err)
				}
				c.notify()
			}
		}
	}()
}

func (c *ConfigReader) refresh() error {
	configuration, err := c.parse()
	if nil != err {
		return err
	}
	c.update(configuration)
	return nil
}

func (c *ConfigReader) notify() {
	c.calendar.refresh(c.currentConfiguration.Calendar)
	c.proofer.refresh()
}

func (c *ConfigReader) parse() (*Configuration, error) {
	configuration, err := getConfiguration(c.fileName)
	if nil != err {
		return nil, err
	}
	return configuration, nil
}

func (c *ConfigReader) getConfig() (*Configuration, error) {
	if nil == c.currentConfiguration {
		return nil, fmt.Errorf("configuration is empty")
	}
	return c.currentConfiguration, nil
}

func (c *ConfigReader) setLog(log *logger.L) error {
	if nil == log {
		return fmt.Errorf("logger %v is nil", log)
	}
	c.log = log
	c.initialized = true
	return nil
}

func (c *ConfigReader) update(newConfiguration *Configuration) {
	c.currentConfiguration = newConfiguration
	c.threadCount = c.optimalThreadCount()
	if c.initialized {
		c.log.Debugf("Updating configuration, target thread count %d, working: %t",
			c.threadCount,
			c.proofer.isWorking(),
		)
	}
}

func (c *ConfigReader) updateCpuCount(count uint32) {
	if count > 0 {
		totalCPUCount = count
	}
}

func (c *ConfigReader) cpuCount() uint32 {
	return totalCPUCount
}

func (c *ConfigReader) optimalThreadCount() uint32 {
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

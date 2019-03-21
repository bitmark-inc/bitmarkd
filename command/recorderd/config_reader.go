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
	initialise(string)
	optimalThreadCount() uint32
	setCalendar(JobCalendar)
	refresh() error
	getConfig() (*Configuration, error)
	setLog(*logger.L) error
	updatePeriodically()
	setProofer(Proofer)
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
}

func newConfigReader() ConfigReader {
	return &ConfigReaderData{
		log:                  nil,
		currentConfiguration: nil,
		threadCount:          1,
		initialized:          false,
		refreshByMinute:      oneMinute,
	}
}

// configuration needs read first to know logger file location
func (c *ConfigReaderData) initialise(fileName string) {
	c.fileName = fileName
}

func (c *ConfigReaderData) setCalendar(calendar JobCalendar) {
	c.calendar = calendar
}

func (c *ConfigReaderData) setProofer(proofer Proofer) {
	c.proofer = proofer
}

func (c *ConfigReaderData) updatePeriodically() {
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

func (c *ConfigReaderData) refresh() error {
	configuration, err := c.parse()
	if nil != err {
		return err
	}
	c.update(configuration)
	return nil
}

func (c *ConfigReaderData) notify() {
	c.calendar.refresh(c.currentConfiguration.Calendar)
	c.proofer.refresh()
}

func (c *ConfigReaderData) parse() (*Configuration, error) {
	configuration, err := getConfiguration(c.fileName)
	if nil != err {
		return nil, err
	}
	return configuration, nil
}

func (c *ConfigReaderData) getConfig() (*Configuration, error) {
	if nil == c.currentConfiguration {
		return nil, fmt.Errorf("configuration is empty")
	}
	return c.currentConfiguration, nil
}

func (c *ConfigReaderData) setLog(log *logger.L) error {
	if nil == log {
		return fmt.Errorf("logger %v is nil", log)
	}
	c.log = log
	c.initialized = true
	return nil
}

func (c *ConfigReaderData) update(newConfiguration *Configuration) {
	c.currentConfiguration = newConfiguration
	c.threadCount = c.optimalThreadCount()
	if c.initialized {
		c.log.Debugf("Updating configuration, target thread count %d, working: %t",
			c.threadCount,
			c.proofer.isWorking(),
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

func (c *ConfigReaderData) optimalThreadCount() uint32 {
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

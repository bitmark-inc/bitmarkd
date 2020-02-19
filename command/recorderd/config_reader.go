// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/bitmark-inc/logger"
)

// ConfigReader - methods supported by the configuration system
type ConfigReader interface {
	OptimalThreadCount() uint32
	SetCalendar(JobCalendar)
	FirstRefresh(string) error
	Refresh() error
	GetConfig() (*Configuration, error)
	SetLog(*logger.L) error
	SetWatcher(watcher FileWatcher)
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

// ConfigReaderData - result from reading configuration
type ConfigReaderData struct {
	refreshByMinute      time.Duration
	log                  *logger.L
	currentConfiguration *Configuration
	initialized          bool
	threadCount          uint32
	calendar             JobCalendar
	proofer              Proofer
	watcher              FileWatcher
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

func (c *ConfigReaderData) SetWatcher(watcher FileWatcher) {
	c.watcher = watcher
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
			case <-c.watcher.ChangeChannel():
				fileName := c.watcher.FileName()
				c.log.Info("receive file change event, wait for 1 minute to adapt")
				time.Sleep(c.refreshByMinute)
				err := c.Refresh()
				if nil != err {
					c.log.Errorf("failed to read configuration from %s error: %s",
						fileName, err)
				}
				c.notify()
			case <-c.watcher.RemoveChannel():
				c.log.Warn("config file removed")
			}
		}

	}()
}

func (c *ConfigReaderData) FirstRefresh(fileName string) error {
	configuration, err := getConfiguration(fileName)
	if nil != err {
		return err
	}
	c.update(configuration)
	return nil
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
	configuration, err := getConfiguration(c.watcher.FilePath())
	if nil != err {
		return nil, err
	}
	return configuration, nil
}

func (c *ConfigReaderData) GetConfig() (*Configuration, error) {
	if nil == c.currentConfiguration {
		return nil, fmt.Errorf("configuration is empty")
	}
	return c.currentConfiguration, nil
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

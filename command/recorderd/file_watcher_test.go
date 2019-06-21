// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/bitmark-inc/logger"
)

const (
	testFileName = "testWatcher"
)

var (
	removeChannel = make(chan struct{}, 1)
	changeChannel = make(chan struct{}, 1)
)

type FakeWatcher struct{}

func (f *FakeWatcher) Start() error {
	return nil
}
func (f *FakeWatcher) FileName() string {
	return "test"
}
func (f *FakeWatcher) FilePath() string {
	return "test"
}
func (f *FakeWatcher) ChangeChannel() <-chan struct{} {
	return make(chan struct{}, 1)
}
func (f *FakeWatcher) RemoveChannel() <-chan struct{} {
	return make(chan struct{}, 1)
}

func setupTestFileWatcher(t *testing.T) *FileWatcherData {
	removeTestFiles()
	setupLogger(t)
	w, _ := fsnotify.NewWatcher()
	filePath, _ := filepath.Abs(filepath.Clean(testFileName))

	fileWatcher := &FileWatcherData{
		watcher: w,
		log:     logger.New("test"),
		channel: WatcherChannel{
			change: changeChannel,
			remove: removeChannel,
		},
		filePath: filePath,
	}

	return fileWatcher
}

func TestStart(t *testing.T) {
	fileWatcher := setupTestFileWatcher(t)
	defer teardown()

	emptyFile, err := os.Create(fileWatcher.filePath)
	if nil != err {
		t.Errorf("create empty file error: %v", err)
	}
	emptyFile.Close()

	changed := false
	removed := false

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for {
			select {
			case <-fileWatcher.channel.change:
				if !changed {
					changed = true
					wg.Done()
				}
			case <-fileWatcher.channel.remove:
				if !removed {
					removed = true
					wg.Done()
				}
			}
		}
	}()

	go fileWatcher.Start()
	time.Sleep(time.Duration(1) * time.Second)

	err = ioutil.WriteFile(fileWatcher.filePath, []byte("test"), 0777)
	if nil != err {
		t.Errorf("write file error: %v", err)
	}

	wg.Wait()
	if !changed {
		t.Errorf("watcher not receive change event")
	}

	wg.Add(1)
	os.Remove(testFileName)
	wg.Wait()

	if !removed {
		t.Errorf("watcher not receive remove event")
	}
}

func TestIsChannelFull(t *testing.T) {
	w := setupTestFileWatcher(t)
	defer teardown()

	ch := make(chan struct{}, 1)
	expected := false
	actual := w.isChannelFull(ch)
	if actual != expected {
		t.Errorf("error get channel status, expected %t but get %t",
			expected, actual)
	}

	go func() {
		<-ch
	}()

	ch <- struct{}{}
	expected = true
	actual = w.isChannelFull(ch)
	if actual != expected {
		t.Errorf("error get channel status, expected %t but get %t",
			expected, actual)
	}
}

func TestSendEvent(t *testing.T) {
	w := setupTestFileWatcher(t)
	defer teardown()

	ch := make(chan struct{}, 1)
	expected := true
	actual := false

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-ch
		actual = true
		wg.Done()
	}()

	w.sendEvent(ch, "test")

	wg.Wait()

	if actual != expected {
		t.Errorf("error send channel event, expected %t but get %t",
			expected, actual)
	}
}

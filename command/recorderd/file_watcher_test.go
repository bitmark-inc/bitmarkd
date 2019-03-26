package main

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bitmark-inc/logger"
	"github.com/fsnotify/fsnotify"
)

const (
	testFileName = "testWatcher"
)

var (
	removeChannel = make(chan struct{}, 1)
	changeChannel = make(chan struct{}, 1)
)

func setupTestFileWatcher(t *testing.T) *FileWatcherData {
	removeTestFiles()
	setupLogger(t)
	w, _ := fsnotify.NewWatcher()

	fileWatcher := &FileWatcherData{
		watcher: w,
		reader: &ConfigReaderData{
			fileName:             testFileName,
			currentConfiguration: &Configuration{},
		},
		log: logger.New("test"),
		watcherData: WatcherData{
			channels: WatcherChannel{
				change: changeChannel,
				remove: removeChannel,
			},
			throttleInterval: time.Duration(0) * time.Second,
		},
	}

	return fileWatcher
}

func TestStart(t *testing.T) {
	fileWatcher := setupTestFileWatcher(t)
	defer teardown()

	changed := false
	removed := false

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		for {
			select {
			case <-fileWatcher.watcherData.channels.change:
				if !changed {
					changed = true
					wg.Done()
				}
			case <-fileWatcher.watcherData.channels.remove:
				if !removed {
					removed = true
					wg.Done()
				}
			}
		}
	}()
	time.Sleep(time.Duration(1) * time.Second)

	go fileWatcher.Start()

	f, _ := os.Create(testFileName)
	f.WriteString("start")
	f.Sync()
	f.Close()
	f, _ = os.OpenFile(testFileName, os.O_RDWR, 0666)
	f.WriteString("this is test string")
	f.Sync()
	f.Close()

	time.Sleep(time.Duration(1) * time.Second)
	os.Remove(testFileName)

	wg.Wait()

	if !changed || !removed {
		t.Errorf("watcher not receive event")
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

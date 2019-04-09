package main

import (
	"os"
	"path/filepath"
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

	f, _ := os.Create(testFileName)
	f.WriteString("start")
	f.Sync()
	f.Close()

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

	time.Sleep(time.Duration(1) * time.Second)

	go fileWatcher.Start()

	f, _ = os.OpenFile(testFileName, os.O_RDWR, 0666)
	f.WriteString("this is test string")
	f.Sync()
	f.Close()

	wg.Wait()
	wg.Add(1)
	os.Remove(testFileName)
	wg.Wait()

	fileWatcher.watcher.Close()

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
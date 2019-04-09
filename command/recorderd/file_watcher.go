package main

import (
	"errors"
	"os"
	"path"
	"path/filepath"

	"github.com/bitmark-inc/logger"
	"github.com/fsnotify/fsnotify"
)

type FileWatcher interface {
	Start() error
}

const (
	FileWatcherLoggerPrefix = "file-watcher"
)

type FileWatcherData struct {
	log         *logger.L
	watcherData WatcherData
	watcher     *fsnotify.Watcher
	filePath    string
}

type WatcherData struct {
	channels WatcherChannel
}

type WatcherChannel struct {
	change chan struct{}
	remove chan struct{}
}

func newFileWatcher(targetFile string, log *logger.L, data WatcherData) (FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if nil != err {
		log.Errorf("new watcher with error: %s", err.Error())
	}

	filePath, err := filepath.Abs(filepath.Clean(targetFile))
	if nil != err {
		log.Errorf("parse file %s error: %v", targetFile, err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, errors.New("file does not exist")
	}

	return &FileWatcherData{
		log:         log,
		watcher:     watcher,
		watcherData: data,
		filePath:    filePath,
	}, nil
}

func (w *FileWatcherData) Start() error {
	err := w.watcher.Add(w.filePath)
	if nil != err {
		w.log.Errorf("watcher add error: %v, abort", err)
		return err
	}

	go func() {
		for {
			event := <-w.watcher.Events
			w.log.Infof("file event: %v", event)
			change := w.watcherData.channels.change
			remove := w.watcherData.channels.remove

			if watcherEventFileRemove(event) {
				w.log.Errorf("file %s removed, stop", w.filePath)
				w.sendEvent(remove, "remove")
				return
			}

			if path.Base(event.Name) != path.Base(filepath.Clean(w.filePath)) {
				w.log.Infof("file %s not match, discard event", w.filePath)
				continue
			}

			if watcherEventFileChange(event) {
				w.log.Info("sending config change event...")
				w.sendEvent(change, "change")
			}
		}
	}()

	return nil
}

func (w *FileWatcherData) isChannelFull(ch chan<- struct{}) bool {
	return len(ch) == cap(ch)
}

func (w *FileWatcherData) sendEvent(ch chan<- struct{}, name string) {
	if !w.isChannelFull(ch) {
		ch <- struct{}{}
	} else {
		w.log.Infof("event channel %s full, discard event", name)
	}
}

func watcherEventFileRemove(event fsnotify.Event) bool {
	return event.Name == "" || event.Op&fsnotify.Remove == fsnotify.Remove
}

func watcherEventFileChange(event fsnotify.Event) bool {
	return event.Op&fsnotify.Write == fsnotify.Write ||
		event.Op&fsnotify.Chmod == fsnotify.Chmod
}

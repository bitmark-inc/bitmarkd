package main

import (
	"path/filepath"
	"time"

	"github.com/bitmark-inc/logger"
	"github.com/fsnotify/fsnotify"
)

type FileWatcher interface {
	Start()
}

const (
	FileWatcherLoggerPrefix = "file-watcher"
)

type FileWatcherData struct {
	reader      ConfigReader
	log         *logger.L
	watcherData WatcherData
	watcher     *fsnotify.Watcher
}

type WatcherData struct {
	channels         WatcherChannel
	throttleInterval time.Duration
}

type WatcherChannel struct {
	change chan struct{}
	remove chan struct{}
}

func newFileWatcher(reader ConfigReader, log *logger.L, data WatcherData) FileWatcher {
	watcher, err := fsnotify.NewWatcher()
	if nil != err {
		log.Errorf("new watcher with error: %s", err.Error())
	}
	return &FileWatcherData{
		reader:      reader,
		log:         log,
		watcher:     watcher,
		watcherData: data,
	}
}

func (w *FileWatcherData) Start() {
	_, fileName, _ := w.reader.GetConfig()
	filePath, _ := filepath.Abs(filepath.Clean(fileName))

	w.watcher.Add(filePath)

	go func() {
		for {
			event := <-w.watcher.Events
			w.log.Infof("file event: %v", event)
			remove := w.watcherData.channels.remove
			change := w.watcherData.channels.change

			if event.Name != filepath.Clean(filePath) {
				w.log.Debug("file not match, discard")
				continue
			}

			if event.Op&fsnotify.Remove == fsnotify.Remove {
				if !w.isChannelFull(remove) {
					remove <- struct{}{}
				} else {
					w.log.Info("remove channel is full, discard event")
				}
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				if !w.isChannelFull(change) {
					change <- struct{}{}
				} else {
					w.log.Info("change channel is full, discard event")
				}
			}
		}
	}()
}

func (w *FileWatcherData) isChannelFull(ch chan struct{}) bool {
	return len(ch) == cap(ch)
}

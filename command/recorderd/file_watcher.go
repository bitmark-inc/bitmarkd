package main

import (
	"errors"
	"fmt"
	"os"
	"path"
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
	log         *logger.L
	watcherData WatcherData
	watcher     *fsnotify.Watcher
	filePath    string
}

type WatcherData struct {
	channels         WatcherChannel
	throttleInterval time.Duration
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

func (w *FileWatcherData) Start() {
	err := w.watcher.Add(w.filePath)
	if nil != err {
		w.log.Errorf("watcher add error: %v", err)
	}

	go func() {
		for {
			event := <-w.watcher.Events
			fmt.Printf("file event: %v\n", event)
			w.log.Infof("file event: %v", event)
			remove := w.watcherData.channels.remove
			change := w.watcherData.channels.change

			if path.Base(event.Name) != path.Base(filepath.Clean(w.filePath)) {
				w.log.Infof("file %s not match, discard event", w.filePath)
				continue
			}

			if event.Op&fsnotify.Remove == fsnotify.Remove {
				w.log.Info("sending file remove event")
				if !w.isChannelFull(remove) {
					remove <- struct{}{}
				} else {
					w.log.Info("remove channel is full, discard event")
				}
			}

			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Chmod == fsnotify.Chmod {
				w.log.Info("sending config change event...")
				if !w.isChannelFull(change) {
					change <- struct{}{}
				} else {
					w.log.Info("config change event channel full, discard event")
				}
			}
		}
	}()
}

func (w *FileWatcherData) isChannelFull(ch chan struct{}) bool {
	return len(ch) == cap(ch)
}

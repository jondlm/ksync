package util

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// Debounce will return a function that is debounced based on the interval you
// request
func Debounce(interval time.Duration, f func()) func() {
	var timer *time.Timer

	return func() {
		if timer == nil {
			timer = time.NewTimer(interval)

			go func() {
				<-timer.C
				timer.Stop()
				timer = nil
				f()
			}()
		} else {
			timer.Reset(interval)
		}
	}
}

// Watch should do stuff
func Watch(dir string, watcher *fsnotify.Watcher, log *logrus.Logger) {
	log.WithFields(logrus.Fields{"dir": dir}).Debug("walking directory")
	watcher.Add(dir)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		log.WithFields(logrus.Fields{"dir": dir}).Debug("adding watch")
		if err != nil {
			log.Errorf("failure to access %q: %v\n", dir, err)
			return err
		}

		if info.IsDir() && path != dir {
			Watch(path, watcher, log)
		}

		return nil
	})

	if err != nil {
		log.Errorf("error walking the path %q: %v\n", dir, err)
	}
}

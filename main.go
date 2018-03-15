package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jawher/mow.cli"
	"github.com/sirupsen/logrus"
)

// This gets set by `go build -ldflags "-X main.version=1.0.0"`
var version string
var log *logrus.Logger

func debounce(interval time.Duration, f func()) func() {
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

func watch(dir string, watcher *fsnotify.Watcher) {
	log.WithFields(logrus.Fields{"dir": dir}).Debug("walking directory")
	watcher.Add(dir)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		log.WithFields(logrus.Fields{"dir": dir}).Debug("adding watch")
		if err != nil {
			log.Errorf("failure to access %q: %v\n", dir, err)
			return err
		}

		if info.IsDir() && path != dir {
			watch(path, watcher)
		}

		return nil
	})

	if err != nil {
		log.Errorf("error walking the path %q: %v\n", dir, err)
	}
}

func main() {
	app := cli.App("lazywatch", "Debounced directory watch")

	app.Spec = "[-v] DIR CMD [ARG...]"
	app.Version("version", version)

	var (
		dir     = app.StringArg("DIR", "", "directory to watch")
		cmd     = app.StringArg("CMD", "", "command to run")
		args    = app.StringsArg("ARG", []string{}, "argument to the command")
		verbose = app.BoolOpt("v verbose", false, "verbose logging")
	)

	app.Before = func() {
		log = logrus.New()
		log.Formatter = new(logrus.TextFormatter)
		log.Out = os.Stdout

		if *verbose {
			log.Level = logrus.DebugLevel
		} else {
			log.Level = logrus.InfoLevel
		}
	}

	app.Action = func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		watch(*dir, watcher)

		running := false
		update := debounce(time.Second, func() {
			if running {
				return
			}
			newArgs := []string{}

			for _, a := range *args {
				newArgs = append(newArgs, a)
			}

			log.WithFields(logrus.Fields{
				"cmd": *cmd,
			}).Info("executing command")

			running = true
			c := exec.Command(*cmd, newArgs...)
			output, err := c.CombinedOutput()

			if err != nil {
				log.WithFields(logrus.Fields{"err": err}).Info("error while running command")
			} else {
				log.Info("success")
			}

			running = false
			fmt.Printf(string(output))
		})

		// Watch loop
		go func() {
			for {
				select {
				case event := <-watcher.Events:
					update()
					if event.Op&fsnotify.Create == fsnotify.Create {
						fi, _ := os.Stat(event.Name)

						if fi.IsDir() {
							watch(event.Name, watcher)
						}
					}
				case err := <-watcher.Errors:
					log.Error(err)
				}
			}
		}()

		// Keep the app running
		done := make(chan bool)
		<-done
	}

	app.Run(os.Args)
}

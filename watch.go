// Copyright 2013 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	cmd          *exec.Cmd
	state        sync.Mutex
	eventTime    = make(map[string]int64)
	scheduleTime time.Time
)

// NewWatcher starts an fsnotify Watcher on the specified paths
func NewWatcher(paths []string, files []string, isgenerate bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Fatalf("Failed to create watcher: %s", err)
	}

	go func() {
		for {
			select {
			case e := <-watcher.Events:
				isbuild := true

				// Skip ignored files
				if shouldIgnoreFile(e.Name) {
					continue
				}
				if !checkIfWatchExt(e.Name) {
					continue
				}

				mt := getFileModTime(e.Name)
				if t := eventTime[e.Name]; mt == t {
					logger.Infof(bold("Skipping: ")+"%s", e.String())
					isbuild = false
				}

				eventTime[e.Name] = mt

				if isbuild {
					logger.Infof("Event fired: %s", e)
					go func() {
						// Wait 1s before autobuild util there is no file change.
						scheduleTime = time.Now().Add(1 * time.Second)
						for {
							time.Sleep(scheduleTime.Sub(time.Now()))
							if time.Now().After(scheduleTime) {
								break
							}
							return
						}

						AutoBuild(files, isgenerate)
					}()
				}
			case err := <-watcher.Errors:
				logger.Warnf("Watcher error: %s", err.Error()) // No need to exit here
			}
		}
	}()

	logger.Info("Initializing watcher...")
	for _, path := range paths {
		logger.Infof(bold("Watching: ")+"%s", path)
		err = watcher.Add(path)
		if err != nil {
			logger.Fatalf("Failed to watch directory: %s", err)
		}
	}

}

// getFileModTime returns unix timestamp of `os.File.ModTime` for the given path.
func getFileModTime(path string) int64 {
	path = strings.Replace(path, "\\", "/", -1)
	f, err := os.Open(path)
	if err != nil {
		logger.Errorf("Failed to open file on '%s': %s", path, err)
		return time.Now().Unix()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		logger.Errorf("Failed to get file stats: %s", err)
		return time.Now().Unix()
	}

	return fi.ModTime().Unix()
}

// AutoBuild builds the specified set of files
func AutoBuild(files []string, isgenerate bool) {
	state.Lock()
	defer state.Unlock()

	os.Chdir(currpath)

	cmdName := "go"
	if conf.Gopm.Enable {
		cmdName = "gopm"
	}

	var (
		err    error
		stderr bytes.Buffer
		stdout bytes.Buffer
	)
	// For applications use full import path like "github.com/.../.."
	// are able to use "go install" to reduce build time.
	if conf.GoInstall {
		icmd := exec.Command(cmdName, "install", "-v")
		icmd.Stdout = os.Stdout
		icmd.Stderr = os.Stderr
		icmd.Env = append(os.Environ(), "GOGC=off")
		icmd.Run()
	}
	if conf.Gopm.Install {
		icmd := exec.Command("go", "list", "./...")
		icmd.Stdout = &stdout
		icmd.Env = append(os.Environ(), "GOGC=off")
		err = icmd.Run()
		if err == nil {
			list := strings.Split(stdout.String(), "\n")[1:]
			for _, pkg := range list {
				if len(pkg) == 0 {
					continue
				}
				icmd = exec.Command(cmdName, "install", pkg)
				icmd.Stdout = os.Stdout
				icmd.Stderr = os.Stderr
				icmd.Env = append(os.Environ(), "GOGC=off")
				err = icmd.Run()
				if err != nil {
					break
				}
			}
		}
	}

	if isgenerate {
		logger.Info("Generating the docs...")
		icmd := exec.Command("bee", "generate", "docs")
		icmd.Env = append(os.Environ(), "GOGC=off")
		err = icmd.Run()
		if err != nil {
			logger.Errorf("Failed to generate the docs.")
			return
		}
		logger.Success("Docs generated!")
	}

	if err == nil {
		appName := appname
		if runtime.GOOS == "windows" {
			appName += ".exe"
		}

		args := []string{"build"}
		args = append(args, "-o", appName)
		if buildTags != "" {
			args = append(args, "-tags", buildTags)
		}
		args = append(args, files...)

		bcmd := exec.Command(cmdName, args...)
		bcmd.Env = append(os.Environ(), "GOGC=off")
		bcmd.Stderr = &stderr
		err = bcmd.Run()
		if err != nil {
			logger.Errorf("Failed to build the application: %s", stderr.String())
			return
		}
	}

	logger.Success("Built Successfully!")
	Restart(appname)
}

// Kill kills the running command process
func Kill() {
	defer func() {
		if e := recover(); e != nil {
			logger.Infof("Kill recover: %s", e)
		}
	}()
	if cmd != nil && cmd.Process != nil {
		err := cmd.Process.Kill()
		if err != nil {
			logger.Errorf("Error while killing cmd process: %s", err)
		}
	}
}

// Restart kills the running command process and starts it again
func Restart(appname string) {
	logger.Debugf("Kill running process", __FILE__(), __LINE__())
	Kill()
	go Start(appname)
}

// Start starts the command process
func Start(appname string) {
	logger.Infof("Restarting '%s'...", appname)
	if strings.Index(appname, "./") == -1 {
		appname = "./" + appname
	}

	cmd = exec.Command(appname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Args = append([]string{appname}, conf.CmdArgs...)
	cmd.Env = append(os.Environ(), conf.Envs...)

	go cmd.Run()
	logger.Successf("'%s' is running...", appname)
	started <- true
}

// shouldIgnoreFile ignores filenames generated by Emacs, Vim or SublimeText.
// It returns true if the file should be ignored, false otherwise.
func shouldIgnoreFile(filename string) bool {
	for _, regex := range ignoredFilesRegExps {
		r, err := regexp.Compile(regex)
		if err != nil {
			logger.Fatalf("Could not compile regular expression: %s", err)
		}
		if r.MatchString(filename) {
			return true
		}
		continue
	}
	return false
}

var watchExts = []string{".go"}
var ignoredFilesRegExps = []string{
	`.#(\w+).go`,
	`.(\w+).go.swp`,
	`(\w+).go~`,
	`(\w+).tmp`,
}

// checkIfWatchExt returns true if the name HasSuffix <watch_ext>.
func checkIfWatchExt(name string) bool {
	for _, s := range watchExts {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

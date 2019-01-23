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

package run

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/beego/bee/config"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/logger/colors"
	"github.com/beego/bee/utils"
	"github.com/fsnotify/fsnotify"
)

var (
	cmd                 *exec.Cmd
	state               sync.Mutex
	eventTime           = make(map[string]int64)
	scheduleTime        time.Time
	watchExts           = config.Conf.WatchExts
	watchExtsStatic     = config.Conf.WatchExtsStatic
	ignoredFilesRegExps = []string{
		`.#(\w+).go`,
		`.(\w+).go.swp`,
		`(\w+).go~`,
		`(\w+).tmp`,
		`commentsRouter_controllers.go`,
	}
)

// NewWatcher starts an fsnotify Watcher on the specified paths
func NewWatcher(paths []string, files []string, isgenerate bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		beeLogger.Log.Fatalf("Failed to create watcher: %s", err)
	}

	go func() {
		for {
			select {
			case e := <-watcher.Events:
				isBuild := true

				if ifStaticFile(e.Name) && config.Conf.EnableReload {
					sendReload(e.String())
					continue
				}
				// Skip ignored files
				if shouldIgnoreFile(e.Name) {
					continue
				}
				if !shouldWatchFileWithExtension(e.Name) {
					continue
				}

				mt := utils.GetFileModTime(e.Name)
				if t := eventTime[e.Name]; mt == t {
					beeLogger.Log.Hintf(colors.Bold("Skipping: ")+"%s", e.String())
					isBuild = false
				}

				eventTime[e.Name] = mt

				if isBuild {
					beeLogger.Log.Hintf("Event fired: %s", e)
					go func() {
						// Wait 1s before autobuild until there is no file change.
						scheduleTime = time.Now().Add(1 * time.Second)
						time.Sleep(time.Until(scheduleTime))
						AutoBuild(files, isgenerate)

						if config.Conf.EnableReload {
							// Wait 100ms more before refreshing the browser
							time.Sleep(100 * time.Millisecond)
							sendReload(e.String())
						}
					}()
				}
			case err := <-watcher.Errors:
				beeLogger.Log.Warnf("Watcher error: %s", err.Error()) // No need to exit here
			}
		}
	}()

	beeLogger.Log.Info("Initializing watcher...")
	for _, path := range paths {
		beeLogger.Log.Hintf(colors.Bold("Watching: ")+"%s", path)
		err = watcher.Add(path)
		if err != nil {
			beeLogger.Log.Fatalf("Failed to watch directory: %s", err)
		}
	}
}

// AutoBuild builds the specified set of files
func AutoBuild(files []string, isgenerate bool) {
	state.Lock()
	defer state.Unlock()

	os.Chdir(currpath)

	cmdName := "go"

	var (
		err    error
		stderr bytes.Buffer
	)
	// For applications use full import path like "github.com/.../.."
	// are able to use "go install" to reduce build time.
	if config.Conf.GoInstall {
		icmd := exec.Command(cmdName, "install", "-v")
		icmd.Stdout = os.Stdout
		icmd.Stderr = os.Stderr
		icmd.Env = append(os.Environ(), "GOGC=off")
		icmd.Run()
	}

	if isgenerate {
		beeLogger.Log.Info("Generating the docs...")
		icmd := exec.Command("bee", "generate", "docs")
		icmd.Env = append(os.Environ(), "GOGC=off")
		err = icmd.Run()
		if err != nil {
			utils.Notify("", "Failed to generate the docs.")
			beeLogger.Log.Errorf("Failed to generate the docs.")
			return
		}
		beeLogger.Log.Success("Docs generated!")
	}
	appName := appname
	if err == nil {

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
			utils.Notify(stderr.String(), "Build Failed")
			beeLogger.Log.Errorf("Failed to build the application: %s", stderr.String())
			return
		}
	}

	beeLogger.Log.Success("Built Successfully!")
	Restart(appName)
}

// Kill kills the running command process
func Kill() {
	defer func() {
		if e := recover(); e != nil {
			beeLogger.Log.Infof("Kill recover: %s", e)
		}
	}()
	if cmd != nil && cmd.Process != nil {
		// Windows does not support Interrupt
		if runtime.GOOS == "windows" {
			cmd.Process.Signal(os.Kill)
		} else {
			cmd.Process.Signal(os.Interrupt)
		}

		ch := make(chan struct{}, 1)
		go func() {
			cmd.Wait()
			ch <- struct{}{}
		}()

		select {
		case <-ch:
			return
		case <-time.After(10 * time.Second):
			beeLogger.Log.Info("Timeout. Force kill cmd process")
			err := cmd.Process.Kill()
			if err != nil {
				beeLogger.Log.Errorf("Error while killing cmd process: %s", err)
			}
			return
		}
	}
}

// Restart kills the running command process and starts it again
func Restart(appname string) {
	beeLogger.Log.Debugf("Kill running process", utils.FILE(), utils.LINE())
	Kill()
	go Start(appname)
}

// Start starts the command process
func Start(appname string) {
	beeLogger.Log.Infof("Restarting '%s'...", appname)
	if !strings.Contains(appname, "./") {
		appname = "./" + appname
	}

	cmd = exec.Command(appname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if runargs != "" {
		r := regexp.MustCompile("'.+'|\".+\"|\\S+")
		m := r.FindAllString(runargs, -1)
		cmd.Args = append([]string{appname}, m...)
	} else {
		cmd.Args = append([]string{appname}, config.Conf.CmdArgs...)
	}
	cmd.Env = append(os.Environ(), config.Conf.Envs...)

	go cmd.Run()
	beeLogger.Log.Successf("'%s' is running...", appname)
	started <- true
}

func ifStaticFile(filename string) bool {
	for _, s := range watchExtsStatic {
		if strings.HasSuffix(filename, s) {
			return true
		}
	}
	return false
}

// shouldIgnoreFile ignores filenames generated by Emacs, Vim or SublimeText.
// It returns true if the file should be ignored, false otherwise.
func shouldIgnoreFile(filename string) bool {
	for _, regex := range ignoredFilesRegExps {
		r, err := regexp.Compile(regex)
		if err != nil {
			beeLogger.Log.Fatalf("Could not compile regular expression: %s", err)
		}
		if r.MatchString(filename) {
			return true
		}
		continue
	}
	return false
}

// shouldWatchFileWithExtension returns true if the name of the file
// hash a suffix that should be watched.
func shouldWatchFileWithExtension(name string) bool {
	for _, s := range watchExts {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

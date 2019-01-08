// Copyright 2017 bee authors
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

// Package dlv ...
package dlv

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
	"github.com/derekparker/delve/service"
	"github.com/derekparker/delve/service/rpc2"
	"github.com/derekparker/delve/service/rpccommon"
	"github.com/derekparker/delve/terminal"
	"github.com/fsnotify/fsnotify"
)

var cmdDlv = &commands.Command{
	CustomFlags: true,
	UsageLine:   "dlv [-package=\"\"] [-port=8181] [-verbose=false]",
	Short:       "Start a debugging session using Delve",
	Long: `dlv command start a debugging session using debugging tool Delve.

  To debug your application using Delve, use: {{"$ bee dlv" | bold}}

  For more information on Delve: https://github.com/derekparker/delve
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    runDlv,
}

var (
	packageName string
	verbose     bool
	port        int
)

func init() {
	fs := flag.NewFlagSet("dlv", flag.ContinueOnError)
	fs.StringVar(&packageName, "package", "", "The package to debug (Must have a main package)")
	fs.BoolVar(&verbose, "verbose", false, "Enable verbose mode")
	fs.IntVar(&port, "port", 8181, "Port to listen to for clients")
	cmdDlv.Flag = *fs
	commands.AvailableCommands = append(commands.AvailableCommands, cmdDlv)
}

func runDlv(cmd *commands.Command, args []string) int {
	if err := cmd.Flag.Parse(args); err != nil {
		beeLogger.Log.Fatalf("Error while parsing flags: %v", err.Error())
	}

	var (
		addr       = fmt.Sprintf("127.0.0.1:%d", port)
		paths      = make([]string, 0)
		notifyChan = make(chan int)
	)

	if err := loadPathsToWatch(&paths); err != nil {
		beeLogger.Log.Fatalf("Error while loading paths to watch: %v", err.Error())
	}
	go startWatcher(paths, notifyChan)
	return startDelveDebugger(addr, notifyChan)
}

// buildDebug builds a debug binary in the current working directory
func buildDebug() (string, error) {
	args := []string{"-gcflags", "-N -l", "-o", "debug"}
	args = append(args, utils.SplitQuotedFields("-ldflags='-linkmode internal'")...)
	args = append(args, packageName)
	if err := utils.GoCommand("build", args...); err != nil {
		return "", err
	}

	fp, err := filepath.Abs("./debug")
	if err != nil {
		return "", err
	}
	return fp, nil
}

// loadPathsToWatch loads the paths that needs to be watched for changes
func loadPathsToWatch(paths *[]string) error {
	directory, err := os.Getwd()
	if err != nil {
		return err
	}
	filepath.Walk(directory, func(path string, info os.FileInfo, _ error) error {
		if strings.HasSuffix(info.Name(), "docs") {
			return filepath.SkipDir
		}
		if strings.HasSuffix(info.Name(), "swagger") {
			return filepath.SkipDir
		}
		if strings.HasSuffix(info.Name(), "vendor") {
			return filepath.SkipDir
		}

		if filepath.Ext(info.Name()) == ".go" {
			*paths = append(*paths, path)
		}
		return nil
	})
	return nil
}

// startDelveDebugger starts the Delve debugger server
func startDelveDebugger(addr string, ch chan int) int {
	beeLogger.Log.Info("Starting Delve Debugger...")

	fp, err := buildDebug()
	if err != nil {
		beeLogger.Log.Fatalf("Error while building debug binary: %v", err)
	}
	defer os.Remove(fp)

	abs, err := filepath.Abs("./debug")
	if err != nil {
		beeLogger.Log.Fatalf("%v", err)
	}

	// Create and start the debugger server
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		beeLogger.Log.Fatalf("Could not start listener: %s", err)
	}
	defer listener.Close()

	server := rpccommon.NewServer(&service.Config{
		Listener:    listener,
		AcceptMulti: true,
		AttachPid:   0,
		APIVersion:  2,
		WorkingDir:  ".",
		ProcessArgs: []string{abs},
	}, false)
	if err := server.Run(); err != nil {
		beeLogger.Log.Fatalf("Could not start debugger server: %v", err)
	}

	// Start the Delve client REPL
	client := rpc2.NewClient(addr)
	// Make sure the client is restarted when new changes are introduced
	go func() {
		for {
			if val := <-ch; val == 0 {
				if _, err := client.Restart(); err != nil {
					utils.Notify("Error while restarting the client: "+err.Error(), "bee")
				} else {
					if verbose {
						utils.Notify("Delve Debugger Restarted", "bee")
					}
				}
			}
		}
	}()

	// Create the terminal and connect it to the client debugger
	term := terminal.New(client, nil)
	status, err := term.Run()
	if err != nil {
		beeLogger.Log.Fatalf("Could not start Delve REPL: %v", err)
	}

	// Stop and kill the debugger server once user quits the REPL
	if err := server.Stop(true); err != nil {
		beeLogger.Log.Fatalf("Could not stop Delve server: %v", err)
	}
	return status
}

var eventsModTime = make(map[string]int64)

// startWatcher starts the fsnotify watcher on the passed paths
func startWatcher(paths []string, ch chan int) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		beeLogger.Log.Fatalf("Could not start the watcher: %v", err)
	}
	defer watcher.Close()

	// Feed the paths to the watcher
	for _, path := range paths {
		if err := watcher.Add(path); err != nil {
			beeLogger.Log.Fatalf("Could not set a watch on path: %v", err)
		}
	}

	for {
		select {
		case evt := <-watcher.Events:
			build := true
			if filepath.Ext(evt.Name) != ".go" {
				continue
			}

			mt := utils.GetFileModTime(evt.Name)
			if t := eventsModTime[evt.Name]; mt == t {
				build = false
			}
			eventsModTime[evt.Name] = mt

			if build {
				go func() {
					if verbose {
						utils.Notify("Rebuilding application with the new changes", "bee")
					}

					// Wait 1s before re-build until there is no file change
					scheduleTime := time.Now().Add(1 * time.Second)
					time.Sleep(time.Until(scheduleTime))
					_, err := buildDebug()
					if err != nil {
						utils.Notify("Build Failed: "+err.Error(), "bee")
					} else {
						ch <- 0 // Notify listeners
					}
				}()
			}
		case err := <-watcher.Errors:
			if err != nil {
				ch <- -1
			}
		}
	}
}

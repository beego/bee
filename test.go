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
	"os"
	"os/exec"
	path "path/filepath"
	"time"

	_ "github.com/smartystreets/goconvey/convey"
)

var cmdTest = &Command{
	UsageLine: "test [appname]",
	Short:     "test the app",
	Long:      ``,
}

func init() {
	cmdTest.Run = testApp
}

func safePathAppend(arr []string, paths ...string) []string {
	for _, path := range paths {
		if pathExists(path) {
			arr = append(arr, path)
		}
	}
	return arr
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

var started = make(chan bool)

func testApp(cmd *Command, args []string) int {
	if len(args) != 1 {
		ColorLog("[ERRO] Cannot start running[ %s ]\n",
			"argument 'appname' is missing")
		os.Exit(2)
	}
	crupath, _ := os.Getwd()
	Debugf("current path:%s\n", crupath)

	err := loadConfig()
	if err != nil {
		ColorLog("[ERRO] Fail to parse bee.json[ %s ]\n", err)
	}
	var paths []string
	readAppDirectories(crupath, &paths)

	NewWatcher(paths, nil, false)
	appname = args[0]
	for {
		select {
		case <-started:
			runTest()
		}
	}
	return 0
}

func runTest() {
	ColorLog("[INFO] Start testing...\n")
	time.Sleep(time.Second * 1)
	crupwd, _ := os.Getwd()
	testDir := path.Join(crupwd, "tests")
	if pathExists(testDir) {
		os.Chdir(testDir)
	}

	var err error
	icmd := exec.Command("go", "test")
	icmd.Stdout = os.Stdout
	icmd.Stderr = os.Stderr
	ColorLog("[TRAC] ============== Test Begin ===================\n")
	err = icmd.Run()
	ColorLog("[TRAC] ============== Test End ===================\n")

	if err != nil {
		ColorLog("[ERRO] ============== Test failed ===================\n")
		ColorLog("[ERRO] %s", err)
		return
	}
	ColorLog("[SUCC] Test finish\n")
}

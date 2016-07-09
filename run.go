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
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	path "path/filepath"
	"runtime"
	"strings"
)

var cmdRun = &Command{
	UsageLine: "run [appname] [watchall] [-main=*.go] [-downdoc=true]  [-gendoc=true]  [-e=Godeps -e=folderToExclude]  [-tags=goBuildTags] [-rip=serverIp] [-rpassword=serverPassword]",
	Short:     "run the app and start a Web server for development",
	Long: `
Run command will supervise the file system of the beego project using inotify,
it will recompile and restart the app after any modifications.

`,
}

var mainFiles ListOpts

var downdoc docValue
var gendoc docValue

// The flags list of the paths excluded from watching
var excludedPaths strFlags

// Pass through to -tags arg of "go build"
var buildTags string

// Info about restart notifications server
var rnsAddress string
var rnsPassword string

func init() {
	cmdRun.Run = runApp
	cmdRun.Flag.Var(&mainFiles, "main", "specify main go files")
	cmdRun.Flag.Var(&gendoc, "gendoc", "auto generate the docs")
	cmdRun.Flag.Var(&downdoc, "downdoc", "auto download swagger file when not exist")
	cmdRun.Flag.Var(&excludedPaths, "e", "Excluded paths[].")
	cmdRun.Flag.StringVar(&buildTags, "tags", "", "Build tags (https://golang.org/pkg/go/build/)")
	cmdRun.Flag.StringVar(&rnsAddress, "rip", "", "Address of server which receive signal to restart application")
	cmdRun.Flag.StringVar(&rnsPassword, "rpassword", "", "Password which should be passed to server to restart application")
}

var appname string

func runApp(cmd *Command, args []string) int {
	fmt.Println("bee   :" + version)
	fmt.Println("beego :" + getbeegoVersion())
	goversion, err := exec.Command("go", "version").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Go    :" + string(goversion))

	exit := make(chan bool)
	crupath, _ := os.Getwd()

	if len(args) == 0 || args[0] == "watchall" {
		appname = path.Base(crupath)
		ColorLog("[INFO] Uses '%s' as 'appname'\n", appname)
	} else {
		appname = args[0]
		ColorLog("[INFO] Uses '%s' as 'appname'\n", appname)
		if strings.HasSuffix(appname, ".go") && isExist(path.Join(crupath, appname)) {
			ColorLog("[WARN] The appname has conflic with crupath's file, do you want to build appname as %s\n", appname)
			ColorLog("[INFO] Do you want to overwrite it? [yes|no]]  ")
			if !askForConfirmation() {
				return 0
			}
		}
	}
	Debugf("current path:%s\n", crupath)

	err = loadConfig()
	if err != nil {
		ColorLog("[ERRO] Fail to parse bee.json[ %s ]\n", err)
	}

	var paths []string

	readAppDirectories(crupath, &paths)

	// Because monitor files has some issues, we watch current directory
	// and ignore non-go files.
	gps := GetGOPATHs()
	if len(gps) == 0 {
		ColorLog("[ERRO] Fail to start[ %s ]\n", "$GOPATH is not set or empty")
		os.Exit(2)
	}
	gopath := gps[0]
	for _, p := range conf.DirStruct.Others {
		paths = append(paths, strings.Replace(p, "$GOPATH", gopath, -1))
	}

	files := []string{}
	for _, arg := range mainFiles {
		if len(arg) > 0 {
			files = append(files, arg)
		}
	}

	isgenerate := gendoc == "true"

	if rnsAddress != "" {
		NewRestartNotificationServer(rnsAddress, []byte(rnsPassword), files, isgenerate)
	}
	NewWatcher(paths, files, isgenerate)
	Autobuild(files, isgenerate)

	if downdoc == "true" {
		if _, err := os.Stat(path.Join(crupath, "swagger")); err != nil {
			if os.IsNotExist(err) {
				downloadFromUrl(swaggerlink, "swagger.zip")
				unzipAndDelete("swagger.zip", "swagger")
			}
		}
	}
	for {
		select {
		case <-exit:
			runtime.Goexit()
		}
	}
}

func readAppDirectories(directory string, paths *[]string) {
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		return
	}

	useDirectory := false
	for _, fileInfo := range fileInfos {
		if strings.HasSuffix(fileInfo.Name(), "docs") {
			continue
		}

		if isExcluded(path.Join(directory, fileInfo.Name())) {
			continue
		}

		if fileInfo.IsDir() == true && fileInfo.Name()[0] != '.' {
			readAppDirectories(directory+"/"+fileInfo.Name(), paths)
			continue
		}

		if useDirectory == true {
			continue
		}

		if path.Ext(fileInfo.Name()) == ".go" {
			*paths = append(*paths, directory)
			useDirectory = true
		}
	}

	return
}

// If a file is excluded
func isExcluded(filePath string) bool {
	for _, p := range excludedPaths {
		absP, err := path.Abs(p)
		if err != nil {
			ColorLog("[ERROR] Can not get absolute path of [ %s ]\n", p)
			continue
		}
		absFilePath, err := path.Abs(filePath)
		if err != nil {
			ColorLog("[ERROR] Can not get absolute path of [ %s ]\n", filePath)
			break
		}
		if strings.HasPrefix(absFilePath, absP) {
			ColorLog("[INFO] Excluding from watching [ %s ]\n", filePath)
			return true
		}
	}
	return false
}

// NewRestartNotificationServer Creates tcp server which receive signal to restart app
func NewRestartNotificationServer(addr string, password []byte, files []string, isgenerate bool) error {
	ColorLog("[INFO] Starting restart notification server on addres %s\n", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		ColorLog("[ERRO] Failt to start restart notification server [%s]\n", err)
		return err
	}
	go func() {
		for {
			client, err := ln.Accept()
			if err != nil {
				ColorLog("[ERRO] Fail to accept client[ %s ]\n", err)
				continue
			}
			if len(password) == 0 {
				Autobuild(files, isgenerate)
				client.Close()
				continue
			}
			data, err := ioutil.ReadAll(client)
			if err != nil {
				ColorLog("[ERRO] Fail to read data from client[ %s ]\n", err)
			}
			data = bytes.TrimSpace(data)
			if bytes.Compare(data, password) != 0 {
				ColorLog("[ERRO] Invalid password to restart app: %v\n", string(data))
				continue
			}
			Autobuild(files, isgenerate)
			client.Close()
		}
	}()
	return nil
}

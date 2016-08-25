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
	"fmt"
	"io/ioutil"
	"os"
	path "path/filepath"
	"runtime"
	"strings"
)

var cmdRun = &Command{
	UsageLine: "run [appname] [watchall] [-main=*.go] [-downdoc=true]  [-gendoc=true] [-vendor=true] [-e=folderToExclude]  [-tags=goBuildTags] [-runmode=BEEGO_RUNMODE]",
	Short:     "run the app and start a Web server for development",
	Long: `
Run command will supervise the file system of the beego project using inotify,
it will recompile and restart the app after any modifications.

`,
}

var (
	mainFiles ListOpts
	downdoc   docValue
	gendoc    docValue
	// The flags list of the paths excluded from watching
	excludedPaths strFlags
	// Pass through to -tags arg of "go build"
	buildTags string
	// Application path
	currpath string
	// Application name
	appname string
	// Channel to signal an Exit
	exit chan bool
	// Flag to watch the vendor folder
	vendorWatch bool
	// Current user workspace
	currentGoPath string
	// Current runmode
	runmode string
)

func init() {
	cmdRun.Run = runApp
	cmdRun.Flag.Var(&mainFiles, "main", "specify main go files")
	cmdRun.Flag.Var(&gendoc, "gendoc", "auto generate the docs")
	cmdRun.Flag.Var(&downdoc, "downdoc", "auto download swagger file when not exist")
	cmdRun.Flag.Var(&excludedPaths, "e", "Excluded paths[].")
	cmdRun.Flag.BoolVar(&vendorWatch, "vendor", false, "Watch vendor folder")
	cmdRun.Flag.StringVar(&buildTags, "tags", "", "Build tags (https://golang.org/pkg/go/build/)")
	cmdRun.Flag.StringVar(&runmode, "runmode", "", "Set BEEGO_RUNMODE env variable.")
	exit = make(chan bool)
}

func runApp(cmd *Command, args []string) int {
	ShowShortVersionBanner()

	if len(args) == 0 || args[0] == "watchall" {
		currpath, _ = os.Getwd()

		if found, _gopath, _ := SearchGOPATHs(currpath); found {
			appname = path.Base(currpath)
			currentGoPath = _gopath
		} else {
			exitPrint(fmt.Sprintf("Bee does not support non Beego project: %s", currpath))
		}
		ColorLog("[INFO] Using '%s' as 'appname'\n", appname)
	} else {
		// Check if passed Bee application path/name exists in the GOPATH(s)
		if found, _gopath, _path := SearchGOPATHs(args[0]); found {
			currpath = _path
			currentGoPath = _gopath
			appname = path.Base(currpath)
		} else {
			panic(fmt.Sprintf("No Beego application '%s' found in your GOPATH", args[0]))
		}

		ColorLog("[INFO] Using '%s' as 'appname'\n", appname)

		if strings.HasSuffix(appname, ".go") && isExist(currpath) {
			ColorLog("[WARN] The appname is in conflict with currpath's file, do you want to build appname as %s\n", appname)
			ColorLog("[INFO] Do you want to overwrite it? [yes|no]]  ")
			if !askForConfirmation() {
				return 0
			}
		}
	}

	Debugf("current path:%s\n", currpath)

	if runmode == "prod" || runmode == "dev"{
		os.Setenv("BEEGO_RUNMODE", runmode)
		ColorLog("[INFO] Using '%s' as 'runmode'\n", os.Getenv("BEEGO_RUNMODE"))
	}else if runmode != ""{
		os.Setenv("BEEGO_RUNMODE", runmode)
		ColorLog("[WARN] Using '%s' as 'runmode'\n", os.Getenv("BEEGO_RUNMODE"))
	}else if os.Getenv("BEEGO_RUNMODE") != ""{
		ColorLog("[WARN] Using '%s' as 'runmode'\n", os.Getenv("BEEGO_RUNMODE"))
	}

	err := loadConfig()
	if err != nil {
		ColorLog("[ERRO] Fail to parse bee.json[ %s ]\n", err)
	}

	var paths []string
	readAppDirectories(currpath, &paths)

	// Because monitor files has some issues, we watch current directory
	// and ignore non-go files.
	for _, p := range conf.DirStruct.Others {
		paths = append(paths, strings.Replace(p, "$GOPATH", currentGoPath, -1))
	}

	files := []string{}
	for _, arg := range mainFiles {
		if len(arg) > 0 {
			files = append(files, arg)
		}
	}
	if downdoc == "true" {
		if _, err := os.Stat(path.Join(currpath, "swagger", "index.html")); err != nil {
			if os.IsNotExist(err) {
				downloadFromURL(swaggerlink, "swagger.zip")
				unzipAndDelete("swagger.zip")
			}
		}
	}
	if gendoc == "true" {
		NewWatcher(paths, files, true)
		Autobuild(files, true)
	} else {
		NewWatcher(paths, files, false)
		Autobuild(files, false)
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
		if strings.HasSuffix(fileInfo.Name(), "swagger") {
			continue
		}

		if !vendorWatch && strings.HasSuffix(fileInfo.Name(), "vendor") {
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

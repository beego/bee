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
	"io/ioutil"
	"os"
	path "path/filepath"
	"runtime"
	"strings"
)

var cmdRun = &Command{
	UsageLine: "run [appname] [watchall] [-main=*.go] [-downdoc=true]  [-gendoc=true] [-vendor=true] [-e=folderToExclude]  [-tags=goBuildTags] [-runmode=BEEGO_RUNMODE]",
	Short:     "Run the application by starting a local development server",
	Long: `
Run command will supervise the filesystem of the application for any changes, and recompile/restart it.

`,
	PreRun: func(cmd *Command, args []string) { ShowShortVersionBanner() },
	Run:    runApp,
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
	cmdRun.Flag.Var(&mainFiles, "main", "Specify main go files.")
	cmdRun.Flag.Var(&gendoc, "gendoc", "Enable auto-generate the docs.")
	cmdRun.Flag.Var(&downdoc, "downdoc", "Enable auto-download of the swagger file if it does not exist.")
	cmdRun.Flag.Var(&excludedPaths, "e", "List of paths to exclude.")
	cmdRun.Flag.BoolVar(&vendorWatch, "vendor", false, "Enable watch vendor folder.")
	cmdRun.Flag.StringVar(&buildTags, "tags", "", "Set the build tags. See: https://golang.org/pkg/go/build/")
	cmdRun.Flag.StringVar(&runmode, "runmode", "", "Set the Beego run mode.")
	exit = make(chan bool)
}

func runApp(cmd *Command, args []string) int {
	if len(args) == 0 || args[0] == "watchall" {
		currpath, _ = os.Getwd()

		if found, _gopath, _ := SearchGOPATHs(currpath); found {
			appname = path.Base(currpath)
			currentGoPath = _gopath
		} else {
			logger.Fatalf("No application '%s' found in your GOPATH", currpath)
		}
	} else {
		// Check if passed Bee application path/name exists in the GOPATH(s)
		if found, _gopath, _path := SearchGOPATHs(args[0]); found {
			currpath = _path
			currentGoPath = _gopath
			appname = path.Base(currpath)
		} else {
			logger.Fatalf("No application '%s' found in your GOPATH", args[0])
		}

		if strings.HasSuffix(appname, ".go") && isExist(currpath) {
			logger.Warnf("The appname is in conflict with file's current path. Do you want to build appname as '%s'", appname)
			logger.Info("Do you want to overwrite it? [yes|no] ")
			if !askForConfirmation() {
				return 0
			}
		}
	}

	logger.Infof("Using '%s' as 'appname'", appname)

	logger.Debugf("Current path: %s", __FILE__(), __LINE__(), currpath)

	if runmode == "prod" || runmode == "dev" {
		os.Setenv("BEEGO_RUNMODE", runmode)
		logger.Infof("Using '%s' as 'runmode'", os.Getenv("BEEGO_RUNMODE"))
	} else if runmode != "" {
		os.Setenv("BEEGO_RUNMODE", runmode)
		logger.Warnf("Using '%s' as 'runmode'", os.Getenv("BEEGO_RUNMODE"))
	} else if os.Getenv("BEEGO_RUNMODE") != "" {
		logger.Warnf("Using '%s' as 'runmode'", os.Getenv("BEEGO_RUNMODE"))
	}

	err := loadConfig()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %s", err)
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
		AutoBuild(files, true)
	} else {
		NewWatcher(paths, files, false)
		AutoBuild(files, false)
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
			logger.Errorf("Cannot get absolute path of '%s'", p)
			continue
		}
		absFilePath, err := path.Abs(filePath)
		if err != nil {
			logger.Errorf("Cannot get absolute path of '%s'", filePath)
			break
		}
		if strings.HasPrefix(absFilePath, absP) {
			logger.Infof("'%s' is not being watched", filePath)
			return true
		}
	}
	return false
}

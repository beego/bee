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
	"io/ioutil"
	"os"
	path "path/filepath"
	"runtime"
	"strings"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	"github.com/beego/bee/config"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
)

var CmdRun = &commands.Command{
	UsageLine: "run [appname] [watchall] [-main=*.go] [-downdoc=true]  [-gendoc=true] [-vendor=true] [-e=folderToExclude] [-ex=extraPackageToWatch] [-tags=goBuildTags] [-runmode=BEEGO_RUNMODE]",
	Short:     "Run the application by starting a local development server",
	Long: `
Run command will supervise the filesystem of the application for any changes, and recompile/restart it.

`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    RunApp,
}

var (
	mainFiles utils.ListOpts
	downdoc   utils.DocValue
	gendoc    utils.DocValue
	// The flags list of the paths excluded from watching
	excludedPaths utils.StrFlags
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
	// Extra directories
	extraPackages utils.StrFlags
)
var started = make(chan bool)

func init() {
	CmdRun.Flag.Var(&mainFiles, "main", "Specify main go files.")
	CmdRun.Flag.Var(&gendoc, "gendoc", "Enable auto-generate the docs.")
	CmdRun.Flag.Var(&downdoc, "downdoc", "Enable auto-download of the swagger file if it does not exist.")
	CmdRun.Flag.Var(&excludedPaths, "e", "List of paths to exclude.")
	CmdRun.Flag.BoolVar(&vendorWatch, "vendor", false, "Enable watch vendor folder.")
	CmdRun.Flag.StringVar(&buildTags, "tags", "", "Set the build tags. See: https://golang.org/pkg/go/build/")
	CmdRun.Flag.StringVar(&runmode, "runmode", "", "Set the Beego run mode.")
	CmdRun.Flag.Var(&extraPackages, "ex", "List of extra package to watch.")
	exit = make(chan bool)
	commands.AvailableCommands = append(commands.AvailableCommands, CmdRun)
}

func RunApp(cmd *commands.Command, args []string) int {
	if len(args) == 0 || args[0] == "watchall" {
		currpath, _ = os.Getwd()
		if found, _gopath, _ := utils.SearchGOPATHs(currpath); found {
			appname = path.Base(currpath)
			currentGoPath = _gopath
		} else {
			beeLogger.Log.Fatalf("No application '%s' found in your GOPATH", currpath)
		}
	} else {
		// Check if passed Bee application path/name exists in the GOPATH(s)
		if found, _gopath, _path := utils.SearchGOPATHs(args[0]); found {
			currpath = _path
			currentGoPath = _gopath
			appname = path.Base(currpath)
		} else {
			beeLogger.Log.Fatalf("No application '%s' found in your GOPATH", args[0])
		}

		if strings.HasSuffix(appname, ".go") && utils.IsExist(currpath) {
			beeLogger.Log.Warnf("The appname is in conflict with file's current path. Do you want to build appname as '%s'", appname)
			beeLogger.Log.Info("Do you want to overwrite it? [yes|no] ")
			if !utils.AskForConfirmation() {
				return 0
			}
		}
	}

	beeLogger.Log.Infof("Using '%s' as 'appname'", appname)

	beeLogger.Log.Debugf("Current path: %s", utils.FILE(), utils.LINE(), currpath)

	if runmode == "prod" || runmode == "dev" {
		os.Setenv("BEEGO_RUNMODE", runmode)
		beeLogger.Log.Infof("Using '%s' as 'runmode'", os.Getenv("BEEGO_RUNMODE"))
	} else if runmode != "" {
		os.Setenv("BEEGO_RUNMODE", runmode)
		beeLogger.Log.Warnf("Using '%s' as 'runmode'", os.Getenv("BEEGO_RUNMODE"))
	} else if os.Getenv("BEEGO_RUNMODE") != "" {
		beeLogger.Log.Warnf("Using '%s' as 'runmode'", os.Getenv("BEEGO_RUNMODE"))
	}

	var paths []string
	readAppDirectories(currpath, &paths)

	// Because monitor files has some issues, we watch current directory
	// and ignore non-go files.
	for _, p := range config.Conf.DirStruct.Others {
		paths = append(paths, strings.Replace(p, "$GOPATH", currentGoPath, -1))
	}

	if len(extraPackages) > 0 {
		// get the full path
		for _, packagePath := range extraPackages {
			if found, _, _fullPath := utils.SearchGOPATHs(packagePath); found {
				readAppDirectories(_fullPath, &paths)
			} else {
				beeLogger.Log.Warnf("No extra package '%s' found in your GOPATH", packagePath)
			}
		}
		// let paths unique
		strSet := make(map[string]struct{})
		for _, p := range paths {
			strSet[p] = struct{}{}
		}
		paths = make([]string, len(strSet))
		index := 0
		for i := range strSet {
			paths[index] = i
			index++
		}
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

	// Start the Reload server (if enabled)
	if config.Conf.EnableReload {
		startReloadServer()
	}
	if gendoc == "true" {
		NewWatcher(paths, files, true)
		AutoBuild(files, true)
	} else {
		NewWatcher(paths, files, false)
		AutoBuild(files, false)
	}

	for {
		<-exit
		runtime.Goexit()
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

		if fileInfo.IsDir() && fileInfo.Name()[0] != '.' {
			readAppDirectories(directory+"/"+fileInfo.Name(), paths)
			continue
		}

		if useDirectory {
			continue
		}

		if path.Ext(fileInfo.Name()) == ".go" || (ifStaticFile(fileInfo.Name()) && config.Conf.EnableReload) {
			*paths = append(*paths, directory)
			useDirectory = true
		}
	}
}

// If a file is excluded
func isExcluded(filePath string) bool {
	for _, p := range excludedPaths {
		absP, err := path.Abs(p)
		if err != nil {
			beeLogger.Log.Errorf("Cannot get absolute path of '%s'", p)
			continue
		}
		absFilePath, err := path.Abs(filePath)
		if err != nil {
			beeLogger.Log.Errorf("Cannot get absolute path of '%s'", filePath)
			break
		}
		if strings.HasPrefix(absFilePath, absP) {
			beeLogger.Log.Infof("'%s' is not being watched", filePath)
			return true
		}
	}
	return false
}

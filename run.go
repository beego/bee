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
	"encoding/json"
	"os"
	path "path/filepath"
	"runtime"

	"github.com/Unknwon/com"
)

var cmdRun = &Command{
	UsageLine: "run [appname]",
	Short:     "run the app which can hot compile",
	Long: `
start the appname throw exec.Command

then start a inotify watch for current dir
										
when the file has changed bee will auto go build and restart the app

	file changed
	     |
  check if it's go file
	     |
     yes     no
      |       |
 go build    do nothing
     |
 restart app   
`,
}

var defaultJson = `
{
	"go_install": false,
	"dir_structure":{
		"controllers": "",
		"models": "",
		"others": []
	},
	"main_files":{
		"main.go": "",
		"others": []
	}
}
`

func init() {
	cmdRun.Run = runApp
}

var appname string
var conf struct {
	// Indicates whether execute "go install" before "go build".
	GoInstall bool     `json:"go_install"`
	WatchExt  []string `json:"watch_ext"`
	DirStruct struct {
		Controllers string
		Models      string
		Others      []string // Other directories.
	} `json:"dir_structure"`

	Bale struct {
		Import string
		Dirs   []string
		IngExt []string `json:"ignore_ext"`
	}
}

func runApp(cmd *Command, args []string) {
	exit := make(chan bool)
	crupath, _ := os.Getwd()
	if len(args) != 1 {
		appname = path.Base(crupath)
		com.ColorLog("[INFO] Uses '%s' as 'appname'\n", appname)
	} else {
		appname = args[0]
	}
	Debugf("current path:%s\n", crupath)

	err := loadConfig()
	if err != nil {
		com.ColorLog("[ERRO] Fail to parse bee.json[ %s ]\n", err)
	}
	var paths []string
	paths = append(paths,
		path.Join(crupath, conf.DirStruct.Controllers),
		path.Join(crupath, conf.DirStruct.Models),
		path.Join(crupath, "./")) // Current path.
	// Because monitor files has some issues, we watch current directory
	// and ignore non-go files.
	paths = append(paths, conf.DirStruct.Others...)

	NewWatcher(paths)
	Autobuild()
	for {
		select {
		case <-exit:
			runtime.Goexit()
		}
	}
}

// loadConfig loads customized configuration.
func loadConfig() error {
	f, err := os.Open("bee.json")
	if err != nil {
		// Use default.
		err = json.Unmarshal([]byte(defaultJson), &conf)
		if err != nil {
			return err
		}
	} else {
		defer f.Close()
		com.ColorLog("[INFO] Detected bee.json\n")
		d := json.NewDecoder(f)
		err = d.Decode(&conf)
		if err != nil {
			return err
		}
	}
	// Set variables.
	if len(conf.DirStruct.Controllers) == 0 {
		conf.DirStruct.Controllers = "controllers"
	}
	if len(conf.DirStruct.Models) == 0 {
		conf.DirStruct.Models = "models"
	}

	// Append watch exts.
	watchExts = append(watchExts, conf.WatchExt...)
	return nil
}

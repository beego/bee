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
package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	beeLogger "github.com/beego/bee/logger"
	"gopkg.in/yaml.v2"
)

const confVer = 0

var Conf = struct {
	Version            int
	WatchExts          []string  `json:"watch_ext" yaml:"watch_ext"`
	WatchExtsStatic    []string  `json:"watch_ext_static" yaml:"watch_ext_static"`
	GoInstall          bool      `json:"go_install" yaml:"go_install"` // Indicates whether execute "go install" before "go build".
	DirStruct          dirStruct `json:"dir_structure" yaml:"dir_structure"`
	CmdArgs            []string  `json:"cmd_args" yaml:"cmd_args"`
	Envs               []string
	Bale               bale
	Database           database
	EnableReload       bool              `json:"enable_reload" yaml:"enable_reload"`
	EnableNotification bool              `json:"enable_notification" yaml:"enable_notification"`
	Scripts            map[string]string `json:"scripts" yaml:"scripts"`
}{
	WatchExts:       []string{".go"},
	WatchExtsStatic: []string{".html", ".tpl", ".js", ".css"},
	GoInstall:       true,
	DirStruct: dirStruct{
		Others: []string{},
	},
	CmdArgs: []string{},
	Envs:    []string{},
	Bale: bale{
		Dirs:   []string{},
		IngExt: []string{},
	},
	Database: database{
		Driver: "mysql",
	},
	EnableNotification: true,
	Scripts:            map[string]string{},
}

// dirStruct describes the application's directory structure
type dirStruct struct {
	WatchAll    bool `json:"watch_all" yaml:"watch_all"`
	Controllers string
	Models      string
	Others      []string // Other directories
}

// bale
type bale struct {
	Import string
	Dirs   []string
	IngExt []string `json:"ignore_ext" yaml:"ignore_ext"`
}

// database holds the database connection information
type database struct {
	Driver string
	Conn   string
	Dir    string
}

// LoadConfig loads the bee tool configuration.
// It looks for Beefile or bee.json in the current path,
// and falls back to default configuration in case not found.
func LoadConfig() {
	currentPath, err := os.Getwd()
	if err != nil {
		beeLogger.Log.Error(err.Error())
	}

	dir, err := os.Open(currentPath)
	if err != nil {
		beeLogger.Log.Error(err.Error())
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		beeLogger.Log.Error(err.Error())
	}

	for _, file := range files {
		switch file.Name() {
		case "bee.json":
			{
				err = parseJSON(filepath.Join(currentPath, file.Name()), &Conf)
				if err != nil {
					beeLogger.Log.Errorf("Failed to parse JSON file: %s", err)
				}
				break
			}
		case "Beefile":
			{
				err = parseYAML(filepath.Join(currentPath, file.Name()), &Conf)
				if err != nil {
					beeLogger.Log.Errorf("Failed to parse YAML file: %s", err)
				}
				break
			}
		}
	}

	// Check format version
	if Conf.Version != confVer {
		beeLogger.Log.Warn("Your configuration file is outdated. Please do consider updating it.")
		beeLogger.Log.Hint("Check the latest version of bee's configuration file.")
	}

	// Set variables
	if len(Conf.DirStruct.Controllers) == 0 {
		Conf.DirStruct.Controllers = "controllers"
	}

	if len(Conf.DirStruct.Models) == 0 {
		Conf.DirStruct.Models = "models"
	}
}

func parseJSON(path string, v interface{}) error {
	var (
		data []byte
		err  error
	)
	data, err = ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, v)
	return err
}

func parseYAML(path string, v interface{}) error {
	var (
		data []byte
		err  error
	)
	data, err = ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, v)
	return err
}

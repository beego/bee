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

	"io"
	"path/filepath"

	beeLogger "github.com/beego/bee/logger"
	"gopkg.in/yaml.v2"
)

const confVer = 0

var defaultConf = `{
	"version": 0,
	"gopm": {
		"enable": false,
		"install": false
	},
	"go_install": true,
	"watch_ext": [],
	"dir_structure": {
		"watch_all": false,
		"controllers": "",
		"models": "",
		"others": []
	},
	"cmd_args": [],
	"envs": [],
	"database": {
		"driver": "mysql"
	},
	"enable_reload": false
}
`
var Conf struct {
	Version int
	// gopm support
	Gopm struct {
		Enable  bool
		Install bool
	}
	// Indicates whether execute "go install" before "go build".
	GoInstall bool     `json:"go_install" yaml:"go_install"`
	WatchExt  []string `json:"watch_ext" yaml:"watch_ext"`
	DirStruct struct {
		WatchAll    bool `json:"watch_all" yaml:"watch_all"`
		Controllers string
		Models      string
		Others      []string // Other directories.
	} `json:"dir_structure" yaml:"dir_structure"`
	CmdArgs []string `json:"cmd_args" yaml:"cmd_args"`
	Envs    []string
	Bale    struct {
		Import string
		Dirs   []string
		IngExt []string `json:"ignore_ext" yaml:"ignore_ext"`
	}
	Database struct {
		Driver string
		Conn   string
	}
	EnableReload bool `json:"enable_reload" yaml:"enable_reload"`
}

func init() {
	loadConfig()
}

// loadConfig loads customized configuration.
func loadConfig() {
	beeLogger.Log.Info("Loading default configuration...")
	err := json.Unmarshal([]byte(defaultConf), &Conf)
	if err != nil {
		beeLogger.Log.Errorf(err.Error())
	}
	err = filepath.Walk(".", func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if fileInfo.IsDir() {
			return nil
		}

		if fileInfo.Name() == "bee.json" {
			beeLogger.Log.Info("Loading configuration from 'bee.json'...")
			err = parseJSON(path, &Conf)
			if err != nil {
				beeLogger.Log.Errorf("Failed to parse JSON file: %s", err)
				return err
			}
			return io.EOF
		}

		if fileInfo.Name() == "Beefile" {
			beeLogger.Log.Info("Loading configuration from 'Beefile'...")
			err = parseYAML(path, &Conf)
			if err != nil {
				beeLogger.Log.Errorf("Failed to parse YAML file: %s", err)
				return err
			}
			return io.EOF
		}
		return nil
	})
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

	// Append watch exts
	//watchExts = append(watchExts, Conf.WatchExt...)
	return
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

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
	"io/ioutil"
	"os"

	"io"
	"path/filepath"

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
	}
}
`
var conf struct {
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
}

// loadConfig loads customized configuration.
func loadConfig() (err error) {
	err = filepath.Walk(".", func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if fileInfo.IsDir() {
			return nil
		}

		if fileInfo.Name() == "bee.json" {
			logger.Info("Loading configuration from 'bee.json'...")
			err = parseJSON(path, conf)
			if err != nil {
				logger.Errorf("Failed to parse JSON file: %s", err)
				return err
			}
			return io.EOF
		}

		if fileInfo.Name() == "Beefile" {
			logger.Info("Loading configuration from 'Beefile'...")
			err = parseYAML(path, conf)
			if err != nil {
				logger.Errorf("Failed to parse YAML file: %s", err)
				return err
			}
			return io.EOF
		}
		return nil
	})

	// In case no configuration file found or an error different than io.EOF,
	// fallback to default configuration
	if err != io.EOF {
		logger.Info("Loading default configuration...")
		err = json.Unmarshal([]byte(defaultConf), &conf)
		if err != nil {
			return
		}
	}

	// No need to return io.EOF error
	err = nil

	// Check format version
	if conf.Version != confVer {
		logger.Warn("Your configuration file is outdated. Please do consider updating it.")
		logger.Hint("Check the latest version of bee's configuration file.")
	}

	// Set variables
	if len(conf.DirStruct.Controllers) == 0 {
		conf.DirStruct.Controllers = "controllers"
	}
	if len(conf.DirStruct.Models) == 0 {
		conf.DirStruct.Models = "models"
	}

	// Append watch exts
	watchExts = append(watchExts, conf.WatchExt...)
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
	err = json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	return nil
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
	err = yaml.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	return nil
}

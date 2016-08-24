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

	"gopkg.in/yaml.v2"
)

const ConfVer = 0

var defaultConf = `{
	"version": 0,
	"gopm": {
		"enable": false,
		"install": false
	},
	"go_install": false,
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
func loadConfig() error {
	foundConf := false
	f, err := os.Open("bee.json")
	if err == nil {
		defer f.Close()
		ColorLog("[INFO] Detected bee.json\n")
		d := json.NewDecoder(f)
		err = d.Decode(&conf)
		if err != nil {
			return err
		}
		foundConf = true
	}
	byml, erryml := ioutil.ReadFile("Beefile")
	if erryml == nil {
		ColorLog("[INFO] Detected Beefile\n")
		err = yaml.Unmarshal(byml, &conf)
		if err != nil {
			return err
		}
		foundConf = true
	}
	if !foundConf {
		// Use default.
		err = json.Unmarshal([]byte(defaultConf), &conf)
		if err != nil {
			return err
		}
	}
	// Check format version.
	if conf.Version != ConfVer {
		ColorLog("[WARN] Your bee.json is out-of-date, please update!\n")
		ColorLog("[HINT] Compare bee.json under bee source code path and yours\n")
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

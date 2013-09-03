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
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Unknwon/com"
)

var cmdBale = &Command{
	UsageLine: "bale",
	Short:     "packs non-Go files to Go source files",
	Long: `
bale packs non-Go files to Go source files and

auto-generate unpack function to main package then run it

during the runtime.

This is mainly used for zealots who are requiring 100% Go code.`,
}

func init() {
	cmdBale.Run = runBale
}

func runBale(cmd *Command, args []string) {
	err := loadConfig()
	if err != nil {
		com.ColorLog("[ERRO] Fail to parse bee.json[ %s ]\n", err)
	}

	os.Mkdir("bale", os.ModePerm)

	for _, p := range conf.Bale.Dirs {
		filepath.Walk(p, walkFn)
	}
}

func walkFn(resPath string, info os.FileInfo, err error) error {
	if info.IsDir() || filterSuffix(resPath) {
		return nil
	}

	resPath = strings.Replace(resPath, "_", "__", -1)
	resPath = strings.Replace(resPath, ".", "___", -1)
	sep := "/"
	if runtime.GOOS == "windows" {
		sep = "\\"
	}
	resPath = strings.Replace(resPath, sep, "_", -1)
	os.MkdirAll(path.Dir(resPath), os.ModePerm)
	os.Create("bale/" + resPath + ".go")
	return nil
}

func filterSuffix(name string) bool {
	for _, s := range conf.Bale.IngExt {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

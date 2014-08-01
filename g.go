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

import "os"

var cmdGenerate = &Command{
	UsageLine: "generate [Command]",
	Short:     "generate code based on application",
	Long: `
bee generate model [modelfile] [dbconfig]
    generate model base on struct
bee generate controller [modelfile]
    generate RESTFul controllers based on modelfile             
bee generate router [controllerfile]
    generate router based on controllerfile
bee generate docs
    generate swagger doc file
bee generate test [routerfile]
    generate testcase
`,
}

func generateCode(cmd *Command, args []string) {
	curpath, _ := os.Getwd()
	if len(args) < 1 {
		ColorLog("[ERRO] command is missing\n")
		os.Exit(2)
	}

	gopath := os.Getenv("GOPATH")
	Debugf("gopath:%s", gopath)
	if gopath == "" {
		ColorLog("[ERRO] $GOPATH not found\n")
		ColorLog("[HINT] Set $GOPATH in your environment vairables\n")
		os.Exit(2)
	}

	gcmd := args[0]
	switch gcmd {
	case "docs":
		generateDocs(curpath)
	case "model":
		generateModel("mysql", "root@tcp(127.0.0.1:3306)/sgfas?charset=utf8", curpath)
	case "controller":
		generateController("mysql", "", curpath)
	case "router":
		generateRouter("mysql", "", curpath)
	default:
		ColorLog("[ERRO] command is missing\n")
	}
	ColorLog("[SUCC] generate successfully created!\n")
}

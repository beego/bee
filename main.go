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
	"flag"
	"log"
	"os"

	"github.com/beego/bee/cmd"
	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/config"
	"github.com/beego/bee/generate/swaggergen"
	"github.com/beego/bee/utils"
)

func main() {
	currentpath, _ := os.Getwd()

	flag.Usage = cmd.Usage
	flag.Parse()
	log.SetFlags(0)

	args := flag.Args()

	if len(args) < 1 {
		cmd.Usage()
		os.Exit(2)
		return
	}

	if args[0] == "help" {
		cmd.Help(args[1:])
		return
	}

	for _, c := range commands.AvailableCommands {
		if c.Name() == args[0] && c.Run != nil {
			c.Flag.Usage = func() { c.Usage() }
			if c.CustomFlags {
				args = args[1:]
			} else {
				c.Flag.Parse(args[1:])
				args = c.Flag.Args()
			}

			if c.PreRun != nil {
				c.PreRun(c, args)
			}

			config.LoadConfig()

			// Check if current directory is inside the GOPATH,
			// if so parse the packages inside it.
			if utils.IsInGOPATH(currentpath) && cmd.IfGenerateDocs(c.Name(), args) {
				swaggergen.ParsePackagesFromDir(currentpath)
			}

			os.Exit(c.Run(c, args))
			return
		}
	}

	utils.PrintErrorAndExit("Unknown subcommand", cmd.ErrorTemplate)
}

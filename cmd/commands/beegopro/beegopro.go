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
package beegopro

import (
	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/internal/app/module/beegopro"
	"github.com/beego/bee/logger"
	"strings"
)

var CmdBeegoPro = &commands.Command{
	UsageLine: "pro [command]",
	Short:     "Source code generator",
	Long:      ``,
	Run:       BeegoPro,
}

func init() {
	CmdBeegoPro.Flag.Var(&beegopro.SQL, "sql", "sql file path")
	CmdBeegoPro.Flag.Var(&beegopro.SQLMode, "sqlmode", "sql mode")
	CmdBeegoPro.Flag.Var(&beegopro.SQLModePath, "sqlpath", "sql mode path")
	commands.AvailableCommands = append(commands.AvailableCommands, CmdBeegoPro)
}

func BeegoPro(cmd *commands.Command, args []string) int {
	if len(args) < 1 {
		beeLogger.Log.Fatal("Command is missing")
	}

	if len(args) >= 2 {
		cmd.Flag.Parse(args[1:])
	}

	gcmd := args[0]
	switch gcmd {
	case "gen":
		beegopro.DefaultBeegoPro.Run()
	case "config":
		beegopro.DefaultBeegoPro.GenConfig()
	case "migration":
		beegopro.DefaultBeegoPro.Migration(args)
	default:
		beeLogger.Log.Fatal("Command is missing")
	}
	beeLogger.Log.Successf("%s successfully generated!", strings.Title(gcmd))
	return 0
}

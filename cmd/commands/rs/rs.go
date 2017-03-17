// Copyright 2017 bee authors
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
package rs

import (
	"os/exec"

	"os"

	"runtime"

	"fmt"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	"github.com/beego/bee/config"
	"github.com/beego/bee/logger"
	"github.com/beego/bee/logger/colors"
)

type customCommand struct {
	Name    string
	Command string
}

var description = `Run script allows you to run arbitrary commands using Bee.
  Custom commands are provided from the "scripts" object inside bee.json or Beefile.
`

func init() {
	CmdNew := &commands.Command{
		UsageLine: "rs",
		Short:     "Run customized scripts",
		PreRun:    func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
		Run:       RunScript,
	}
	config.LoadConfig()
	CmdNew.Long = description
	for commandName, command := range config.Conf.Scripts {
		CmdNew.Long += fmt.Sprintf("  ▶ $ bee %s %s ==> %s\n", CmdNew.UsageLine, commandName, command)
	}
	commands.AvailableCommands = append(commands.AvailableCommands, CmdNew)
}

func RunScript(cmd *commands.Command, args []string) int {
	if len(args) == 0 {
		cmd.Usage()
		return 0
	}
	for _, arg := range args {
		if c, exist := config.Conf.Scripts[arg]; exist {
			command := customCommand{
				Name:    arg,
				Command: c,
			}
			if err := command.run(); err != nil {
				beeLogger.Log.Error(err.Error())
			}
		} else {
			beeLogger.Log.Errorf("Command %s not found in Beefile", arg)
		}
	}
	return 0
}

func (c *customCommand) run() error {
	beeLogger.Log.Info(colors.GreenBold(fmt.Sprintf("Running '%s':", c.Name)))
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd = exec.Command("sh", "-c", c.Command)
	case "windows": //TODO
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

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

// Package rs ...
package rs

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"strings"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	"github.com/beego/bee/config"
	"github.com/beego/bee/logger"
	"github.com/beego/bee/logger/colors"
	"github.com/beego/bee/utils"
)

var cmdRs = &commands.Command{
	UsageLine: "rs",
	Short:     "Run customized scripts",
	Long: `Run script allows you to run arbitrary commands using Bee.
  Custom commands are provided from the "scripts" object inside bee.json or Beefile.

  To run a custom command, use: {{"$ bee rs mycmd ARGS" | bold}}
  {{if len .}}
{{"AVAILABLE SCRIPTS"|headline}}{{range $cmdName, $cmd := .}}
  {{$cmdName | bold}}
      {{$cmd}}{{end}}{{end}}
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    runScript,
}

func init() {
	config.LoadConfig()
	cmdRs.Long = utils.TmplToString(cmdRs.Long, config.Conf.Scripts)
	commands.AvailableCommands = append(commands.AvailableCommands, cmdRs)
}

func runScript(cmd *commands.Command, args []string) int {
	if len(args) == 0 {
		cmd.Usage()
	}

	start := time.Now()
	script, args := args[0], args[1:]

	if c, exist := config.Conf.Scripts[script]; exist {
		command := customCommand{
			Name:    script,
			Command: c,
			Args:    args,
		}
		if err := command.run(); err != nil {
			beeLogger.Log.Error(err.Error())
		}
	} else {
		beeLogger.Log.Errorf("Command '%s' not found in Beefile/bee.json", script)
	}
	elapsed := time.Since(start)
	fmt.Println(colors.GreenBold(fmt.Sprintf("Finished in %s.", elapsed)))
	return 0
}

type customCommand struct {
	Name    string
	Command string
	Args    []string
}

func (c *customCommand) run() error {
	beeLogger.Log.Info(colors.GreenBold(fmt.Sprintf("Running '%s'...", c.Name)))
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		args := append([]string{c.Command}, c.Args...)
		cmd = exec.Command("sh", "-c", strings.Join(args, " "))
	case "windows":
		args := append([]string{c.Command}, c.Args...)
		cmd = exec.Command("cmd", "/C", strings.Join(args, " "))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

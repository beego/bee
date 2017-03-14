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
package run

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/logger/colors"
)

var cmdRunScript = &commands.Command{
	UsageLine: "run-script",
	Short:     "Run customized scripts",
	Long: `run-script allows you to run arbitrary commands using Bee.
  Custom commands are provided from the "scripts" object inside bee.json or Beefile.
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    runScript,
}

var (
	scripts map[string]string
)

func init() {
	commands.AvailableCommands = append(commands.AvailableCommands, cmdRunScript)
	scripts = make(map[string]string)
	scripts["test"] = "go test -v github.com/astaxie/beego"
	scripts["ls"] = "ls -all"
}

func runScript(cmd *commands.Command, args []string) int {
	if len(args) == 0 {
		cmd.Usage()
	}

	start := time.Now()
	for _, arg := range os.Args[2:] {
		script := scriptCmd{
			Name:    arg,
			Command: scripts[arg],
		}
		if err := script.run(); err != nil {
			beeLogger.Log.Error(err.Error())
		}
	}
	elapsed := time.Since(start)

	fmt.Println(colors.CyanBold(fmt.Sprintf("Finished in %s.", elapsed)))
	return 0
}

type scriptCmd struct {
	Name    string
	Command string
}

func (sc *scriptCmd) run() error {
	beeLogger.Log.Info(colors.CyanBold(fmt.Sprintf("Running '%s'...", sc.Name)))

	if len(sc.Command) == 0 {
		return fmt.Errorf(fmt.Sprintf("No script to run for command '%s'", sc.Name))
	}

	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		c = exec.Command("sh", "-c", sc.Command)
	case "windows":
		// Not supported yet
	}
	out, err := c.Output()
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

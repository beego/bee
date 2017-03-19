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

// Package dlv ...
package dlv

import (
	"flag"
	"fmt"
	"os/exec"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	beeLogger "github.com/beego/bee/logger"
	"github.com/derekparker/delve/pkg/terminal"
	"github.com/derekparker/delve/service"
	"github.com/derekparker/delve/service/rpc2"

	dlvConfig "github.com/derekparker/delve/pkg/config"
)

var cmdDlv = &commands.Command{
	CustomFlags: true,
	UsageLine:   "dlv [-package=\"\"] [-port=8181]",
	Short:       "Start a debugging session using Delve",
	Long: `dlv command start a debugging session using debugging tool Delve.

  To debug your application using Delve, use: {{"$ bee dlv" | bold}}

  For more information on Delve: https://github.com/derekparker/delve
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    runDlv,
}

var (
	packageName string
	port        string
)

func init() {
	fs := flag.NewFlagSet("dlv", flag.ContinueOnError)
	fs.StringVar(&port, "port", "8181", "Port to listen to for clients")
	fs.StringVar(&packageName, "package", "", "The package to debug (Must have a main package)")
	cmdDlv.Flag = *fs
	commands.AvailableCommands = append(commands.AvailableCommands, cmdDlv)
}

func runDlv(cmd *commands.Command, args []string) int {
	if err := cmd.Flag.Parse(args); err != nil {
		beeLogger.Log.Fatalf("Error parsing flags: %v", err.Error())
	}
	addr := fmt.Sprintf("127.0.0.1:%s", port)

	startChan := make(chan bool)
	defer close(startChan)

	go runDelve(addr, startChan)

	if started := <-startChan; started {
		beeLogger.Log.Info("Starting Delve Debugger...")
		status, err := startRepl(addr)
		if err != nil {
			beeLogger.Log.Fatal(err.Error())
		}
		return status
	}
	return 0
}

// runDelve runs the Delve debugger in API mode
func runDelve(addr string, c chan bool) {
	args := []string{
		"debug",
		"--headless",
		"--accept-multiclient=true",
		"--api-version=2",
		fmt.Sprintf("--listen=%s", addr),
	}
	if err := exec.Command("dlv", args...).Start(); err == nil {
		c <- true
	}
}

// startRepl starts the Delve REPL
func startRepl(addr string) (int, error) {
	var client service.Client
	client = rpc2.NewClient(addr)
	term := terminal.New(client, dlvConfig.LoadConfig())

	status, err := term.Run()
	if err != nil {
		return status, err
	}
	defer term.Close()

	return 0, nil
}

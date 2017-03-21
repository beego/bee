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
	"net"
	"os"
	"path/filepath"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
	"github.com/derekparker/delve/pkg/terminal"
	"github.com/derekparker/delve/service"
	"github.com/derekparker/delve/service/rpc2"
	"github.com/derekparker/delve/service/rpccommon"
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
	port        int
)

func init() {
	fs := flag.NewFlagSet("dlv", flag.ContinueOnError)
	fs.IntVar(&port, "port", 8181, "Port to listen to for clients")
	fs.StringVar(&packageName, "package", "", "The package to debug (Must have a main package)")
	cmdDlv.Flag = *fs
	commands.AvailableCommands = append(commands.AvailableCommands, cmdDlv)
}

func runDlv(cmd *commands.Command, args []string) int {
	if err := cmd.Flag.Parse(args); err != nil {
		beeLogger.Log.Fatalf("Error parsing flags: %v", err.Error())
	}

	debugname := "debug"
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return runDelve(addr, debugname)
}

// runDelve runs the Delve debugger server
func runDelve(addr, debugname string) int {
	beeLogger.Log.Info("Starting Delve Debugger...")

	err := utils.GoBuild(debugname, packageName)
	if err != nil {
		beeLogger.Log.Fatalf("%v", err)
	}

	fp, err := filepath.Abs("./" + debugname)
	if err != nil {
		beeLogger.Log.Fatalf("%v", err)
	}
	defer os.Remove(fp)

	abs, err := filepath.Abs(debugname)
	if err != nil {
		beeLogger.Log.Fatalf("%v", err)
	}

	//
	// Create and start the debugger server
	//
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		beeLogger.Log.Fatalf("Could not start listener: %s", err)
	}
	defer listener.Close()

	server := rpccommon.NewServer(&service.Config{
		Listener:    listener,
		AcceptMulti: true,
		AttachPid:   0,
		APIVersion:  2,
		WorkingDir:  "./",
		ProcessArgs: []string{abs},
	}, false)
	if err := server.Run(); err != nil {
		beeLogger.Log.Fatalf("Could not start debugger server: %v", err)
	}

	//
	// Start the Delve client REPL
	//
	client := rpc2.NewClient(addr)
	term := terminal.New(client, nil)

	status, err := term.Run()
	if err != nil {
		beeLogger.Log.Fatalf("Could not start Delve REPL: %v", err)
	}
	defer term.Close()

	// Stop and kill the debugger server once
	// user quits the REPL
	if err := server.Stop(true); err != nil {
		beeLogger.Log.Fatalf("Could not stop Delve server: %v", err)
	}
	return status
}

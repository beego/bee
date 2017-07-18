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

package apiapp

import (
	"net/http"

	beeLogger "github.com/beego/bee/logger"

	"os"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	"github.com/beego/bee/utils"
)

var CmdServer = &commands.Command{
	// CustomFlags: true,
	UsageLine: "server [port]",
	Short:     "serving static content over HTTP on port",
	Long: `
  The command 'server' creates a Beego API application.
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    createAPI,
}

var (
	a utils.DocValue
	p utils.DocValue
	f utils.DocValue
)

func init() {
	CmdServer.Flag.Var(&a, "a", "Listen address")
	CmdServer.Flag.Var(&p, "p", "Listen port")
	CmdServer.Flag.Var(&f, "f", "Static files fold")
	commands.AvailableCommands = append(commands.AvailableCommands, CmdServer)
}

func createAPI(cmd *commands.Command, args []string) int {
	if len(args) > 0 {
		err := cmd.Flag.Parse(args[1:])
		if err != nil {
			beeLogger.Log.Error(err.Error())
		}
	}
	if a == "" {
		a = "127.0.0.1"
	}
	if p == "" {
		p = "8080"
	}
	if f == "" {
		cwd, _ := os.Getwd()
		f = utils.DocValue(cwd)
	}
	beeLogger.Log.Infof("Start server on http://%s:%s, static file %s", a, p, f)
	err := http.ListenAndServe(string(a)+":"+string(p), http.FileServer(http.Dir(f)))
	if err != nil {
		beeLogger.Log.Error(err.Error())
	}
	return 0
}

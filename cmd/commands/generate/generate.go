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
package generate

import (
	"os"
	"strings"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	"github.com/beego/bee/config"
	"github.com/beego/bee/generate"
	"github.com/beego/bee/generate/swaggergen"
	"github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
)

var CmdGenerate = &commands.Command{
	UsageLine: "generate [command]",
	Short:     "Source code generator",
	Long: `▶ {{"To scaffold out your entire application:"|bold}}

     $ bee generate scaffold [scaffoldname] [-fields="title:string,body:text"] [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"]

  ▶ {{"To generate a Model based on fields:"|bold}}

     $ bee generate model [modelname] [-fields="name:type"]

  ▶ {{"To generate a controller:"|bold}}

     $ bee generate controller [controllerfile]

  ▶ {{"To generate a CRUD view:"|bold}}

     $ bee generate view [viewpath]

  ▶ {{"To generate a migration file for making database schema updates:"|bold}}

     $ bee generate migration [migrationfile] [-fields="name:type"]

  ▶ {{"To generate swagger doc file:"|bold}}

     $ bee generate docs

  ▶ {{"To generate a test case:"|bold}}

     $ bee generate test [routerfile]

  ▶ {{"To generate appcode based on an existing database:"|bold}}

     $ bee generate appcode [-tables=""] [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"] [-level=3]
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    GenerateCode,
}

func init() {
	CmdGenerate.Flag.Var(&generate.Tables, "tables", "List of table names separated by a comma.")
	CmdGenerate.Flag.Var(&generate.SQLDriver, "driver", "Database SQLDriver. Either mysql, postgres or sqlite.")
	CmdGenerate.Flag.Var(&generate.SQLConn, "conn", "Connection string used by the SQLDriver to connect to a database instance.")
	CmdGenerate.Flag.Var(&generate.Level, "level", "Either 1, 2 or 3. i.e. 1=models; 2=models and controllers; 3=models, controllers and routers.")
	CmdGenerate.Flag.Var(&generate.Fields, "fields", "List of table Fields.")
	CmdGenerate.Flag.Var(&generate.DDL, "ddl", "Generate DDL Migration")
	commands.AvailableCommands = append(commands.AvailableCommands, CmdGenerate)
}

func GenerateCode(cmd *commands.Command, args []string) int {
	currpath, _ := os.Getwd()
	if len(args) < 1 {
		beeLogger.Log.Fatal("Command is missing")
	}

	gps := utils.GetGOPATHs()
	if len(gps) == 0 {
		beeLogger.Log.Fatal("GOPATH environment variable is not set or empty")
	}

	gopath := gps[0]

	beeLogger.Log.Debugf("GOPATH: %s", utils.FILE(), utils.LINE(), gopath)

	gcmd := args[0]
	switch gcmd {
	case "scaffold":
		scaffold(cmd, args, currpath)
	case "docs":
		swaggergen.GenerateDocs(currpath)
	case "appcode":
		appCode(cmd, args, currpath)
	case "migration":
		migration(cmd, args, currpath)
	case "controller":
		controller(args, currpath)
	case "model":
		model(cmd, args, currpath)
	case "view":
		view(args, currpath)
	default:
		beeLogger.Log.Fatal("Command is missing")
	}
	beeLogger.Log.Successf("%s successfully generated!", strings.Title(gcmd))
	return 0
}

func scaffold(cmd *commands.Command, args []string, currpath string) {
	if len(args) < 2 {
		beeLogger.Log.Fatal("Wrong number of arguments. Run: bee help generate")
	}

	cmd.Flag.Parse(args[2:])
	if generate.SQLDriver == "" {
		generate.SQLDriver = utils.DocValue(config.Conf.Database.Driver)
		if generate.SQLDriver == "" {
			generate.SQLDriver = "mysql"
		}
	}
	if generate.SQLConn == "" {
		generate.SQLConn = utils.DocValue(config.Conf.Database.Conn)
		if generate.SQLConn == "" {
			generate.SQLConn = "root:@tcp(127.0.0.1:3306)/test"
		}
	}
	if generate.Fields == "" {
		beeLogger.Log.Hint("Fields option should not be empty, i.e. -Fields=\"title:string,body:text\"")
		beeLogger.Log.Fatal("Wrong number of arguments. Run: bee help generate")
	}
	sname := args[1]
	generate.GenerateScaffold(sname, generate.Fields.String(), currpath, generate.SQLDriver.String(), generate.SQLConn.String())
}

func appCode(cmd *commands.Command, args []string, currpath string) {
	cmd.Flag.Parse(args[1:])
	if generate.SQLDriver == "" {
		generate.SQLDriver = utils.DocValue(config.Conf.Database.Driver)
		if generate.SQLDriver == "" {
			generate.SQLDriver = "mysql"
		}
	}
	if generate.SQLConn == "" {
		generate.SQLConn = utils.DocValue(config.Conf.Database.Conn)
		if generate.SQLConn == "" {
			if generate.SQLDriver == "mysql" {
				generate.SQLConn = "root:@tcp(127.0.0.1:3306)/test"
			} else if generate.SQLDriver == "postgres" {
				generate.SQLConn = "postgres://postgres:postgres@127.0.0.1:5432/postgres"
			}
		}
	}
	if generate.Level == "" {
		generate.Level = "3"
	}
	beeLogger.Log.Infof("Using '%s' as 'SQLDriver'", generate.SQLDriver)
	beeLogger.Log.Infof("Using '%s' as 'SQLConn'", generate.SQLConn)
	beeLogger.Log.Infof("Using '%s' as 'Tables'", generate.Tables)
	beeLogger.Log.Infof("Using '%s' as 'Level'", generate.Level)
	generate.GenerateAppcode(generate.SQLDriver.String(), generate.SQLConn.String(), generate.Level.String(), generate.Tables.String(), currpath)
}

func migration(cmd *commands.Command, args []string, currpath string) {
	if len(args) < 2 {
		beeLogger.Log.Fatal("Wrong number of arguments. Run: bee help generate")
	}
	cmd.Flag.Parse(args[2:])
	mname := args[1]

	beeLogger.Log.Infof("Using '%s' as migration name", mname)

	upsql := ""
	downsql := ""
	if generate.Fields != "" {
		dbMigrator := generate.NewDBDriver()
		upsql = dbMigrator.GenerateCreateUp(mname)
		downsql = dbMigrator.GenerateCreateDown(mname)
	}
	generate.GenerateMigration(mname, upsql, downsql, currpath)
}

func controller(args []string, currpath string) {
	if len(args) == 2 {
		cname := args[1]
		generate.GenerateController(cname, currpath)
	} else {
		beeLogger.Log.Fatal("Wrong number of arguments. Run: bee help generate")
	}
}

func model(cmd *commands.Command, args []string, currpath string) {
	if len(args) < 2 {
		beeLogger.Log.Fatal("Wrong number of arguments. Run: bee help generate")
	}
	cmd.Flag.Parse(args[2:])
	if generate.Fields == "" {
		beeLogger.Log.Hint("Fields option should not be empty, i.e. -Fields=\"title:string,body:text\"")
		beeLogger.Log.Fatal("Wrong number of arguments. Run: bee help generate")
	}
	sname := args[1]
	generate.GenerateModel(sname, generate.Fields.String(), currpath)
}

func view(args []string, currpath string) {
	if len(args) == 2 {
		cname := args[1]
		generate.GenerateView(cname, currpath)
	} else {
		beeLogger.Log.Fatal("Wrong number of arguments. Run: bee help generate")
	}
}

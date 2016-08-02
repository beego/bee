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
	"os"
	"strings"
)

var cmdGenerate = &Command{
	UsageLine: "generate [Command]",
	Short:     "source code generator",
	Long: `
bee generate scaffold [scaffoldname] [-fields=""] [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    The generate scaffold command will do a number of things for you.
    -fields: a list of table fields. Format: field:type, ...
    -driver: [mysql | postgres | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test
    example: bee generate scaffold post -fields="title:string,body:text"

bee generate model [modelname] [-fields=""]
    generate RESTFul model based on fields
    -fields: a list of table fields. Format: field:type, ...

bee generate controller [controllerfile]
    generate RESTful controllers

bee generate view [viewpath]
    generate CRUD view in viewpath

bee generate migration [migrationfile] [-fields=""]
    generate migration file for making database schema update
    -fields: a list of table fields. Format: field:type, ...

bee generate docs
    generate swagger doc file

bee generate test [routerfile]
    generate testcase

bee generate appcode [-tables=""] [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"] [-level=3]
    generate appcode based on an existing database
    -tables: a list of table names separated by ',', default is empty, indicating all tables
    -driver: [mysql | postgres | sqlite], the default is mysql
    -conn:   the connection string used by the driver.
             default for mysql:    root:@tcp(127.0.0.1:3306)/test
             default for postgres: postgres://postgres:postgres@127.0.0.1:5432/postgres
    -level:  [1 | 2 | 3], 1 = models; 2 = models,controllers; 3 = models,controllers,router
`,
}

var driver docValue
var conn docValue
var level docValue
var tables docValue
var fields docValue

func init() {
	cmdGenerate.Run = generateCode
	cmdGenerate.Flag.Var(&tables, "tables", "specify tables to generate model")
	cmdGenerate.Flag.Var(&driver, "driver", "database driver: mysql, postgresql, etc.")
	cmdGenerate.Flag.Var(&conn, "conn", "connection string used by the driver to connect to a database instance")
	cmdGenerate.Flag.Var(&level, "level", "1 = models only; 2 = models and controllers; 3 = models, controllers and routers")
	cmdGenerate.Flag.Var(&fields, "fields", "specify the fields want to generate.")
}

func generateCode(cmd *Command, args []string) int {
	ShowShortVersionBanner()

	currpath, _ := os.Getwd()
	if len(args) < 1 {
		ColorLog("[ERRO] command is missing\n")
		os.Exit(2)
	}

	gps := GetGOPATHs()
	if len(gps) == 0 {
		ColorLog("[ERRO] Fail to start[ %s ]\n", "GOPATH environment variable is not set or empty")
		os.Exit(2)
	}
	gopath := gps[0]
	Debugf("GOPATH: %s", gopath)

	gcmd := args[0]
	switch gcmd {
	case "scaffold":
		if len(args) < 2 {
			ColorLog("[ERRO] Wrong number of arguments\n")
			ColorLog("[HINT] Usage: bee generate scaffold [scaffoldname] [-fields=\"\"]\n")
			os.Exit(2)
		}
		err := loadConfig()
		if err != nil {
			ColorLog("[ERRO] Fail to parse bee.json[ %s ]\n", err)
		}
		cmd.Flag.Parse(args[2:])
		if driver == "" {
			driver = docValue(conf.Database.Driver)
			if driver == "" {
				driver = "mysql"
			}
		}
		if conn == "" {
			conn = docValue(conf.Database.Conn)
			if conn == "" {
				conn = "root:@tcp(127.0.0.1:3306)/test"
			}
		}
		if fields == "" {
			ColorLog("[ERRO] Wrong number of arguments\n")
			ColorLog("[HINT] Usage: bee generate scaffold [scaffoldname] [-fields=\"title:string,body:text\"]\n")
			os.Exit(2)
		}
		sname := args[1]
		generateScaffold(sname, fields.String(), currpath, driver.String(), conn.String())
	case "docs":
		generateDocs(currpath)
	case "appcode":
		// load config
		err := loadConfig()
		if err != nil {
			ColorLog("[ERRO] Fail to parse bee.json[ %s ]\n", err)
		}
		cmd.Flag.Parse(args[1:])
		if driver == "" {
			driver = docValue(conf.Database.Driver)
			if driver == "" {
				driver = "mysql"
			}
		}
		if conn == "" {
			conn = docValue(conf.Database.Conn)
			if conn == "" {
				if driver == "mysql" {
					conn = "root:@tcp(127.0.0.1:3306)/test"
				} else if driver == "postgres" {
					conn = "postgres://postgres:postgres@127.0.0.1:5432/postgres"
				}
			}
		}
		if level == "" {
			level = "3"
		}
		ColorLog("[INFO] Using '%s' as 'driver'\n", driver)
		ColorLog("[INFO] Using '%s' as 'conn'\n", conn)
		ColorLog("[INFO] Using '%s' as 'tables'\n", tables)
		ColorLog("[INFO] Using '%s' as 'level'\n", level)
		generateAppcode(driver.String(), conn.String(), level.String(), tables.String(), currpath)
	case "migration":
		if len(args) < 2 {
			ColorLog("[ERRO] Wrong number of arguments\n")
			ColorLog("[HINT] Usage: bee generate migration [migrationname] [-fields=\"\"]\n")
			os.Exit(2)
		}
		cmd.Flag.Parse(args[2:])
		mname := args[1]
		ColorLog("[INFO] Using '%s' as migration name\n", mname)
		upsql := ""
		downsql := ""
		if fields != "" {
			upsql = `m.SQL("CREATE TABLE ` + mname + "(" + generateSQLFromFields(fields.String()) + `)");`
			downsql = `m.SQL("DROP TABLE ` + "`" + mname + "`" + `")`
			if driver == "postgres" {
				downsql = strings.Replace(downsql, "`", "", -1)
			}
		}
		generateMigration(mname, upsql, downsql, currpath)
	case "controller":
		if len(args) == 2 {
			cname := args[1]
			generateController(cname, currpath)
		} else {
			ColorLog("[ERRO] Wrong number of arguments\n")
			ColorLog("[HINT] Usage: bee generate controller [controllername]\n")
			os.Exit(2)
		}
	case "model":
		if len(args) < 2 {
			ColorLog("[ERRO] Wrong number of arguments\n")
			ColorLog("[HINT] Usage: bee generate model [modelname] [-fields=\"\"]\n")
			os.Exit(2)
		}
		cmd.Flag.Parse(args[2:])
		if fields == "" {
			ColorLog("[ERRO] Wrong number of arguments\n")
			ColorLog("[HINT] Usage: bee generate model [modelname] [-fields=\"title:string,body:text\"]\n")
			os.Exit(2)
		}
		sname := args[1]
		generateModel(sname, fields.String(), currpath)
	case "view":
		if len(args) == 2 {
			cname := args[1]
			generateView(cname, currpath)
		} else {
			ColorLog("[ERRO] Wrong number of arguments\n")
			ColorLog("[HINT] Usage: bee generate view [viewpath]\n")
			os.Exit(2)
		}
	default:
		ColorLog("[ERRO] Command is missing\n")
	}
	ColorLog("[SUCC] %s successfully generated!\n", strings.Title(gcmd))
	return 0
}

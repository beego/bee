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
	cmdGenerate.PreRun = func(cmd *Command, args []string) { ShowShortVersionBanner() }
	cmdGenerate.Flag.Var(&tables, "tables", "specify tables to generate model")
	cmdGenerate.Flag.Var(&driver, "driver", "database driver: mysql, postgresql, etc.")
	cmdGenerate.Flag.Var(&conn, "conn", "connection string used by the driver to connect to a database instance")
	cmdGenerate.Flag.Var(&level, "level", "1 = models only; 2 = models and controllers; 3 = models, controllers and routers")
	cmdGenerate.Flag.Var(&fields, "fields", "specify the fields want to generate.")
}

func generateCode(cmd *Command, args []string) int {
	currpath, _ := os.Getwd()
	if len(args) < 1 {
		logger.Fatal("Command is missing")
	}

	gps := GetGOPATHs()
	if len(gps) == 0 {
		logger.Fatal("GOPATH environment variable is not set or empty")
	}

	gopath := gps[0]

	logger.Debugf("GOPATH: %s", __FILE__(), __LINE__(), gopath)

	gcmd := args[0]
	switch gcmd {
	case "scaffold":
		if len(args) < 2 {
			logger.Fatal("Wrong number of arguments. Run: bee help generate")
		}
		// Load the configuration
		err := loadConfig()
		if err != nil {
			logger.Fatalf("Failed to load configuration: %s", err)
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
			logger.Hint("fields option should not be empty, i.e. -fields=\"title:string,body:text\"")
			logger.Fatal("Wrong number of arguments. Run: bee help generate")
		}
		sname := args[1]
		generateScaffold(sname, fields.String(), currpath, driver.String(), conn.String())
	case "docs":
		generateDocs(currpath)
	case "appcode":
		// Load the configuration
		err := loadConfig()
		if err != nil {
			logger.Fatalf("Failed to load configuration: %s", err)
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
		logger.Infof("Using '%s' as 'driver'", driver)
		logger.Infof("Using '%s' as 'conn'", conn)
		logger.Infof("Using '%s' as 'tables'", tables)
		logger.Infof("Using '%s' as 'level'", level)
		generateAppcode(driver.String(), conn.String(), level.String(), tables.String(), currpath)
	case "migration":
		if len(args) < 2 {
			logger.Fatal("Wrong number of arguments. Run: bee help generate")
		}
		cmd.Flag.Parse(args[2:])
		mname := args[1]

		logger.Infof("Using '%s' as migration name", mname)

		upsql := ""
		downsql := ""
		if fields != "" {
			dbMigrator := newDBDriver()
			upsql = dbMigrator.generateCreateUp(mname)
			downsql = dbMigrator.generateCreateDown(mname)
		}
		generateMigration(mname, upsql, downsql, currpath)
	case "controller":
		if len(args) == 2 {
			cname := args[1]
			generateController(cname, currpath)
		} else {
			logger.Fatal("Wrong number of arguments. Run: bee help generate")
		}
	case "model":
		if len(args) < 2 {
			logger.Fatal("Wrong number of arguments. Run: bee help generate")
		}
		cmd.Flag.Parse(args[2:])
		if fields == "" {
			logger.Hint("fields option should not be empty, i.e. -fields=\"title:string,body:text\"")
			logger.Fatal("Wrong number of arguments. Run: bee help generate")
		}
		sname := args[1]
		generateModel(sname, fields.String(), currpath)
	case "view":
		if len(args) == 2 {
			cname := args[1]
			generateView(cname, currpath)
		} else {
			logger.Fatal("Wrong number of arguments. Run: bee help generate")
		}
	default:
		logger.Fatal("Command is missing")
	}
	logger.Successf("%s successfully generated!", strings.Title(gcmd))
	return 0
}

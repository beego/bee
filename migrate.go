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

import "os"

var cmdMigrate = &Command{
	UsageLine: "migrate [Command]",
	Short:     "run database migrations",
	Long: `
bee migrate
    run all outstanding migrations

bee migrate rollback
    rollback the last migration operation

bee migrate reset
    rollback all migrations

bee migrate refresh
    rollback all migrations and run them all again
`,
}

func init() {
	cmdMigrate.Run = runMigration
}

func runMigration(cmd *Command, args []string) {
	//curpath, _ := os.Getwd()

	gopath := os.Getenv("GOPATH")
	Debugf("gopath:%s", gopath)
	if gopath == "" {
		ColorLog("[ERRO] $GOPATH not found\n")
		ColorLog("[HINT] Set $GOPATH in your environment vairables\n")
		os.Exit(2)
	}

	if len(args) == 0 {
		// run all outstanding migrations
		ColorLog("[INFO] running all outstanding migrations\n")
		migrateUpdate()
	} else {
		mcmd := args[0]
		switch mcmd {
		case "rollback":
			ColorLog("[INFO] rolling back the last migration operation\n")
			migrateRollback()
		case "reset":
			ColorLog("[INFO] reseting all migrations\n")
			migrateReset()
		case "refresh":
			ColorLog("[INFO] refreshing all migrations\n")
			migrateReset()
		default:
			ColorLog("[ERRO] command is missing\n")
			os.Exit(2)
		}
		ColorLog("[SUCC] migration successful!\n")
	}
}

func migrateUpdate() {
	println("=>update")
}

func migrateRollback() {
	println("=>rollback")
}

func migrateReset() {
	println("=>reset")
}

func migrateRefresh() {
	println("=>refresh")
}

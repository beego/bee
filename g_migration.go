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
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

const (
	M_PATH        = "migrations"
	M_DATE_FORMAT = "2006-01-02"
)

// generateMigration generates migration file template for database schema update.
// The generated file template consists of an up() method for updating schema and
// a down() method for reverting the update.
func generateMigration(mname string, curpath string) {
	migrationFilePath := path.Join(curpath, M_PATH)
	if _, err := os.Stat(migrationFilePath); os.IsNotExist(err) {
		// create migrations directory
		if err := os.Mkdir(migrationFilePath, 0777); err != nil {
			ColorLog("[ERRO] Could not create migration directory: %s\n", err)
			os.Exit(2)
		}
	}
	// create file
	today := time.Now().Format(M_DATE_FORMAT)
	fpath := path.Join(migrationFilePath, fmt.Sprintf("%s_%s.go", today, mname))
	if f, err := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer f.Close()
		content := strings.Replace(MIGRATION_TPL, "{{StructName}}", camelCase(mname), -1)
		content = strings.Replace(content, "{{DateFormat}}", M_DATE_FORMAT, -1)
		content = strings.Replace(content, "{{CurrTime}}", today, -1)
		f.WriteString(content)
		// gofmt generated source code
		formatSourceCode(fpath)
		ColorLog("[INFO] Migration file generated: %s\n", fpath)
	} else {
		// error creating file
		ColorLog("[ERRO] Could not create migration file: %s\n", err)
		os.Exit(2)
	}
}

// formatSourceCode formats the source code using gofmt
func formatSourceCode(fpath string) {
	cmd := exec.Command("gofmt", "-w", fpath)
	cmd.Run()
}

const MIGRATION_TPL = `
package main

import (
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/migration"
)

func init() {
	m := &{{StructName}}{}
	m.Created = time.Parse("{{DateFormat}}", "{{CurrTime}}")
	migration.Register(m)
}

type {{StructName}} struct {
	migration.Migration
}

// Run the migrations
func (m *{{StructName}}) up() {
	// use m.Sql("create table ...") to make schema update
}

// Reverse the migrations
func (m *{{StructName}}) down() {
	// use m.Sql("drop table ...") to reverse schema update
}
`

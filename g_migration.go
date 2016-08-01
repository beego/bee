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
	"path"
	"strings"
	"time"
)

const (
	MPath = "migrations"
	MDateFormat = "20060102_150405"
)

// generateMigration generates migration file template for database schema update.
// The generated file template consists of an up() method for updating schema and
// a down() method for reverting the update.
func generateMigration(mname, upsql, downsql, curpath string) {
	migrationFilePath := path.Join(curpath, "database", MPath)
	if _, err := os.Stat(migrationFilePath); os.IsNotExist(err) {
		// create migrations directory
		if err := os.MkdirAll(migrationFilePath, 0777); err != nil {
			ColorLog("[ERRO] Could not create migration directory: %s\n", err)
			os.Exit(2)
		}
	}
	// create file
	today := time.Now().Format(MDateFormat)
	fpath := path.Join(migrationFilePath, fmt.Sprintf("%s_%s.go", today, mname))
	if f, err := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer f.Close()
		content := strings.Replace(MigrationTPL, "{{StructName}}", camelCase(mname)+"_"+today, -1)
		content = strings.Replace(content, "{{CurrTime}}", today, -1)
		content = strings.Replace(content, "{{UpSQL}}", upsql, -1)
		content = strings.Replace(content, "{{DownSQL}}", downsql, -1)
		f.WriteString(content)
		// gofmt generated source code
		formatSourceCode(fpath)
		fmt.Println("\tcreate\t", fpath)
	} else {
		// error creating file
		ColorLog("[ERRO] Could not create migration file: %s\n", err)
		os.Exit(2)
	}
}

const MigrationTPL = `package main

import (
	"github.com/astaxie/beego/migration"
)

// DO NOT MODIFY
type {{StructName}} struct {
	migration.Migration
}

// DO NOT MODIFY
func init() {
	m := &{{StructName}}{}
	m.Created = "{{CurrTime}}"
	migration.Register("{{StructName}}", m)
}

// Run the migrations
func (m *{{StructName}}) Up() {
	// use m.SQL("CREATE TABLE ...") to make schema update
	{{UpSQL}}
}

// Reverse the migrations
func (m *{{StructName}}) Down() {
	// use m.SQL("DROP TABLE ...") to reverse schema update
	{{DownSQL}}
}
`

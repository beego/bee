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
	MPath       = "migrations"
	MDateFormat = "20060102_150405"
	DBPath      = "database"
)

type DBDriver interface {
	generateCreateUp(tableName string) string
	generateCreateDown(tableName string) string
}

type mysqlDriver struct{}

func (m mysqlDriver) generateCreateUp(tableName string) string {
	upsql := `m.SQL("CREATE TABLE ` + tableName + "(" + m.generateSQLFromFields(fields.String()) + `)");`
	return upsql
}

func (m mysqlDriver) generateCreateDown(tableName string) string {
	downsql := `m.SQL("DROP TABLE ` + "`" + tableName + "`" + `")`
	return downsql
}

func (m mysqlDriver) generateSQLFromFields(fields string) string {
	sql, tags := "", ""
	fds := strings.Split(fields, ",")
	for i, v := range fds {
		kv := strings.SplitN(v, ":", 2)
		if len(kv) != 2 {
			logger.Error("Fields format is wrong. Should be: key:type,key:type " + v)
			return ""
		}
		typ, tag := m.getSQLType(kv[1])
		if typ == "" {
			logger.Error("Fields format is wrong. Should be: key:type,key:type " + v)
			return ""
		}
		if i == 0 && strings.ToLower(kv[0]) != "id" {
			sql += "`id` int(11) NOT NULL AUTO_INCREMENT,"
			tags = tags + "PRIMARY KEY (`id`),"
		}
		sql += "`" + snakeString(kv[0]) + "` " + typ + ","
		if tag != "" {
			tags = tags + fmt.Sprintf(tag, "`"+snakeString(kv[0])+"`") + ","
		}
	}
	sql = strings.TrimRight(sql+tags, ",")
	return sql
}

func (m mysqlDriver) getSQLType(ktype string) (tp, tag string) {
	kv := strings.SplitN(ktype, ":", 2)
	switch kv[0] {
	case "string":
		if len(kv) == 2 {
			return "varchar(" + kv[1] + ") NOT NULL", ""
		}
		return "varchar(128) NOT NULL", ""
	case "text":
		return "longtext  NOT NULL", ""
	case "auto":
		return "int(11) NOT NULL AUTO_INCREMENT", ""
	case "pk":
		return "int(11) NOT NULL", "PRIMARY KEY (%s)"
	case "datetime":
		return "datetime NOT NULL", ""
	case "int", "int8", "int16", "int32", "int64":
		fallthrough
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "int(11) DEFAULT NULL", ""
	case "bool":
		return "tinyint(1) NOT NULL", ""
	case "float32", "float64":
		return "float NOT NULL", ""
	case "float":
		return "float NOT NULL", ""
	}
	return "", ""
}

type postgresqlDriver struct{}

func (m postgresqlDriver) generateCreateUp(tableName string) string {
	upsql := `m.SQL("CREATE TABLE ` + tableName + "(" + m.generateSQLFromFields(fields.String()) + `)");`
	return upsql
}

func (m postgresqlDriver) generateCreateDown(tableName string) string {
	downsql := `m.SQL("DROP TABLE ` + tableName + `")`
	return downsql
}

func (m postgresqlDriver) generateSQLFromFields(fields string) string {
	sql, tags := "", ""
	fds := strings.Split(fields, ",")
	for i, v := range fds {
		kv := strings.SplitN(v, ":", 2)
		if len(kv) != 2 {
			logger.Error("Fields format is wrong. Should be: key:type,key:type " + v)
			return ""
		}
		typ, tag := m.getSQLType(kv[1])
		if typ == "" {
			logger.Error("Fields format is wrong. Should be: key:type,key:type " + v)
			return ""
		}
		if i == 0 && strings.ToLower(kv[0]) != "id" {
			sql += "id serial primary key,"
		}
		sql += snakeString(kv[0]) + " " + typ + ","
		if tag != "" {
			tags = tags + fmt.Sprintf(tag, snakeString(kv[0])) + ","
		}
	}
	if tags != "" {
		sql = strings.TrimRight(sql+" "+tags, ",")
	} else {
		sql = strings.TrimRight(sql, ",")
	}
	return sql
}

func (m postgresqlDriver) getSQLType(ktype string) (tp, tag string) {
	kv := strings.SplitN(ktype, ":", 2)
	switch kv[0] {
	case "string":
		if len(kv) == 2 {
			return "char(" + kv[1] + ") NOT NULL", ""
		}
		return "TEXT NOT NULL", ""
	case "text":
		return "TEXT NOT NULL", ""
	case "auto", "pk":
		return "serial primary key", ""
	case "datetime":
		return "TIMESTAMP WITHOUT TIME ZONE NOT NULL", ""
	case "int", "int8", "int16", "int32", "int64":
		fallthrough
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "integer DEFAULT NULL", ""
	case "bool":
		return "boolean NOT NULL", ""
	case "float32", "float64", "float":
		return "numeric NOT NULL", ""
	}
	return "", ""
}

func newDBDriver() DBDriver {
	switch driver {
	case "mysql":
		return mysqlDriver{}
	case "postgres":
		return postgresqlDriver{}
	default:
		logger.Fatal("Driver not supported")
		return nil
	}
}

// generateMigration generates migration file template for database schema update.
// The generated file template consists of an up() method for updating schema and
// a down() method for reverting the update.
func generateMigration(mname, upsql, downsql, curpath string) {
	w := NewColorWriter(os.Stdout)
	migrationFilePath := path.Join(curpath, DBPath, MPath)
	if _, err := os.Stat(migrationFilePath); os.IsNotExist(err) {
		// create migrations directory
		if err := os.MkdirAll(migrationFilePath, 0777); err != nil {
			logger.Fatalf("Could not create migration directory: %s", err)
		}
	}
	// create file
	today := time.Now().Format(MDateFormat)
	fpath := path.Join(migrationFilePath, fmt.Sprintf("%s_%s.go", today, mname))
	if f, err := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer CloseFile(f)
		content := strings.Replace(MigrationTPL, "{{StructName}}", camelCase(mname)+"_"+today, -1)
		content = strings.Replace(content, "{{CurrTime}}", today, -1)
		content = strings.Replace(content, "{{UpSQL}}", upsql, -1)
		content = strings.Replace(content, "{{DownSQL}}", downsql, -1)
		f.WriteString(content)
		// Run 'gofmt' on the generated source code
		formatSourceCode(fpath)
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", fpath, "\x1b[0m")
	} else {
		logger.Fatalf("Could not create migration file: %s", err)
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

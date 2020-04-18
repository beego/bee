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
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/beego/bee/logger"
	"github.com/beego/bee/logger/colors"
	"github.com/beego/bee/utils"
)

const (
	MPath       = "migrations"
	MDateFormat = "20060102_150405"
	DBPath      = "database"
)

type DBDriver interface {
	GenerateCreateUp(tableName string) string
	GenerateCreateDown(tableName string) string
}

type mysqlDriver struct{}

func (m mysqlDriver) GenerateCreateUp(tableName string) string {
	upsql := `m.SQL("CREATE TABLE ` + tableName + "(" + m.generateSQLFromFields(Fields.String()) + `)");`
	return upsql
}

func (m mysqlDriver) GenerateCreateDown(tableName string) string {
	downsql := `m.SQL("DROP TABLE ` + "`" + tableName + "`" + `")`
	return downsql
}

func (m mysqlDriver) generateSQLFromFields(fields string) string {
	sql, tags := "", ""
	fds := strings.Split(fields, ",")
	for i, v := range fds {
		kv := strings.SplitN(v, ":", 2)
		if len(kv) != 2 {
			beeLogger.Log.Error("Fields format is wrong. Should be: key:type,key:type " + v)
			return ""
		}
		typ, tag := m.getSQLType(kv[1])
		if typ == "" {
			beeLogger.Log.Error("Fields format is wrong. Should be: key:type,key:type " + v)
			return ""
		}
		if i == 0 && strings.ToLower(kv[0]) != "id" {
			sql += "`id` int(11) NOT NULL AUTO_INCREMENT,"
			tags = tags + "PRIMARY KEY (`id`),"
		}
		sql += "`" + utils.SnakeString(kv[0]) + "` " + typ + ","
		if tag != "" {
			tags = tags + fmt.Sprintf(tag, "`"+utils.SnakeString(kv[0])+"`") + ","
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

func (m postgresqlDriver) GenerateCreateUp(tableName string) string {
	upsql := `m.SQL("CREATE TABLE ` + tableName + "(" + m.generateSQLFromFields(Fields.String()) + `)");`
	return upsql
}

func (m postgresqlDriver) GenerateCreateDown(tableName string) string {
	downsql := `m.SQL("DROP TABLE ` + tableName + `")`
	return downsql
}

func (m postgresqlDriver) generateSQLFromFields(fields string) string {
	sql, tags := "", ""
	fds := strings.Split(fields, ",")
	for i, v := range fds {
		kv := strings.SplitN(v, ":", 2)
		if len(kv) != 2 {
			beeLogger.Log.Error("Fields format is wrong. Should be: key:type,key:type " + v)
			return ""
		}
		typ, tag := m.getSQLType(kv[1])
		if typ == "" {
			beeLogger.Log.Error("Fields format is wrong. Should be: key:type,key:type " + v)
			return ""
		}
		if i == 0 && strings.ToLower(kv[0]) != "id" {
			sql += "id serial primary key,"
		}
		sql += utils.SnakeString(kv[0]) + " " + typ + ","
		if tag != "" {
			tags = tags + fmt.Sprintf(tag, utils.SnakeString(kv[0])) + ","
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

func NewDBDriver() DBDriver {
	switch SQLDriver {
	case "mysql":
		return mysqlDriver{}
	case "postgres":
		return postgresqlDriver{}
	default:
		beeLogger.Log.Fatal("Driver not supported")
		return nil
	}
}

// generateMigration generates migration file template for database schema update.
// The generated file template consists of an up() method for updating schema and
// a down() method for reverting the update.
func GenerateMigration(mname, upsql, downsql, curpath string) {
	w := colors.NewColorWriter(os.Stdout)
	migrationFilePath := path.Join(curpath, DBPath, MPath)
	if _, err := os.Stat(migrationFilePath); os.IsNotExist(err) {
		// create migrations directory
		if err := os.MkdirAll(migrationFilePath, 0777); err != nil {
			beeLogger.Log.Fatalf("Could not create migration directory: %s", err)
		}
	}
	// create file
	today := time.Now().Format(MDateFormat)
	fpath := path.Join(migrationFilePath, fmt.Sprintf("%s_%s.go", today, mname))
	if f, err := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer utils.CloseFile(f)
		ddlSpec := ""
		spec := ""
		up := ""
		down := ""
		if DDL != "" {
			ddlSpec = "m.ddlSpec()"
			switch strings.Title(DDL.String()) {
			case "Create":
				spec = strings.Replace(DDLSpecCreate, "{{StructName}}", utils.CamelCase(mname)+"_"+today, -1)
			case "Alter":
				spec = strings.Replace(DDLSpecAlter, "{{StructName}}", utils.CamelCase(mname)+"_"+today, -1)
			}
			spec = strings.Replace(spec, "{{tableName}}", mname, -1)
		} else {
			up = strings.Replace(MigrationUp, "{{UpSQL}}", upsql, -1)
			up = strings.Replace(up, "{{StructName}}", utils.CamelCase(mname)+"_"+today, -1)
			down = strings.Replace(MigrationDown, "{{DownSQL}}", downsql, -1)
			down = strings.Replace(down, "{{StructName}}", utils.CamelCase(mname)+"_"+today, -1)
		}

		header := strings.Replace(MigrationHeader, "{{StructName}}", utils.CamelCase(mname)+"_"+today, -1)
		header = strings.Replace(header, "{{ddlSpec}}", ddlSpec, -1)
		header = strings.Replace(header, "{{CurrTime}}", today, -1)
		f.WriteString(header + spec + up + down)
		// Run 'gofmt' on the generated source code
		utils.FormatSourceCode(fpath)
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", fpath, "\x1b[0m")
	} else {
		beeLogger.Log.Fatalf("Could not create migration file: %s", err)
	}
}

const (
	MigrationHeader = `package main
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
							{{ddlSpec}}
							migration.Register("{{StructName}}", m)
						}
					   `

	DDLSpecCreate = `
				/*
				refer beego/migration/doc.go
				*/
				func(m *{{StructName}}) ddlSpec(){
				m.CreateTable("{{tableName}}", "InnoDB", "utf8")
				m.PriCol("id").SetAuto(true).SetNullable(false).SetDataType("INT(10)").SetUnsigned(true)

				}
				`
	DDLSpecAlter = `
				/*
				refer beego/migration/doc.go
				*/
				func(m *{{StructName}}) ddlSpec(){
				m.AlterTable("{{tableName}}")

				}
				`
	MigrationUp = `
				// Run the migrations
				func (m *{{StructName}}) Up() {
					// use m.SQL("CREATE TABLE ...") to make schema update
					{{UpSQL}}
				}`
	MigrationDown = `
				// Reverse the migrations
				func (m *{{StructName}}) Down() {
					// use m.SQL("DROP TABLE ...") to reverse schema update
					{{DownSQL}}
				}
				`
)

// Copyright 2013 Dylan LYU (mingzong.lyu@gmail.com)
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
	"errors"
	"os"
	"path"
	"strings"
)

func generateStructure(cname, fields, crupath string) {
	p, f := path.Split(cname)
	structureName := strings.Title(f)
	packageName := "structures"
	if p != "" {
		i := strings.LastIndex(p[:len(p)-1], "/")
		packageName = p[i+1 : len(p)-1]
	}
	ColorLog("[INFO] Using '%s' as structure name\n", structureName)
	ColorLog("[INFO] Using '%s' as package name\n", packageName)
	fp := path.Join(crupath, packageName, p)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		// create controller directory
		if err := os.MkdirAll(fp, 0777); err != nil {
			ColorLog("[ERRO] Could not create structures directory: %s\n", err)
			os.Exit(2)
		}
	}
	fpath := path.Join(fp, strings.ToLower(structureName)+"_structure.go")
	if f, err := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer f.Close()
		var content string

		if fields != "" {
			structStruct, err, hastime := getStruct(structureName, fields)
			if err != nil {
				ColorLog("[ERRO] Could not genrate struct: %s\n", err)
				os.Exit(2)
			}
			content = strings.Replace(STRUCTURE_TPL, "{{packageName}}", packageName, -1)
			content = strings.Replace(content, "{{structStruct}}", structStruct, -1)
			if hastime {
				content = strings.Replace(content, "{{timePkg}}", `"time"`, -1)
			} else {
				content = strings.Replace(content, "{{timePkg}}", "", -1)
			}

		} else {
			content = strings.Replace(BAST_STRUCTURE_TPL, "{{packageName}}", packageName, -1)
		}
		content = strings.Replace(content, "{{structureName}}", structureName, -1)
		f.WriteString(content)
		// gofmt generated source code
		formatSourceCode(fpath)
		ColorLog("[INFO] structure file generated: %s\n", fpath)
	} else {
		// error creating file
		ColorLog("[ERRO] Could not create structure file: %s\n", err)
		os.Exit(2)
	}

}

func getStruct(structname, fields string) (string, error, bool) {
	if fields == "" {
		return "", errors.New("fields can't empty"), false
	}
	hastime := false
	structStr := "type " + structname + " struct{\n"
	fds := strings.Split(fields, ",")
	for i, v := range fds {
		kv := strings.SplitN(v, ":", 2)
		if len(kv) != 2 {
			return "", errors.New("the filds format is wrong. should key:type,key:type " + v), false
		}
		typ, tag, hastimeinner := getType(kv[1])
		if typ == "" {
			return "", errors.New("the filds format is wrong. should key:type,key:type " + v), false
		}
		if i == 0 && strings.ToLower(kv[0]) != "id" {
			structStr = structStr + "Id     int64     `orm:\"auto\"`\n"
		}
		if hastimeinner {
			hastime = true
		}
		structStr = structStr + camelString(kv[0]) + "       " + typ + "     " + tag + "\n"
	}
	structStr += "}\n"
	return structStr, nil, hastime
}

// fields support type
// http://beego.me/docs/mvc/model/models.md#mysql
func getType(ktype string) (kt, tag string, hasTime bool) {
	kv := strings.SplitN(ktype, ":", 2)
	switch kv[0] {
	case "string":
		if len(kv) == 2 {
			return "string", "`orm:\"size(" + kv[1] + ")\"`", false
		} else {
			return "string", "`orm:\"size(128)\"`", false
		}
	case "text":
		return "string", "`orm:\"type(longtext)\"`", false
	case "auto":
		return "int64", "`orm:\"auto\"`", false
	case "pk":
		return "int64", "`orm:\"pk\"`", false
	case "datetime":
		return "time.Time", "`orm:\"type(datetime)\"`", true
	case "int", "int8", "int16", "int32", "int64":
		fallthrough
	case "uint", "uint8", "uint16", "uint32", "uint64":
		fallthrough
	case "bool":
		fallthrough
	case "float32", "float64":
		return kv[0], "", false
	case "float":
		return "float64", "", false
	}
	return "", "", false
}

const (
	BAST_STRUCTURE_TPL = `package {{packageName}}

	type {{structureName}} struct {

	}
	`

	STRUCTURE_TPL = `package {{packageName}}

	import(
	"github.com/astaxie/beego/orm"

	{{timePkg}}
	)

	{{structStruct}}

	func init() {
		orm.RegisterModel(new({{structureName}}))
	}
	`
)

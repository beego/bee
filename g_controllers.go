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
	"path"
	"strings"
)

// article
// cms/article
//
func generateController(cname, crupath string) {
	p, f := path.Split(cname)
	controllerName := strings.Title(f)
	packageName := "controllers"
	if p != "" {
		i := strings.LastIndex(p[:len(p)-1], "/")
		packageName = p[i+1 : len(p)-1]
	}
	ColorLog("[INFO] Using '%s' as controller name\n", controllerName)
	ColorLog("[INFO] Using '%s' as package name\n", packageName)
	fp := path.Join(crupath, "controllers", p)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		// create controller directory
		if err := os.MkdirAll(fp, 0777); err != nil {
			ColorLog("[ERRO] Could not create controllers directory: %s\n", err)
			os.Exit(2)
		}
	}
	fpath := path.Join(fp, strings.ToLower(controllerName)+".go")
	if f, err := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer f.Close()
		modelPath := path.Join(crupath, "models", strings.ToLower(controllerName)+".go")
		var content string
		if _, err := os.Stat(modelPath); err == nil {
			ColorLog("[INFO] Using matching model '%s'\n", controllerName)
			content = strings.Replace(controllerModelTpl, "{{packageName}}", packageName, -1)
			pkgPath := getPackagePath(crupath)
			content = strings.Replace(content, "{{pkgPath}}", pkgPath, -1)
		} else {
			content = strings.Replace(controllerTpl, "{{packageName}}", packageName, -1)
		}
		content = strings.Replace(content, "{{controllerName}}", controllerName, -1)
		f.WriteString(content)
		// gofmt generated source code
		formatSourceCode(fpath)
		ColorLog("[INFO] controller file generated: %s\n", fpath)
	} else {
		// error creating file
		ColorLog("[ERRO] Could not create controller file: %s\n", err)
		os.Exit(2)
	}
}

var controllerTpl = `package {{packageName}}

import (
	"github.com/astaxie/beego"
)

// operations for {{controllerName}}
type {{controllerName}}Controller struct {
	beego.Controller
}

func (c *{{controllerName}}Controller) URLMapping() {
	c.Mapping("Post", c.Post)
	c.Mapping("GetOne", c.GetOne)
	c.Mapping("GetAll", c.GetAll)
	c.Mapping("Put", c.Put)
	c.Mapping("Delete", c.Delete)
}

// @Title Post
// @Description create {{controllerName}}
// @Param	body		body 	models.{{controllerName}}	true		"body for {{controllerName}} content"
// @Success 200 {int} models.{{controllerName}}.Id
// @Failure 403 body is empty
// @router / [post]
func (c *{{controllerName}}Controller) Post() {

}

// @Title Get
// @Description get {{controllerName}} by id
// @Param	id		path 	string	true		"The key for staticblock"
// @Success 200 {object} models.{{controllerName}}
// @Failure 403 :id is empty
// @router /:id [get]
func (c *{{controllerName}}Controller) GetOne() {

}

// @Title Get All
// @Description get {{controllerName}}
// @Param	query	query	string	false	"Filter. e.g. col1:v1,col2:v2 ..."
// @Param	fields	query	string	false	"Fields returned. e.g. col1,col2 ..."
// @Param	sortby	query	string	false	"Sorted-by fields. e.g. col1,col2 ..."
// @Param	order	query	string	false	"Order corresponding to each sortby field, if single value, apply to all sortby fields. e.g. desc,asc ..."
// @Param	limit	query	string	false	"Limit the size of result set. Must be an integer"
// @Param	offset	query	string	false	"Start position of result set. Must be an integer"
// @Success 200 {object} models.{{controllerName}}
// @Failure 403
// @router / [get]
func (c *{{controllerName}}Controller) GetAll() {

}

// @Title Update
// @Description update the {{controllerName}}
// @Param	id		path 	string	true		"The id you want to update"
// @Param	body		body 	models.{{controllerName}}	true		"body for {{controllerName}} content"
// @Success 200 {object} models.{{controllerName}}
// @Failure 403 :id is not int
// @router /:id [put]
func (c *{{controllerName}}Controller) Put() {
	
}

// @Title Delete
// @Description delete the {{controllerName}}
// @Param	id		path 	string	true		"The id you want to delete"
// @Success 200 {string} delete success!
// @Failure 403 id is empty
// @router /:id [delete]
func (c *{{controllerName}}Controller) Delete() {
	
}
`

var controllerModelTpl = `package {{packageName}}

import (
	"{{pkgPath}}/models"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/astaxie/beego"
)

// oprations for {{controllerName}}
type {{controllerName}}Controller struct {
	beego.Controller
}

func (c *{{controllerName}}Controller) URLMapping() {
	c.Mapping("Post", c.Post)
	c.Mapping("GetOne", c.GetOne)
	c.Mapping("GetAll", c.GetAll)
	c.Mapping("Put", c.Put)
	c.Mapping("Delete", c.Delete)
}

// @Title Post
// @Description create {{controllerName}}
// @Param	body		body 	models.{{controllerName}}	true		"body for {{controllerName}} content"
// @Success 200 {int} models.{{controllerName}}.Id
// @Failure 403 body is empty
// @router / [post]
func (c *{{controllerName}}Controller) Post() {
	var v models.{{controllerName}}
	json.Unmarshal(c.Ctx.Input.RequestBody, &v)
	if id, err := models.Add{{controllerName}}(&v); err == nil {
		c.Data["json"] = map[string]int64{"id": id}
	} else {
		c.Data["json"] = err.Error()
	}
	c.ServeJson()
}

// @Title Get
// @Description get {{controllerName}} by id
// @Param	id		path 	string	true		"The key for staticblock"
// @Success 200 {object} models.{{controllerName}}
// @Failure 403 :id is empty
// @router /:id [get]
func (c *{{controllerName}}Controller) GetOne() {
	idStr := c.Ctx.Input.Params[":id"]
	id, _ := strconv.ParseInt(idStr, 0, 64)
	v, err := models.Get{{controllerName}}ById(id)
	if err != nil {
		c.Data["json"] = err.Error()
	} else {
		c.Data["json"] = v
	}
	c.ServeJson()
}

// @Title Get All
// @Description get {{controllerName}}
// @Param	query	query	string	false	"Filter. e.g. col1:v1,col2:v2 ..."
// @Param	fields	query	string	false	"Fields returned. e.g. col1,col2 ..."
// @Param	sortby	query	string	false	"Sorted-by fields. e.g. col1,col2 ..."
// @Param	order	query	string	false	"Order corresponding to each sortby field, if single value, apply to all sortby fields. e.g. desc,asc ..."
// @Param	limit	query	string	false	"Limit the size of result set. Must be an integer"
// @Param	offset	query	string	false	"Start position of result set. Must be an integer"
// @Success 200 {object} models.{{controllerName}}
// @Failure 403 
// @router / [get]
func (c *{{controllerName}}Controller) GetAll() {
	var fields []string
	var sortby []string
	var order []string
	var query map[string]string = make(map[string]string)
	var limit int64 = 10
	var offset int64 = 0

	// fields: col1,col2,entity.col3
	if v := c.GetString("fields"); v != "" {
		fields = strings.Split(v, ",")
	}
	// limit: 10 (default is 10)
	if v, err := c.GetInt64("limit"); err == nil {
		limit = v
	}
	// offset: 0 (default is 0)
	if v, err := c.GetInt64("offset"); err == nil {
		offset = v
	}
	// sortby: col1,col2
	if v := c.GetString("sortby"); v != "" {
		sortby = strings.Split(v, ",")
	}
	// order: desc,asc
	if v := c.GetString("order"); v != "" {
		order = strings.Split(v, ",")
	}
	// query: k:v,k:v
	if v := c.GetString("query"); v != "" {
		for _, cond := range strings.Split(v, ",") {
			kv := strings.Split(cond, ":")
			if len(kv) != 2 {
				c.Data["json"] = errors.New("Error: invalid query key/value pair")
				c.ServeJson()
				return
			}
			k, v := kv[0], kv[1]
			query[k] = v
		}
	}

	l, err := models.GetAll{{controllerName}}(query, fields, sortby, order, offset, limit)
	if err != nil {
		c.Data["json"] = err.Error()
	} else {
		c.Data["json"] = l
	}
	c.ServeJson()
}

// @Title Update
// @Description update the {{controllerName}}
// @Param	id		path 	string	true		"The id you want to update"
// @Param	body		body 	models.{{controllerName}}	true		"body for {{controllerName}} content"
// @Success 200 {object} models.{{controllerName}}
// @Failure 403 :id is not int
// @router /:id [put]
func (c *{{controllerName}}Controller) Put() {
	idStr := c.Ctx.Input.Params[":id"]
	id, _ := strconv.ParseInt(idStr, 0, 64)
	v := models.{{controllerName}}{Id: id}
	json.Unmarshal(c.Ctx.Input.RequestBody, &v)
	if err := models.Update{{controllerName}}ById(&v); err == nil {
		c.Data["json"] = "OK"
	} else {
		c.Data["json"] = err.Error()
	}
	c.ServeJson()
}

// @Title Delete
// @Description delete the {{controllerName}}
// @Param	id		path 	string	true		"The id you want to delete"
// @Success 200 {string} delete success!
// @Failure 403 id is empty
// @router /:id [delete]
func (c *{{controllerName}}Controller) Delete() {
	idStr := c.Ctx.Input.Params[":id"]
	id, _ := strconv.ParseInt(idStr, 0, 64)
	if err := models.Delete{{controllerName}}(id); err == nil {
		c.Data["json"] = "OK"
	} else {
		c.Data["json"] = err.Error()
	}
	c.ServeJson()
}
`

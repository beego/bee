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
	path "path/filepath"
	"strings"
)

var cmdApiapp = &Command{
	// CustomFlags: true,
	UsageLine: "api [appname]",
	Short:     "create an api application base on beego framework",
	Long: `
create an api application base on beego framework

In the current path, will create a folder named [appname]

In the appname folder has the follow struct:

	├── conf
	│   └── app.conf
	├── controllers
	│   └── default.go
	├── routers
	│   └── router.go
	├── tests
	│   └── default_test.go		
	├── main.go
	└── models
	    └── object.go            

`,
}

var apiconf = `
appname = {{.Appname}}
httpport = 8080
runmode = dev
autorender = false
copyrequestbody = true
`
var apiMaingo = `package main

import (
	_ "{{.Appname}}/routers"
	"github.com/astaxie/beego"	
)

//		Objects

//	URL					HTTP Verb				Functionality
//	/object				POST					Creating Objects
//	/object/<objectId>	GET						Retrieving Objects
//	/object/<objectId>	PUT						Updating Objects
//	/object				GET						Queries
//	/object/<objectId>	DELETE					Deleting Objects

func main() {
	beego.Run()
}
`
var apirouter = `package routers

import (
	"{{.Appname}}/controllers"
	"github.com/astaxie/beego"
)

func init() {
    beego.RESTRouter("/object", &controllers.ObjectController{})
}
`

var apiModels = `package models

import (
	"errors"
	"strconv"
	"time"
)

var (
	Objects map[string]*Object
)

type Object struct {
	ObjectId   string
	Score      int64
	PlayerName string
}

func init() {
	Objects = make(map[string]*Object)
	Objects["hjkhsbnmn123"] = &Object{"hjkhsbnmn123", 100, "astaxie"}
	Objects["mjjkxsxsaa23"] = &Object{"mjjkxsxsaa23", 101, "someone"}
}

func AddOne(object Object) (ObjectId string) {
	object.ObjectId = "astaxie" + strconv.FormatInt(time.Now().UnixNano(), 10)
	Objects[object.ObjectId] = &object
	return object.ObjectId
}

func GetOne(ObjectId string) (object *Object, err error) {
	if v, ok := Objects[ObjectId]; ok {
		return v, nil
	}
	return nil, errors.New("ObjectId Not Exist")
}

func GetAll() map[string]*Object {
	return Objects
}

func Update(ObjectId string, Score int64) (err error) {
	if v, ok := Objects[ObjectId]; ok {
		v.Score = Score
		return nil
	}
	return errors.New("ObjectId Not Exist")
}

func Delete(ObjectId string) {
	delete(Objects, ObjectId)
}
`

var apiControllers = `package controllers

import (
	"encoding/json"
	"github.com/astaxie/beego"
	"{{.Appname}}/models"
)

type ResponseInfo struct {
}

type ObjectController struct {
	beego.Controller
}

func (this *ObjectController) Post() {
	var ob models.Object
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	objectid := models.AddOne(ob)
	this.Data["json"] = map[string]string{"ObjectId": objectid}
	this.ServeJson()
}

func (this *ObjectController) Get() {
	objectId := this.Ctx.Input.Params[":objectId"]
	if objectId != "" {
		ob, err := models.GetOne(objectId)
		if err != nil {
			this.Data["json"] = err
		} else {
			this.Data["json"] = ob
		}
	} else {
		obs := models.GetAll()
		this.Data["json"] = obs
	}
	this.ServeJson()
}

func (this *ObjectController) Put() {
	objectId := this.Ctx.Input.Params[":objectId"]
	var ob models.Object
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	err := models.Update(objectId, ob.Score)
	if err != nil {
		this.Data["json"] = err
	} else {
		this.Data["json"] = "update success!"
	}
	this.ServeJson()
}

func (this *ObjectController) Delete() {
	objectId := this.Ctx.Input.Params[":objectId"]
	models.Delete(objectId)
	this.Data["json"] = "delete success!"
	this.ServeJson()
}

`

var apiTests = `package test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	_ "{{.Appname}}/routers"
		
	"github.com/astaxie/beego"
	. "github.com/smartystreets/goconvey/convey"
)

// TestGet is a sample to run an endpoint test
func TestGet(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)
	
	beego.Trace("testing", "TestGet", "Code[%d]\n%s", w.Code, w.Body.String())
	
	Convey("Subject: Test Station Endpoint\n", t, func() {
	        Convey("Status Code Should Be 200", func() {
	                So(w.Code, ShouldEqual, 200)
	        })
	        Convey("The Result Should Not Be Empty", func() {
	                So(w.Body.Len(), ShouldBeGreaterThan, 0)
	        })
	})
}

`

func init() {
	cmdApiapp.Run = createapi
}

func createapi(cmd *Command, args []string) {
	if len(args) != 1 {
		fmt.Println("error args")
		os.Exit(2)
	}
	apppath, packpath, err := checkEnv(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	os.MkdirAll(apppath, 0755)
	fmt.Println("create app folder:", apppath)
	os.Mkdir(path.Join(apppath, "conf"), 0755)
	fmt.Println("create conf:", path.Join(apppath, "conf"))
	os.Mkdir(path.Join(apppath, "controllers"), 0755)
	fmt.Println("create controllers:", path.Join(apppath, "controllers"))
	os.Mkdir(path.Join(apppath, "models"), 0755)
	fmt.Println(path.Join(apppath, "routers") + string(path.Separator))
	os.Mkdir(path.Join(apppath, "routers"), 0755)
	fmt.Println("create models:", path.Join(apppath, "models"))
	os.Mkdir(path.Join(apppath, "tests"), 0755)
	fmt.Println("create tests:", path.Join(apppath, "tests"))

	fmt.Println("create conf app.conf:", path.Join(apppath, "conf", "app.conf"))
	writetofile(path.Join(apppath, "conf", "app.conf"),
		strings.Replace(apiconf, "{{.Appname}}", args[0], -1))

	fmt.Println("create controllers default.go:", path.Join(apppath, "controllers", "default.go"))
	writetofile(path.Join(apppath, "controllers", "default.go"),
		strings.Replace(apiControllers, "{{.Appname}}", packpath, -1))

	fmt.Println("create tests default.go:", path.Join(apppath, "tests", "default_test.go"))
	writetofile(path.Join(apppath, "tests", "default_test.go"),
		strings.Replace(apiTests, "{{.Appname}}", packpath, -1))

	fmt.Println("create routers router.go:", path.Join(apppath, "routers", "router.go"))
	writetofile(path.Join(apppath, "routers", "router.go"),
		strings.Replace(apirouter, "{{.Appname}}", packpath, -1))

	fmt.Println("create models object.go:", path.Join(apppath, "models", "object.go"))
	writetofile(path.Join(apppath, "models", "object.go"), apiModels)

	fmt.Println("create main.go:", path.Join(apppath, "main.go"))
	writetofile(path.Join(apppath, "main.go"),
		strings.Replace(apiMaingo, "{{.Appname}}", packpath, -1))
}

func checkEnv(appname string) (apppath, packpath string, err error) {
	curpath, err := os.Getwd()
	if err != nil {
		return
	}

	gopath := os.Getenv("GOPATH")
	Debugf("gopath:%s", gopath)
	if gopath == "" {
		err = fmt.Errorf("you should set GOPATH in the env")
		return
	}

	appsrcpath := ""
	haspath := false
	wgopath := path.SplitList(gopath)
	for _, wg := range wgopath {
		wg = path.Join(wg, "src")

		if path.HasPrefix(strings.ToLower(curpath), strings.ToLower(wg)) {
			haspath = true
			appsrcpath = wg
			break
		}
	}

	if !haspath {
		err = fmt.Errorf("can't create application outside of GOPATH `%s`\n"+
			"you first should `cd $GOPATH%ssrc` then use create\n", gopath, string(path.Separator))
		return
	}
	apppath = path.Join(curpath, appname)

	if _, e := os.Stat(apppath); os.IsNotExist(e) == false {
		err = fmt.Errorf("path `%s` exists, can not create app without remove it\n", apppath)
		return
	}
	packpath = strings.Join(strings.Split(apppath[len(appsrcpath)+1:], string(path.Separator)), "/")
	return
}

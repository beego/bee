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

var cmdHproseapp = &Command{
	// CustomFlags: true,
	UsageLine: "hprose [appname]",
	Short:     "Creates an RPC application based on Hprose and Beego frameworks",
	Long: `
  The command 'hprose' creates an RPC application based on both Beego and Hprose (http://hprose.com/).

  {{"To scaffold out your application, use:"|bold}}

      $ bee hprose [appname] [-tables=""] [-driver=mysql] [-conn=root:@tcp(127.0.0.1:3306)/test]

  If 'conn' is empty, the command will generate a sample application. Otherwise the command
  will connect to your database and generate models based on the existing tables.

  The command 'hprose' creates a folder named [appname] with the following structure:

	    ├── main.go
	    ├── {{"conf"|foldername}}
	    │     └── app.conf
	    └── {{"models"|foldername}}
	          └── object.go
	          └── user.go
`,
	PreRun: func(cmd *Command, args []string) { ShowShortVersionBanner() },
	Run:    createhprose,
}

var hproseconf = `appname = {{.Appname}}
httpport = 8080
runmode = dev
autorender = false
copyrequestbody = true
EnableDocs = true
`
var hproseMaingo = `package main

import (
	"fmt"
	"reflect"

	"{{.Appname}}/models"
	"github.com/hprose/hprose-golang/rpc"

	"github.com/astaxie/beego"
)

func logInvokeHandler(
	name string,
	args []reflect.Value,
	context rpc.Context,
	next rpc.NextInvokeHandler) (results []reflect.Value, err error) {
	fmt.Printf("%s(%v) = ", name, args)
	results, err = next(name, args, context)
	fmt.Printf("%v %v\r\n", results, err)
	return
}

func main() {
	// Create WebSocketServer
	// service := rpc.NewWebSocketService()

	// Create Http Server
	service := rpc.NewHTTPService()

	// Use Logger Middleware
	service.AddInvokeHandler(logInvokeHandler)

	// Publish Functions
	service.AddFunction("AddOne", models.AddOne)
	service.AddFunction("GetOne", models.GetOne)

	// Start Service
	beego.Handler("/", service)
	beego.Run()
}
`

var hproseMainconngo = `package main

import (
	"fmt"
	"reflect"

	"{{.Appname}}/models"
	"github.com/hprose/hprose-golang/rpc"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	{{.DriverPkg}}
)

func init() {
	orm.RegisterDataBase("default", "{{.DriverName}}", "{{.conn}}")
}

func logInvokeHandler(
	name string,
	args []reflect.Value,
	context rpc.Context,
	next rpc.NextInvokeHandler) (results []reflect.Value, err error) {
	fmt.Printf("%s(%v) = ", name, args)
	results, err = next(name, args, context)
	fmt.Printf("%v %v\r\n", results, err)
	return
}

func main() {
	// Create WebSocketServer
	// service := rpc.NewWebSocketService()

	// Create Http Server
	service := rpc.NewHTTPService()

	// Use Logger Middleware
	service.AddInvokeHandler(logInvokeHandler)

	{{HproseFunctionList}}

	// Start Service
	beego.Handler("/", service)
	beego.Run()
}

`

var hproseModels = `package models

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

var hproseModels2 = `package models

import (
	"errors"
	"strconv"
	"time"
)

var (
	UserList map[string]*User
)

func init() {
	UserList = make(map[string]*User)
	u := User{"user_11111", "astaxie", "11111", Profile{"male", 20, "Singapore", "astaxie@gmail.com"}}
	UserList["user_11111"] = &u
}

type User struct {
	Id       string
	Username string
	Password string
	Profile  Profile
}

type Profile struct {
	Gender  string
	Age     int
	Address string
	Email   string
}

func AddUser(u User) string {
	u.Id = "user_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	UserList[u.Id] = &u
	return u.Id
}

func GetUser(uid string) (u *User, err error) {
	if u, ok := UserList[uid]; ok {
		return u, nil
	}
	return nil, errors.New("User not exists")
}

func GetAllUsers() map[string]*User {
	return UserList
}

func UpdateUser(uid string, uu *User) (a *User, err error) {
	if u, ok := UserList[uid]; ok {
		if uu.Username != "" {
			u.Username = uu.Username
		}
		if uu.Password != "" {
			u.Password = uu.Password
		}
		if uu.Profile.Age != 0 {
			u.Profile.Age = uu.Profile.Age
		}
		if uu.Profile.Address != "" {
			u.Profile.Address = uu.Profile.Address
		}
		if uu.Profile.Gender != "" {
			u.Profile.Gender = uu.Profile.Gender
		}
		if uu.Profile.Email != "" {
			u.Profile.Email = uu.Profile.Email
		}
		return u, nil
	}
	return nil, errors.New("User Not Exist")
}

func Login(username, password string) bool {
	for _, u := range UserList {
		if u.Username == username && u.Password == password {
			return true
		}
	}
	return false
}

func DeleteUser(uid string) {
	delete(UserList, uid)
}
`

var hproseAddFunctions = []string{}

func init() {
	cmdHproseapp.Flag.Var(&tables, "tables", "List of table names separated by a comma.")
	cmdHproseapp.Flag.Var(&driver, "driver", "Database driver. Either mysql, postgres or sqlite.")
	cmdHproseapp.Flag.Var(&conn, "conn", "Connection string used by the driver to connect to a database instance.")
}

func createhprose(cmd *Command, args []string) int {
	output := cmd.Out()

	curpath, _ := os.Getwd()
	if len(args) > 1 {
		cmd.Flag.Parse(args[1:])
	}
	apppath, packpath, err := checkEnv(args[0])
	if err != nil {
		logger.Fatalf("%s", err)
	}
	if driver == "" {
		driver = "mysql"
	}
	if conn == "" {
	}

	logger.Info("Creating Hprose application...")

	os.MkdirAll(apppath, 0755)
	fmt.Fprintf(output, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", apppath, "\x1b[0m")
	os.Mkdir(path.Join(apppath, "conf"), 0755)
	fmt.Fprintf(output, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", path.Join(apppath, "conf"), "\x1b[0m")
	fmt.Fprintf(output, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", path.Join(apppath, "conf", "app.conf"), "\x1b[0m")
	WriteToFile(path.Join(apppath, "conf", "app.conf"),
		strings.Replace(hproseconf, "{{.Appname}}", args[0], -1))

	if conn != "" {
		logger.Infof("Using '%s' as 'driver'", driver)
		logger.Infof("Using '%s' as 'conn'", conn)
		logger.Infof("Using '%s' as 'tables'", tables)
		generateHproseAppcode(string(driver), string(conn), "1", string(tables), path.Join(curpath, args[0]))
		fmt.Fprintf(output, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", path.Join(apppath, "main.go"), "\x1b[0m")
		maingoContent := strings.Replace(hproseMainconngo, "{{.Appname}}", packpath, -1)
		maingoContent = strings.Replace(maingoContent, "{{.DriverName}}", string(driver), -1)
		maingoContent = strings.Replace(maingoContent, "{{HproseFunctionList}}", strings.Join(hproseAddFunctions, ""), -1)
		if driver == "mysql" {
			maingoContent = strings.Replace(maingoContent, "{{.DriverPkg}}", `_ "github.com/go-sql-driver/mysql"`, -1)
		} else if driver == "postgres" {
			maingoContent = strings.Replace(maingoContent, "{{.DriverPkg}}", `_ "github.com/lib/pq"`, -1)
		}
		WriteToFile(path.Join(apppath, "main.go"),
			strings.Replace(
				maingoContent,
				"{{.conn}}",
				conn.String(),
				-1,
			),
		)
	} else {
		os.Mkdir(path.Join(apppath, "models"), 0755)
		fmt.Fprintf(output, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", path.Join(apppath, "models"), "\x1b[0m")

		fmt.Fprintf(output, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", path.Join(apppath, "models", "object.go"), "\x1b[0m")
		WriteToFile(path.Join(apppath, "models", "object.go"), apiModels)

		fmt.Fprintf(output, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", path.Join(apppath, "models", "user.go"), "\x1b[0m")
		WriteToFile(path.Join(apppath, "models", "user.go"), apiModels2)

		fmt.Fprintf(output, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", path.Join(apppath, "main.go"), "\x1b[0m")
		WriteToFile(path.Join(apppath, "main.go"),
			strings.Replace(hproseMaingo, "{{.Appname}}", packpath, -1))
	}
	logger.Success("New Hprose application successfully created!")
	return 0
}

/**********************************************************\
|                                                          |
|                          hprose                          |
|                                                          |
| Official WebSite: http://www.hprose.com/                 |
|                   http://www.hprose.org/                 |
|                                                          |
\**********************************************************/
/**********************************************************\
 *                                                        *
 * Build rpc application use Hprose base on beego         *
 *                                                        *
 * LastModified: Oct 13, 2014                             *
 * Author: Liu jian <laoliu@lanmv.com>                    *
 *                                                        *
\**********************************************************/

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
	Short:     "create an rpc application use hprose base on beego framework",
	Long: `	
create an rpc application use hprose base on beego framework

bee hprose [appname] [-tables=""] [-driver=mysql] [-conn=root:@tcp(127.0.0.1:3306)/test]
    -tables: a list of table names separated by ',', default is empty, indicating all tables
    -driver: [mysql | postgres | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is ''
             e.g. for mysql:    root:@tcp(127.0.0.1:3306)/test
             e.g. for postgres: postgres://postgres:postgres@127.0.0.1:5432/postgres
	
if conn is empty will create a example rpc application. otherwise generate rpc application use hprose based on an existing database.

In the current path, will create a folder named [appname]

In the appname folder has the follow struct:

	├── conf
	│   └── app.conf
	├── main.go
	└── models
	    └── object.go
	    └── user.go
`,
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
	"{{.Appname}}/models"
	"github.com/hprose/hprose-go/hprose"

	"github.com/astaxie/beego"
)

func main() {
	service := hprose.NewHttpService()
	service.AddFunction("AddOne", models.AddOne)
	service.AddFunction("GetOne", models.GetOne)
	beego.Handler("/", service)
	beego.Run()
}
`

var hproseMainconngo = `package main

import (
	"{{.Appname}}/models"
	"github.com/hprose/hprose-go/hprose"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	{{.DriverPkg}}
)

func init() {
	orm.RegisterDataBase("default", "{{.DriverName}}", "{{.conn}}")
}

func main() {
	service := hprose.NewHttpService()
	{{HproseFunctionList}}
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
	cmdHproseapp.Run = createhprose
	cmdHproseapp.Flag.Var(&tables, "tables", "specify tables to generate model")
	cmdHproseapp.Flag.Var(&driver, "driver", "database driver: mysql, postgresql, etc.")
	cmdHproseapp.Flag.Var(&conn, "conn", "connection string used by the driver to connect to a database instance")
}

func createhprose(cmd *Command, args []string) int {
	curpath, _ := os.Getwd()
	if len(args) > 1 {
		cmd.Flag.Parse(args[1:])
	}
	apppath, packpath, err := checkEnv(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	if driver == "" {
		driver = "mysql"
	}
	if conn == "" {
	}
	os.MkdirAll(apppath, 0755)
	fmt.Println("create app folder:", apppath)
	os.Mkdir(path.Join(apppath, "conf"), 0755)
	fmt.Println("create conf:", path.Join(apppath, "conf"))
	fmt.Println("create conf app.conf:", path.Join(apppath, "conf", "app.conf"))
	writetofile(path.Join(apppath, "conf", "app.conf"),
		strings.Replace(hproseconf, "{{.Appname}}", args[0], -1))

	if conn != "" {
		ColorLog("[INFO] Using '%s' as 'driver'\n", driver)
		ColorLog("[INFO] Using '%s' as 'conn'\n", conn)
		ColorLog("[INFO] Using '%s' as 'tables'\n", tables)
		generateHproseAppcode(string(driver), string(conn), "1", string(tables), path.Join(curpath, args[0]))
		fmt.Println("create main.go:", path.Join(apppath, "main.go"))
		maingoContent := strings.Replace(hproseMainconngo, "{{.Appname}}", packpath, -1)
		maingoContent = strings.Replace(maingoContent, "{{.DriverName}}", string(driver), -1)
		maingoContent = strings.Replace(maingoContent, "{{HproseFunctionList}}", strings.Join(hproseAddFunctions, ""), -1)
		if driver == "mysql" {
			maingoContent = strings.Replace(maingoContent, "{{.DriverPkg}}", `_ "github.com/go-sql-driver/mysql"`, -1)
		} else if driver == "postgres" {
			maingoContent = strings.Replace(maingoContent, "{{.DriverPkg}}", `_ "github.com/lib/pq"`, -1)
		}
		writetofile(path.Join(apppath, "main.go"),
			strings.Replace(
				maingoContent,
				"{{.conn}}",
				conn.String(),
				-1,
			),
		)
	} else {
		os.Mkdir(path.Join(apppath, "models"), 0755)
		fmt.Println("create models:", path.Join(apppath, "models"))

		fmt.Println("create models object.go:", path.Join(apppath, "models", "object.go"))
		writetofile(path.Join(apppath, "models", "object.go"), apiModels)

		fmt.Println("create models user.go:", path.Join(apppath, "models", "user.go"))
		writetofile(path.Join(apppath, "models", "user.go"), apiModels2)

		fmt.Println("create main.go:", path.Join(apppath, "main.go"))
		writetofile(path.Join(apppath, "main.go"),
			strings.Replace(hproseMaingo, "{{.Appname}}", packpath, -1))
	}
	return 0
}

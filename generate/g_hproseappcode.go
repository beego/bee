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
 * LastModified: Oct 23, 2014                             *
 * Author: Liu jian <laoliu@lanmv.com>                    *
 *                                                        *
\**********************************************************/

package generate

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"strings"

	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/logger/colors"
	"github.com/beego/bee/utils"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var Hproseconf = `appname = {{.Appname}}
httpport = 8080
runmode = dev
autorender = false
copyrequestbody = true
EnableDocs = true
`
var HproseMaingo = `package main

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

var HproseMainconngo = `package main

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

var HproseModels = `package models

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

var HproseModels2 = `package models

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
var HproseAddFunctions = []string{}

func GenerateHproseAppcode(driver, connStr, level, tables, currpath string) {
	var mode byte
	switch level {
	case "1":
		mode = OModel
	case "2":
		mode = OModel | OController
	case "3":
		mode = OModel | OController | ORouter
	default:
		beeLogger.Log.Fatal("Invalid 'level' option. Level must be either \"1\", \"2\" or \"3\"")
	}
	var selectedTables map[string]bool
	if tables != "" {
		selectedTables = make(map[string]bool)
		for _, v := range strings.Split(tables, ",") {
			selectedTables[v] = true
		}
	}
	switch driver {
	case "mysql":
	case "postgres":
	case "sqlite":
		beeLogger.Log.Fatal("Generating app code from SQLite database is not supported yet")
	default:
		beeLogger.Log.Fatalf("Unknown database driver '%s'. Driver must be one of mysql, postgres or sqlite", driver)
	}
	genHprose(driver, connStr, mode, selectedTables, currpath)
}

// Generate takes table, column and foreign key information from database connection
// and generate corresponding golang source files
func genHprose(dbms, connStr string, mode byte, selectedTableNames map[string]bool, currpath string) {
	db, err := sql.Open(dbms, connStr)
	if err != nil {
		beeLogger.Log.Fatalf("Could not connect to '%s' database using '%s': %s", dbms, connStr, err)
	}
	defer db.Close()
	if trans, ok := dbDriver[dbms]; ok {
		beeLogger.Log.Info("Analyzing database tables...")
		tableNames := trans.GetTableNames(db)
		tables := getTableObjects(tableNames, db, trans)
		mvcPath := new(MvcPath)
		mvcPath.ModelPath = path.Join(currpath, "models")
		createPaths(mode, mvcPath)
		pkgPath := getPackagePath(currpath)
		writeHproseSourceFiles(pkgPath, tables, mode, mvcPath, selectedTableNames)
	} else {
		beeLogger.Log.Fatalf("Generating app code from '%s' database is not supported yet", dbms)
	}
}

// writeHproseSourceFiles generates source files for model/controller/router
// It will wipe the following directories and recreate them:./models, ./controllers, ./routers
// Newly geneated files will be inside these folders.
func writeHproseSourceFiles(pkgPath string, tables []*Table, mode byte, paths *MvcPath, selectedTables map[string]bool) {
	if (OModel & mode) == OModel {
		beeLogger.Log.Info("Creating model files...")
		writeHproseModelFiles(tables, paths.ModelPath, selectedTables)
	}
}

// writeHproseModelFiles generates model files
func writeHproseModelFiles(tables []*Table, mPath string, selectedTables map[string]bool) {
	w := colors.NewColorWriter(os.Stdout)

	for _, tb := range tables {
		// if selectedTables map is not nil and this table is not selected, ignore it
		if selectedTables != nil {
			if _, selected := selectedTables[tb.Name]; !selected {
				continue
			}
		}
		filename := getFileName(tb.Name)
		fpath := path.Join(mPath, filename+".go")
		var f *os.File
		var err error
		if utils.IsExist(fpath) {
			beeLogger.Log.Warnf("'%s' already exists. Do you want to overwrite it? [Yes|No] ", fpath)
			if utils.AskForConfirmation() {
				f, err = os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0666)
				if err != nil {
					beeLogger.Log.Warnf("%s", err)
					continue
				}
			} else {
				beeLogger.Log.Warnf("Skipped create file '%s'", fpath)
				continue
			}
		} else {
			f, err = os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, 0666)
			if err != nil {
				beeLogger.Log.Warnf("%s", err)
				continue
			}
		}
		var template string
		if tb.Pk == "" {
			template = HproseStructModelTPL
		} else {
			template = HproseModelTPL
			HproseAddFunctions = append(HproseAddFunctions, strings.Replace(HproseAddFunction, "{{modelName}}", utils.CamelCase(tb.Name), -1))
		}
		fileStr := strings.Replace(template, "{{modelStruct}}", tb.String(), 1)
		fileStr = strings.Replace(fileStr, "{{modelName}}", utils.CamelCase(tb.Name), -1)
		// if table contains time field, import time.Time package
		timePkg := ""
		importTimePkg := ""
		if tb.ImportTimePkg {
			timePkg = "\"time\"\n"
			importTimePkg = "import \"time\"\n"
		}
		fileStr = strings.Replace(fileStr, "{{timePkg}}", timePkg, -1)
		fileStr = strings.Replace(fileStr, "{{importTimePkg}}", importTimePkg, -1)
		if _, err := f.WriteString(fileStr); err != nil {
			beeLogger.Log.Fatalf("Could not write model file to '%s'", fpath)
		}
		utils.CloseFile(f)
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", fpath, "\x1b[0m")
		utils.FormatSourceCode(fpath)
	}
}

const (
	HproseAddFunction = `
	// publish about {{modelName}} function
	service.AddFunction("Add{{modelName}}", models.Add{{modelName}})
	service.AddFunction("Get{{modelName}}ById", models.Get{{modelName}}ById)
	service.AddFunction("GetAll{{modelName}}", models.GetAll{{modelName}})
	service.AddFunction("Update{{modelName}}ById", models.Update{{modelName}}ById)
	service.AddFunction("Delete{{modelName}}", models.Delete{{modelName}})

`
	HproseStructModelTPL = `package models
{{importTimePkg}}
{{modelStruct}}
`

	HproseModelTPL = `package models

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	{{timePkg}}
	"github.com/astaxie/beego/orm"
)

{{modelStruct}}

func init() {
	orm.RegisterModel(new({{modelName}}))
}

// Add{{modelName}} insert a new {{modelName}} into database and returns
// last inserted Id on success.
func Add{{modelName}}(m *{{modelName}}) (id int64, err error) {
	o := orm.NewOrm()
	id, err = o.Insert(m)
	return
}

// Get{{modelName}}ById retrieves {{modelName}} by Id. Returns error if
// Id doesn't exist
func Get{{modelName}}ById(id int) (v *{{modelName}}, err error) {
	o := orm.NewOrm()
	v = &{{modelName}}{Id: id}
	if err = o.Read(v); err == nil {
		return v, nil
	}
	return nil, err
}

// GetAll{{modelName}} retrieves all {{modelName}} matches certain condition. Returns empty list if
// no records exist
func GetAll{{modelName}}(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable(new({{modelName}}))
	// query k=v
	for k, v := range query {
		// rewrite dot-notation to Object__Attribute
		k = strings.Replace(k, ".", "__", -1)
		qs = qs.Filter(k, v)
	}
	// order by:
	var sortFields []string
	if len(sortby) != 0 {
		if len(sortby) == len(order) {
			// 1) for each sort field, there is an associated order
			for i, v := range sortby {
				orderby := ""
				if order[i] == "desc" {
					orderby = "-" + v
				} else if order[i] == "asc" {
					orderby = v
				} else {
					return nil, errors.New("Error: Invalid order. Must be either [asc|desc]")
				}
				sortFields = append(sortFields, orderby)
			}
			qs = qs.OrderBy(sortFields...)
		} else if len(sortby) != len(order) && len(order) == 1 {
			// 2) there is exactly one order, all the sorted fields will be sorted by this order
			for _, v := range sortby {
				orderby := ""
				if order[0] == "desc" {
					orderby = "-" + v
				} else if order[0] == "asc" {
					orderby = v
				} else {
					return nil, errors.New("Error: Invalid order. Must be either [asc|desc]")
				}
				sortFields = append(sortFields, orderby)
			}
		} else if len(sortby) != len(order) && len(order) != 1 {
			return nil, errors.New("Error: 'sortby', 'order' sizes mismatch or 'order' size is not 1")
		}
	} else {
		if len(order) != 0 {
			return nil, errors.New("Error: unused 'order' fields")
		}
	}

	var l []{{modelName}}
	qs = qs.OrderBy(sortFields...)
	if _, err = qs.Limit(limit, offset).All(&l, fields...); err == nil {
		if len(fields) == 0 {
			for _, v := range l {
				ml = append(ml, v)
			}
		} else {
			// trim unused fields
			for _, v := range l {
				m := make(map[string]interface{})
				val := reflect.ValueOf(v)
				for _, fname := range fields {
					m[fname] = val.FieldByName(fname).Interface()
				}
				ml = append(ml, m)
			}
		}
		return ml, nil
	}
	return nil, err
}

// Update{{modelName}} updates {{modelName}} by Id and returns error if
// the record to be updated doesn't exist
func Update{{modelName}}ById(m *{{modelName}}) (err error) {
	o := orm.NewOrm()
	v := {{modelName}}{Id: m.Id}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Update(m); err == nil {
			fmt.Println("Number of records updated in database:", num)
		}
	}
	return
}

// Delete{{modelName}} deletes {{modelName}} by Id and returns error if
// the record to be deleted doesn't exist
func Delete{{modelName}}(id int) (err error) {
	o := orm.NewOrm()
	v := {{modelName}}{Id: id}
	// ascertain id exists in the database
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Delete(&{{modelName}}{Id: id}); err == nil {
			fmt.Println("Number of records deleted in database:", num)
		}
	}
	return
}
`
)

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

package main

import (
	"database/sql"
	"os"
	"path"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func generateHproseAppcode(driver, connStr, level, tables, currpath string) {
	var mode byte
	switch level {
	case "1":
		mode = O_MODEL
	case "2":
		mode = O_MODEL | O_CONTROLLER
	case "3":
		mode = O_MODEL | O_CONTROLLER | O_ROUTER
	default:
		ColorLog("[ERRO] Invalid 'level' option: %s\n", level)
		ColorLog("[HINT] Level must be either 1, 2 or 3\n")
		os.Exit(2)
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
		ColorLog("[ERRO] Generating app code from SQLite database is not supported yet.\n")
		os.Exit(2)
	default:
		ColorLog("[ERRO] Unknown database driver: %s\n", driver)
		ColorLog("[HINT] Driver must be one of mysql, postgres or sqlite\n")
		os.Exit(2)
	}
	genHprose(driver, connStr, mode, selectedTables, currpath)
}

// Generate takes table, column and foreign key information from database connection
// and generate corresponding golang source files
func genHprose(dbms, connStr string, mode byte, selectedTableNames map[string]bool, currpath string) {
	db, err := sql.Open(dbms, connStr)
	if err != nil {
		ColorLog("[ERRO] Could not connect to %s database: %s, %s\n", dbms, connStr, err)
		os.Exit(2)
	}
	defer db.Close()
	if trans, ok := dbDriver[dbms]; ok {
		ColorLog("[INFO] Analyzing database tables...\n")
		tableNames := trans.GetTableNames(db)
		tables := getTableObjects(tableNames, db, trans)
		mvcPath := new(MvcPath)
		mvcPath.ModelPath = path.Join(currpath, "models")
		createPaths(mode, mvcPath)
		pkgPath := getPackagePath(currpath)
		writeHproseSourceFiles(pkgPath, tables, mode, mvcPath, selectedTableNames)
	} else {
		ColorLog("[ERRO] Generating app code from %s database is not supported yet.\n", dbms)
		os.Exit(2)
	}
}

// writeHproseSourceFiles generates source files for model/controller/router
// It will wipe the following directories and recreate them:./models, ./controllers, ./routers
// Newly geneated files will be inside these folders.
func writeHproseSourceFiles(pkgPath string, tables []*Table, mode byte, paths *MvcPath, selectedTables map[string]bool) {
	if (O_MODEL & mode) == O_MODEL {
		ColorLog("[INFO] Creating model files...\n")
		writeHproseModelFiles(tables, paths.ModelPath, selectedTables)
	}
}

// writeHproseModelFiles generates model files
func writeHproseModelFiles(tables []*Table, mPath string, selectedTables map[string]bool) {
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
		if isExist(fpath) {
			ColorLog("[WARN] %v is exist, do you want to overwrite it? Yes or No?\n", fpath)
			if askForConfirmation() {
				f, err = os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0666)
				if err != nil {
					ColorLog("[WARN] %v\n", err)
					continue
				}
			} else {
				ColorLog("[WARN] skip create file\n")
				continue
			}
		} else {
			f, err = os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, 0666)
			if err != nil {
				ColorLog("[WARN] %v\n", err)
				continue
			}
		}
		template := ""
		if tb.Pk == "" {
			template = HPROSE_STRUCT_MODEL_TPL
		} else {
			template = HPROSE_MODEL_TPL
			hproseAddFunctions = append(hproseAddFunctions, strings.Replace(HPROSE_ADDFUNCTION, "{{modelName}}", camelCase(tb.Name), -1))
		}
		fileStr := strings.Replace(template, "{{modelStruct}}", tb.String(), 1)
		fileStr = strings.Replace(fileStr, "{{modelName}}", camelCase(tb.Name), -1)
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
			ColorLog("[ERRO] Could not write model file to %s\n", fpath)
			os.Exit(2)
		}
		f.Close()
		ColorLog("[INFO] model => %s\n", fpath)
		formatSourceCode(fpath)
	}
}

const (
	HPROSE_ADDFUNCTION = `
	// publish about {{modelName}} function
	service.AddFunction("Add{{modelName}}", models.Add{{modelName}})
	service.AddFunction("Get{{modelName}}ById", models.Get{{modelName}}ById)
	service.AddFunction("GetAll{{modelName}}", models.GetAll{{modelName}})
	service.AddFunction("Update{{modelName}}ById", models.Update{{modelName}}ById)
	service.AddFunction("Delete{{modelName}}", models.Delete{{modelName}})

`
	HPROSE_STRUCT_MODEL_TPL = `package models
{{importTimePkg}}
{{modelStruct}}
`

	HPROSE_MODEL_TPL = `package models

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
	if _, err := qs.Limit(limit, offset).All(&l, fields...); err == nil {
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

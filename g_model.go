package main

import (
	"os"
	"path"
	"strings"
)

func generateModel(mname, fields, crupath string) {
	p, f := path.Split(mname)
	modelName := strings.Title(f)
	packageName := "models"
	if p != "" {
		i := strings.LastIndex(p[:len(p)-1], "/")
		packageName = p[i+1 : len(p)-1]
	}
	ColorLog("[INFO] Using '%s' as model name\n", modelName)
	ColorLog("[INFO] Using '%s' as package name\n", packageName)
	fp := path.Join(crupath, "models", p)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		// create controller directory
		if err := os.MkdirAll(fp, 0777); err != nil {
			ColorLog("[ERRO] Could not create models directory: %s\n", err)
			os.Exit(2)
		}
	}
	fpath := path.Join(fp, strings.ToLower(modelName)+".go")
	if f, err := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer f.Close()
		var content string
		if fields != "" {
			generateStructure(modelName,fields,crupath)
			content = strings.Replace(CRUD_MODEL_TPL, "{{packageName}}", packageName, -1)
			pkgPath := getPackagePath(crupath)
			content = strings.Replace(content, "{{pkgPath}}", pkgPath, -1)
		} else {
			content = strings.Replace(BASE_MODEL_TPL, "{{packageName}}", packageName, -1)
		}
		content = strings.Replace(content, "{{modelName}}", modelName, -1)
		f.WriteString(content)
		// gofmt generated source code
		formatSourceCode(fpath)
		ColorLog("[INFO] model file generated: %s\n", fpath)
	} else {
		// error creating file
		ColorLog("[ERRO] Could not create model file: %s\n", err)
		os.Exit(2)
	}
}

const (
	BASE_MODEL_TPL = `package {{packageName}}

	// Add{{modelName}} insert a new {{modelName}} into database and returns
	// last inserted Id on success.
	func Add{{modelName}}() () {

	}

	// Get{{modelName}}ById retrieves {{modelName}} by Id. Returns error if
	// Id doesn't exist
	func Get{{modelName}}ById() () {

	}

	// GetAll{{modelName}} retrieves all {{modelName}} matches certain condition. Returns empty list if
	// no records exist
	func GetAll{{modelName}}() () {

	}

	// Update{{modelName}} updates {{modelName}} by Id and returns error if
	// the record to be updated doesn't exist
	func Update{{modelName}}ById() () {

	}

	// Delete{{modelName}} deletes {{modelName}} by Id and returns error if
	// the record to be deleted doesn't exist
	func Delete{{modelName}}() () {

	}
	`
	CRUD_MODEL_TPL = `package {{packageName}}

	import (
		"{{pkgPath}}/structures"
		"errors"
		"fmt"
		"reflect"
		"strings"

		"github.com/astaxie/beego/orm"
	)

	// Add{{modelName}} insert a new {{modelName}} into database and returns
	// last inserted Id on success.
	func Add{{modelName}}(m *structures.{{modelName}}) (id int64, err error) {
		o := orm.NewOrm()
		id, err = o.Insert(m)
		return
	}

	// Get{{modelName}}ById retrieves {{modelName}} by Id. Returns error if
	// Id doesn't exist
	func Get{{modelName}}ById(id int64) (v *structures.{{modelName}}, err error) {
		o := orm.NewOrm()
		v = &structures.{{modelName}}{Id: id}
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
		qs := o.QueryTable(new(structures.{{modelName}}))
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

		var l []structures.{{modelName}}
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
	func Update{{modelName}}ById(m *structures.{{modelName}}) (err error) {
		o := orm.NewOrm()
		v := structures.{{modelName}}{Id: m.Id}
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
	func Delete{{modelName}}(id int64) (err error) {
		o := orm.NewOrm()
		v := structures.{{modelName}}{Id: id}
		// ascertain id exists in the database
		if err = o.Read(&v); err == nil {
			var num int64
			if num, err = o.Delete(&structures.{{modelName}}{Id: id}); err == nil {
				fmt.Println("Number of records deleted in database:", num)
			}
		}
		return
	}
	`
)

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
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

const (
	O_MODEL byte = 1 << iota
	O_CONTROLLER
	O_ROUTER
)

type MvcPath struct {
	ModelPath      string
	ControllerPath string
	RouterPath     string
}

// typeMapping maps a SQL data type to its corresponding Go data type
var typeMapping = map[string]string{
	"int":                "int", // int signed
	"integer":            "int",
	"tinyint":            "int8",
	"smallint":           "int16",
	"mediumint":          "int32",
	"bigint":             "int64",
	"int unsigned":       "uint", // int unsigned
	"integer unsigned":   "uint",
	"tinyint unsigned":   "uint8",
	"smallint unsigned":  "uint16",
	"mediumint unsigned": "uint32",
	"bigint unsigned":    "uint64",
	"bool":               "bool",   // boolean
	"enum":               "string", // enum
	"set":                "string", // set
	"varchar":            "string", // string & text
	"char":               "string",
	"tinytext":           "string",
	"mediumtext":         "string",
	"text":               "string",
	"longtext":           "string",
	"blob":               "string", // blob
	"longblob":           "string",
	"date":               "time.Time", // time
	"datetime":           "time.Time",
	"timestamp":          "time.Time",
	"float":              "float32", // float & decimal
	"double":             "float64",
	"decimal":            "float64",
}

// Table represent a table in a database
type Table struct {
	Name    string
	Pk      string
	Uk      []string
	Fk      map[string]*ForeignKey
	Columns []*Column
}

// Column reprsents a column for a table
type Column struct {
	Name string
	Type string
	Tag  *OrmTag
}

// ForeignKey represents a foreign key column for a table
type ForeignKey struct {
	Name      string
	RefSchema string
	RefTable  string
	RefColumn string
}

// OrmTag contains Beego ORM tag information for a column
type OrmTag struct {
	Auto        bool
	Pk          bool
	Null        bool
	Index       bool
	Unique      bool
	Column      string
	Size        string
	Decimals    string
	Digits      string
	AutoNow     bool
	AutoNowAdd  bool
	Type        string
	Default     string
	RelOne      bool
	ReverseOne  bool
	RelFk       bool
	ReverseMany bool
	RelM2M      bool
}

// String returns the source code string for the Table struct
func (tb *Table) String() string {
	rv := fmt.Sprintf("type %s struct {\n", camelCase(tb.Name))
	for _, v := range tb.Columns {
		rv += v.String() + "\n"
	}
	rv += "}\n"
	return rv
}

// String returns the source code string of a field in Table struct
// It maps to a column in database table. e.g. Id int `orm:"column(id);auto"`
func (col *Column) String() string {
	return fmt.Sprintf("%s %s %s", col.Name, col.Type, col.Tag.String())
}

// String returns the ORM tag string for a column
func (tag *OrmTag) String() string {
	var ormOptions []string
	if tag.Column != "" {
		ormOptions = append(ormOptions, fmt.Sprintf("column(%s)", tag.Column))
	}
	if tag.Auto {
		ormOptions = append(ormOptions, "auto")
	}
	if tag.Size != "" {
		ormOptions = append(ormOptions, fmt.Sprintf("size(%s)", tag.Size))
	}
	if tag.Type != "" {
		ormOptions = append(ormOptions, fmt.Sprintf("type(%s)", tag.Type))
	}
	if tag.Null {
		ormOptions = append(ormOptions, "null")
	}
	if tag.AutoNow {
		ormOptions = append(ormOptions, "auto_now")
	}
	if tag.AutoNowAdd {
		ormOptions = append(ormOptions, "auto_now_add")
	}
	if tag.Decimals != "" {
		ormOptions = append(ormOptions, fmt.Sprintf("digits(%s);decimals(%s)", tag.Digits, tag.Decimals))
	}
	if tag.RelFk {
		ormOptions = append(ormOptions, "rel(fk)")
	}
	if tag.RelOne {
		ormOptions = append(ormOptions, "rel(one)")
	}
	if tag.ReverseOne {
		ormOptions = append(ormOptions, "reverse(one)")
	}
	if tag.ReverseMany {
		ormOptions = append(ormOptions, "reverse(many)")
	}
	if tag.RelM2M {
		ormOptions = append(ormOptions, "rel(m2m)")
	}
	if tag.Pk {
		ormOptions = append(ormOptions, "pk")
	}
	if tag.Unique {
		ormOptions = append(ormOptions, "unique")
	}
	if tag.Default != "" {
		ormOptions = append(ormOptions, fmt.Sprintf("default(%s)", tag.Default))
	}

	if len(ormOptions) == 0 {
		return ""
	}
	return fmt.Sprintf("`orm:\"%s\"`", strings.Join(ormOptions, ";"))
}

func generateModel(driver string, connStr string, level string, currpath string) {
	var mode byte
	if level == "1" {
		mode = O_MODEL
	} else if level == "2" {
		mode = O_MODEL | O_CONTROLLER
	} else if level == "3" {
		mode = O_MODEL | O_CONTROLLER | O_ROUTER
	} else {
		ColorLog("[ERRO] Invalid 'level' option: %s\n", level)
		ColorLog("[HINT] Level must be either 1, 2 or 3\n")
		os.Exit(2)
	}
	gen(driver, connStr, mode, currpath)
}

// Generate takes table, column and foreign key information from database connection
// and generate corresponding golang source files
func gen(dbms string, connStr string, mode byte, currpath string) {
	db, err := sql.Open(dbms, connStr)
	if err != nil {
		ColorLog("[ERRO] Could not connect to %s: %s\n", dbms, connStr)
		os.Exit(2)
	}
	defer db.Close()
	ColorLog("[INFO] Analyzing database tables...\n")
	tableNames := getTableNames(db)
	tables := getTableObjects(tableNames, db)
	mvcPath := new(MvcPath)
	mvcPath.ModelPath = path.Join(currpath, "models")
	mvcPath.ControllerPath = path.Join(currpath, "controllers")
	mvcPath.RouterPath = path.Join(currpath, "routers")
	deleteAndRecreatePaths(mode, mvcPath)
	writeSourceFiles(tables, mode, mvcPath)
}

// getTables gets a list table names in current database
func getTableNames(db *sql.DB) (tables []string) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		ColorLog("[ERRO] Could not show tables\n")
		ColorLog("[HINT] Check your connection string\n")
		os.Exit(2)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			ColorLog("[ERRO] Could not show tables\n")
			os.Exit(2)
		}
		tables = append(tables, name)
	}
	return
}

// getTableObjects process each table name
func getTableObjects(tableNames []string, db *sql.DB) (tables []*Table) {
	// if a table has a composite pk or doesn't have pk, we can't use it yet
	// these tables will be put into blacklist so that other struct will not
	// reference it.
	blackList := make(map[string]bool)
	// process constraints information for each table, also gather blacklisted table names
	for _, tableName := range tableNames {
		// create a table struct
		tb := new(Table)
		tb.Name = tableName
		tb.Fk = make(map[string]*ForeignKey)
		getConstraints(db, tb, blackList)
		tables = append(tables, tb)
	}
	// process columns, ignoring blacklisted tables
	for _, tb := range tables {
		getColumns(db, tb, blackList)
	}
	return
}

// getConstraints gets primary key, unique key and foreign keys of a table from information_schema
// and fill in Table struct
func getConstraints(db *sql.DB, table *Table, blackList map[string]bool) {
	rows, err := db.Query(
		`SELECT 
			c.constraint_type, u.column_name, u.referenced_table_schema, u.referenced_table_name, referenced_column_name, u.ordinal_position
		FROM
			information_schema.table_constraints c 
		INNER JOIN
			information_schema.key_column_usage u ON c.constraint_name = u.constraint_name 
		WHERE
			c.table_schema = database() AND c.table_name = ? AND u.table_schema = database() AND u.table_name = ?`,
		table.Name, table.Name) //  u.position_in_unique_constraint,
	if err != nil {
		ColorLog("[ERRO] Could not query INFORMATION_SCHEMA for PK/UK/FK information\n")
		os.Exit(2)
	}
	for rows.Next() {
		var constraintTypeBytes, columnNameBytes, refTableSchemaBytes, refTableNameBytes, refColumnNameBytes, refOrdinalPosBytes []byte
		if err := rows.Scan(&constraintTypeBytes, &columnNameBytes, &refTableSchemaBytes, &refTableNameBytes, &refColumnNameBytes, &refOrdinalPosBytes); err != nil {
			ColorLog("[ERRO] Could not read INFORMATION_SCHEMA for PK/UK/FK information\n")
			os.Exit(2)
		}
		constraintType, columnName, refTableSchema, refTableName, refColumnName, refOrdinalPos :=
			string(constraintTypeBytes), string(columnNameBytes), string(refTableSchemaBytes),
			string(refTableNameBytes), string(refColumnNameBytes), string(refOrdinalPosBytes)
		if constraintType == "PRIMARY KEY" {
			if refOrdinalPos == "1" {
				table.Pk = columnName
			} else {
				table.Pk = ""
				// add table to blacklist so that other struct will not reference it, because we are not
				// registering blacklisted tables
				blackList[table.Name] = true
			}
		} else if constraintType == "UNIQUE" {
			table.Uk = append(table.Uk, columnName)
		} else if constraintType == "FOREIGN KEY" {
			fk := new(ForeignKey)
			fk.Name = columnName
			fk.RefSchema = refTableSchema
			fk.RefTable = refTableName
			fk.RefColumn = refColumnName
			table.Fk[columnName] = fk
		}
	}
}

// getColumns retrieve columns details from information_schema
// and fill in the Column struct
func getColumns(db *sql.DB, table *Table, blackList map[string]bool) {
	// retrieve columns
	colDefRows, _ := db.Query(
		`SELECT
			column_name, data_type, column_type, is_nullable, column_default, extra 
		FROM
			information_schema.columns 
		WHERE
			table_schema = database() AND table_name = ?`,
		table.Name)
	defer colDefRows.Close()
	for colDefRows.Next() {
		// datatype as bytes so that SQL <null> values can be retrieved
		var colNameBytes, dataTypeBytes, columnTypeBytes, isNullableBytes, columnDefaultBytes, extraBytes []byte
		if err := colDefRows.Scan(&colNameBytes, &dataTypeBytes, &columnTypeBytes, &isNullableBytes, &columnDefaultBytes, &extraBytes); err != nil {
			ColorLog("[ERRO] Could not query INFORMATION_SCHEMA for column information\n")
			os.Exit(2)
		}
		colName, dataType, columnType, isNullable, columnDefault, extra :=
			string(colNameBytes), string(dataTypeBytes), string(columnTypeBytes), string(isNullableBytes), string(columnDefaultBytes), string(extraBytes)
		// create a column
		col := new(Column)
		col.Name = camelCase(colName)
		col.Type = getGoDataType(dataType)
		// Tag info
		tag := new(OrmTag)
		tag.Column = colName
		if table.Pk == colName {
			col.Name = "Id"
			col.Type = "int"
			if extra == "auto_increment" {
				tag.Auto = true
			} else {
				tag.Pk = true
			}
		} else {
			fkCol, isFk := table.Fk[colName]
			isBl := false
			if isFk {
				_, isBl = blackList[fkCol.RefTable]
			}
			// check if the current column is a foreign key
			if isFk && !isBl {
				tag.RelFk = true
				refStructName := fkCol.RefTable
				col.Name = camelCase(colName)
				col.Type = "*" + camelCase(refStructName)
			} else {
				// if the name of column is Id, and it's not primary key
				if colName == "id" {
					col.Name = "Id_RENAME"
				}
				if isNullable == "YES" {
					tag.Null = true
				}
				if isSQLSignedIntType(dataType) {
					sign := extractIntSignness(columnType)
					if sign == "unsigned" && extra != "auto_increment" {
						col.Type = getGoDataType(dataType + " " + sign)
					}
				}
				if isSQLStringType(dataType) {
					tag.Size = extractColSize(columnType)
				}
				if isSQLTemporalType(dataType) {
					tag.Type = dataType
					//check auto_now, auto_now_add
					if columnDefault == "CURRENT_TIMESTAMP" && extra == "on update CURRENT_TIMESTAMP" {
						tag.AutoNow = true
					} else if columnDefault == "CURRENT_TIMESTAMP" {
						tag.AutoNowAdd = true
					}
				}
				if isSQLDecimal(dataType) {
					tag.Digits, tag.Decimals = extractDecimal(columnType)
				}
			}
		}
		col.Tag = tag
		table.Columns = append(table.Columns, col)
	}
}

// deleteAndRecreatePaths removes several directories completely
func deleteAndRecreatePaths(mode byte, paths *MvcPath) {
	if (mode & O_MODEL) == O_MODEL {
		//os.RemoveAll(paths.ModelPath)
		os.Mkdir(paths.ModelPath, 0777)
	}
	if (mode & O_CONTROLLER) == O_CONTROLLER {
		//os.RemoveAll(paths.ControllerPath)
		os.Mkdir(paths.ControllerPath, 0777)
	}
	if (mode & O_ROUTER) == O_ROUTER {
		//os.RemoveAll(paths.RouterPath)
		os.Mkdir(paths.RouterPath, 0777)
	}
}

// writeSourceFiles generates source files for model/controller/router
// It will wipe the following directories and recreate them:./models, ./controllers, ./routers
// Newly geneated files will be inside these folders.
func writeSourceFiles(tables []*Table, mode byte, paths *MvcPath) {
	if (O_MODEL & mode) == O_MODEL {
		ColorLog("[INFO] Creating model files...\n")
		writeModelFiles(tables, paths.ModelPath)
	}
	if (O_CONTROLLER & mode) == O_CONTROLLER {
		ColorLog("[INFO] Creating controller files...\n")
		writeControllerFiles(tables, paths.ControllerPath)
	}
	if (O_ROUTER & mode) == O_ROUTER {
		ColorLog("[INFO] Creating router files...\n")
		writeRouterFile(tables, paths.RouterPath)
	}
}

// writeModelFiles generates model files
func writeModelFiles(tables []*Table, mPath string) {
	for _, tb := range tables {
		filename := getFileName(tb.Name)
		fpath := path.Join(mPath, filename+".go")
		f, _ := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
		defer f.Close()
		template := ""
		if tb.Pk == "" {
			template = STRUCT_MODEL_TPL
		} else {
			template = MODEL_TPL
		}
		fileStr := strings.Replace(template, "{{modelStruct}}", tb.String(), 1)
		fileStr = strings.Replace(fileStr, "{{modelName}}", camelCase(tb.Name), -1)
		if _, err := f.WriteString(fileStr); err != nil {
			ColorLog("[ERRO] Could not write model file to %s\n", fpath)
			os.Exit(2)
		}
		ColorLog("[INFO] model => %s\n", fpath)
		formatAndFixImports(fpath)
	}
}

// writeControllerFiles generates controller files
func writeControllerFiles(tables []*Table, cPath string) {
	for _, tb := range tables {
		if tb.Pk == "" {
			continue
		}
		filename := getFileName(tb.Name)
		fpath := path.Join(cPath, filename+".go")
		f, _ := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
		defer f.Close()
		fileStr := strings.Replace(CTRL_TPL, "{{ctrlName}}", camelCase(tb.Name), -1)
		if _, err := f.WriteString(fileStr); err != nil {
			ColorLog("[ERRO] Could not write controller file to %s\n", fpath)
			os.Exit(2)
		}
		ColorLog("[INFO] controller => %s\n", fpath)
		formatAndFixImports(fpath)
	}
}

// writeRouterFile generates router file
func writeRouterFile(tables []*Table, rPath string) {
	var nameSpaces []string
	for _, tb := range tables {
		if tb.Pk == "" {
			continue
		}
		// add name spaces
		nameSpace := strings.Replace(NAMESPACE_TPL, "{{nameSpace}}", tb.Name, -1)
		nameSpace = strings.Replace(nameSpace, "{{ctrlName}}", camelCase(tb.Name), -1)
		nameSpaces = append(nameSpaces, nameSpace)
	}
	// add export controller
	fpath := path.Join(rPath, "router.go")
	routerStr := strings.Replace(ROUTER_TPL, "{{nameSpaces}}", strings.Join(nameSpaces, ""), 1)
	f, _ := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
	defer f.Close()
	if _, err := f.WriteString(routerStr); err != nil {
		ColorLog("[ERRO] Could not write router file to %s\n", fpath)
		os.Exit(2)
	}
	ColorLog("[INFO] router => %s\n", fpath)
	formatAndFixImports(fpath)
}

// formatAndFixImports formats source files (add imports, too)
func formatAndFixImports(filename string) {
	cmd := exec.Command("goimports", "-w", filename)
	cmd.Run()
}

// camelCase converts a _ delimited string to camel case
// e.g. very_important_person => VeryImportantPerson
func camelCase(in string) string {
	tokens := strings.Split(in, "_")
	for i := range tokens {
		tokens[i] = strings.ToUpper(tokens[i][:1]) + tokens[i][1:]
	}
	return strings.Join(tokens, "")
}

// getGoDataType maps an SQL data type to Golang data type
func getGoDataType(sqlType string) (goType string) {
	if v, ok := typeMapping[sqlType]; ok {
		return v
	} else {
		fmt.Println("Error:", sqlType, "not found!")
		os.Exit(1)
	}
	return goType
}

func isSQLTemporalType(t string) bool {
	return t == "date" || t == "datetime" || t == "timestamp"
}

func isSQLStringType(t string) bool {
	return t == "char" || t == "varchar"
}

func isSQLSignedIntType(t string) bool {
	return t == "int" || t == "tinyint" || t == "smallint" || t == "mediumint" || t == "bigint"
}

func isSQLDecimal(t string) bool {
	return t == "decimal"
}

// extractColSize extracts field size: e.g. varchar(255) => 255
func extractColSize(colType string) string {
	regex := regexp.MustCompile(`^[a-z]+\(([0-9]+)\)$`)
	size := regex.FindStringSubmatch(colType)
	return size[1]
}

func extractIntSignness(colType string) string {
	regex := regexp.MustCompile(`(int|smallint|mediumint|bigint)\([0-9]+\)(.*)`)
	signRegex := regex.FindStringSubmatch(colType)
	return strings.Trim(signRegex[2], " ")
}

func extractDecimal(colType string) (digits string, decimals string) {
	decimalRegex := regexp.MustCompile(`decimal\(([0-9]+),([0-9]+)\)`)
	decimal := decimalRegex.FindStringSubmatch(colType)
	digits, decimals = decimal[1], decimal[2]
	return
}

func getFileName(tbName string) (filename string) {
	// avoid test file
	filename = tbName
	for strings.HasSuffix(filename, "_test") {
		pos := strings.LastIndex(filename, "_")
		filename = filename[:pos] + filename[pos+1:]
	}
	return
}

const (
	STRUCT_MODEL_TPL = `
package models

{{modelStruct}}
`

	MODEL_TPL = `
package models

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
	CTRL_TPL = `
package controllers

type {{ctrlName}}Controller struct {
	beego.Controller
}

// @Title Post
// @Description create {{ctrlName}}
// @Param	body		body 	models.{{ctrlName}}	true		"body for {{ctrlName}} content"
// @Success 200 {int} models.{{ctrlName}}.Id
// @Failure 403 body is empty
// @router / [post]
func (this *{{ctrlName}}Controller) Post() {
	var v models.{{ctrlName}}
	json.Unmarshal(this.Ctx.Input.RequestBody, &v)
	if id, err := models.Add{{ctrlName}}(&v); err == nil {
		this.Data["json"] = map[string]int64{"id": id}
	} else {
		this.Data["json"] = err.Error()
	}
	this.ServeJson()
}

// @Title Get
// @Description get {{ctrlName}} by id
// @Param	id		path 	string	true		"The key for staticblock"
// @Success 200 {object} models.{{ctrlName}}
// @Failure 403 :id is empty
// @router /:id [get]
func (this *{{ctrlName}}Controller) GetOne() {
	idStr := this.Ctx.Input.Params[":id"]
	id, _ := strconv.Atoi(idStr)
	v, err := models.Get{{ctrlName}}ById(id)
	if err != nil {
		this.Data["json"] = err.Error()
	} else {
		this.Data["json"] = v
	}
	this.ServeJson()
}

// @Title update
// @Description update the {{ctrlName}}
// @Param	id		path 	string	true		"The id you want to update"
// @Param	body		body 	models.{{ctrlName}}	true		"body for {{ctrlName}} content"
// @Success 200 {object} models.{{ctrlName}}
// @Failure 403 :id is not int
// @router /:id [put]
func (this *{{ctrlName}}Controller) Put() {
	idStr := this.Ctx.Input.Params[":id"]
	id, _ := strconv.Atoi(idStr)
	v := models.{{ctrlName}}{Id: id}
	json.Unmarshal(this.Ctx.Input.RequestBody, &v)
	if err := models.Update{{ctrlName}}ById(&v); err == nil {
		this.Data["json"] = "OK"
	} else {
		this.Data["json"] = err.Error()
	}
	this.ServeJson()
}

// @Title delete
// @Description delete the {{ctrlName}}
// @Param	id		path 	string	true		"The id you want to delete"
// @Success 200 {string} delete success!
// @Failure 403 id is empty
// @router /:id [delete]
func (this *{{ctrlName}}Controller) Delete() {
	idStr := this.Ctx.Input.Params[":id"]
	id, _ := strconv.Atoi(idStr)
	if err := models.Delete{{ctrlName}}(id); err == nil {
		this.Data["json"] = "OK"
	} else {
		this.Data["json"] = err.Error()
	}
	this.ServeJson()
}
`
	ROUTER_TPL = `
// @APIVersion 1.0.0
// @Title beego Test API
// @Description beego has a very cool tools to autogenerate documents for your API
// @Contact astaxie@gmail.com
// @TermsOfServiceUrl http://beego.me/
// @License Apache 2.0
// @LicenseUrl http://www.apache.org/licenses/LICENSE-2.0.html
package routers

import (
	"api/controllers"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/v1",
		/*
		beego.NSNamespace("/object",
			beego.NSInclude(
				&controllers.ObjectController{},
			),
		),
		beego.NSNamespace("/user",
			beego.NSInclude(
				&controllers.UserController{},
			),
		),
*/
		{{nameSpaces}}
	)
	beego.AddNamespace(ns)
}
`
	NAMESPACE_TPL = `
beego.NSNamespace("/{{nameSpace}}",
	beego.NSInclude(
		&controllers.{{ctrlName}}Controller{},
	),
),
`
)

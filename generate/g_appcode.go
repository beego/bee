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
	"database/sql"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/logger/colors"
	"github.com/beego/bee/utils"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	OModel byte = 1 << iota
	OController
	ORouter
)

// DbTransformer has method to reverse engineer a database schema to restful api code
type DbTransformer interface {
	GetTableNames(conn *sql.DB) []string
	GetConstraints(conn *sql.DB, table *Table, blackList map[string]bool)
	GetColumns(conn *sql.DB, table *Table, blackList map[string]bool)
	GetGoDataType(sqlType string) (string, error)
}

// MysqlDB is the MySQL version of DbTransformer
type MysqlDB struct {
}

// PostgresDB is the PostgreSQL version of DbTransformer
type PostgresDB struct {
}

// dbDriver maps a DBMS name to its version of DbTransformer
var dbDriver = map[string]DbTransformer{
	"mysql":    &MysqlDB{},
	"postgres": &PostgresDB{},
}

type MvcPath struct {
	ModelPath      string
	ControllerPath string
	RouterPath     string
}

// typeMapping maps SQL data type to corresponding Go data type
var typeMappingMysql = map[string]string{
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
	"bit":                "uint64",
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
	"tinyblob":           "string",
	"mediumblob":         "string",
	"longblob":           "string",
	"date":               "time.Time", // time
	"datetime":           "time.Time",
	"timestamp":          "time.Time",
	"time":               "time.Time",
	"float":              "float32", // float & decimal
	"double":             "float64",
	"decimal":            "float64",
	"binary":             "string", // binary
	"varbinary":          "string",
	"year":               "int16",
}

// typeMappingPostgres maps SQL data type to corresponding Go data type
var typeMappingPostgres = map[string]string{
	"serial":                      "int", // serial
	"big serial":                  "int64",
	"smallint":                    "int16", // int
	"integer":                     "int",
	"bigint":                      "int64",
	"boolean":                     "bool",   // bool
	"char":                        "string", // string
	"character":                   "string",
	"character varying":           "string",
	"varchar":                     "string",
	"text":                        "string",
	"date":                        "time.Time", // time
	"time":                        "time.Time",
	"timestamp":                   "time.Time",
	"timestamp without time zone": "time.Time",
	"timestamp with time zone":    "time.Time",
	"interval":                    "string",  // time interval, string for now
	"real":                        "float32", // float & decimal
	"double precision":            "float64",
	"decimal":                     "float64",
	"numeric":                     "float64",
	"money":                       "float64", // money
	"bytea":                       "string",  // binary
	"tsvector":                    "string",  // fulltext
	"ARRAY":                       "string",  // array
	"USER-DEFINED":                "string",  // user defined
	"uuid":                        "string",  // uuid
	"json":                        "string",  // json
	"jsonb":                       "string",  // jsonb
	"inet":                        "string",  // ip address
}

// Table represent a table in a database
type Table struct {
	Name          string
	Pk            string
	Uk            []string
	Fk            map[string]*ForeignKey
	Columns       []*Column
	ImportTimePkg bool
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
	Comment     string //column comment
}

// String returns the source code string for the Table struct
func (tb *Table) String() string {
	rv := fmt.Sprintf("type %s struct {\n", utils.CamelCase(tb.Name))
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
	if tag.Comment != "" {
		return fmt.Sprintf("`orm:\"%s\" description:\"%s\"`", strings.Join(ormOptions, ";"), tag.Comment)
	}
	return fmt.Sprintf("`orm:\"%s\"`", strings.Join(ormOptions, ";"))
}

func GenerateAppcode(driver, connStr, level, tables, currpath string) {
	var mode byte
	switch level {
	case "1":
		mode = OModel
	case "2":
		mode = OModel | OController
	case "3":
		mode = OModel | OController | ORouter
	default:
		beeLogger.Log.Fatal("Invalid level value. Must be either \"1\", \"2\", or \"3\"")
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
		beeLogger.Log.Fatal("Generating app code from SQLite database is not supported yet.")
	default:
		beeLogger.Log.Fatal("Unknown database driver. Must be either \"mysql\", \"postgres\" or \"sqlite\"")
	}
	gen(driver, connStr, mode, selectedTables, currpath)
}

// Generate takes table, column and foreign key information from database connection
// and generate corresponding golang source files
func gen(dbms, connStr string, mode byte, selectedTableNames map[string]bool, apppath string) {
	db, err := sql.Open(dbms, connStr)
	if err != nil {
		beeLogger.Log.Fatalf("Could not connect to '%s' database using '%s': %s", dbms, connStr, err)
	}
	defer db.Close()
	if trans, ok := dbDriver[dbms]; ok {
		beeLogger.Log.Info("Analyzing database tables...")
		var tableNames []string
		if len(selectedTableNames) != 0 {
			for tableName := range selectedTableNames {
				tableNames = append(tableNames, tableName)
			}
		} else {
			tableNames = trans.GetTableNames(db)
		}
		tables := getTableObjects(tableNames, db, trans)
		mvcPath := new(MvcPath)
		mvcPath.ModelPath = path.Join(apppath, "models")
		mvcPath.ControllerPath = path.Join(apppath, "controllers")
		mvcPath.RouterPath = path.Join(apppath, "routers")
		createPaths(mode, mvcPath)
		pkgPath := getPackagePath(apppath)
		writeSourceFiles(pkgPath, tables, mode, mvcPath)
	} else {
		beeLogger.Log.Fatalf("Generating app code from '%s' database is not supported yet.", dbms)
	}
}

// GetTableNames returns a slice of table names in the current database
func (*MysqlDB) GetTableNames(db *sql.DB) (tables []string) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		beeLogger.Log.Fatalf("Could not show tables: %s", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			beeLogger.Log.Fatalf("Could not show tables: %s", err)
		}
		tables = append(tables, name)
	}
	return
}

// getTableObjects process each table name
func getTableObjects(tableNames []string, db *sql.DB, dbTransformer DbTransformer) (tables []*Table) {
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
		dbTransformer.GetConstraints(db, tb, blackList)
		tables = append(tables, tb)
	}
	// process columns, ignoring blacklisted tables
	for _, tb := range tables {
		dbTransformer.GetColumns(db, tb, blackList)
	}
	return
}

// GetConstraints gets primary key, unique key and foreign keys of a table from
// information_schema and fill in the Table struct
func (*MysqlDB) GetConstraints(db *sql.DB, table *Table, blackList map[string]bool) {
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
		beeLogger.Log.Fatal("Could not query INFORMATION_SCHEMA for PK/UK/FK information")
	}
	for rows.Next() {
		var constraintTypeBytes, columnNameBytes, refTableSchemaBytes, refTableNameBytes, refColumnNameBytes, refOrdinalPosBytes []byte
		if err := rows.Scan(&constraintTypeBytes, &columnNameBytes, &refTableSchemaBytes, &refTableNameBytes, &refColumnNameBytes, &refOrdinalPosBytes); err != nil {
			beeLogger.Log.Fatal("Could not read INFORMATION_SCHEMA for PK/UK/FK information")
		}
		constraintType, columnName, refTableSchema, refTableName, refColumnName, refOrdinalPos :=
			string(constraintTypeBytes), string(columnNameBytes), string(refTableSchemaBytes),
			string(refTableNameBytes), string(refColumnNameBytes), string(refOrdinalPosBytes)
		if constraintType == "PRIMARY KEY" {
			if refOrdinalPos == "1" {
				table.Pk = columnName
			} else {
				table.Pk = ""
				// Add table to blacklist so that other struct will not reference it, because we are not
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

// GetColumns retrieves columns details from
// information_schema and fill in the Column struct
func (mysqlDB *MysqlDB) GetColumns(db *sql.DB, table *Table, blackList map[string]bool) {
	// retrieve columns
	colDefRows, err := db.Query(
		`SELECT
			column_name, data_type, column_type, is_nullable, column_default, extra, column_comment 
		FROM
			information_schema.columns
		WHERE
			table_schema = database() AND table_name = ?`,
		table.Name)
	if err != nil {
		beeLogger.Log.Fatalf("Could not query the database: %s", err)
	}
	defer colDefRows.Close()

	for colDefRows.Next() {
		// datatype as bytes so that SQL <null> values can be retrieved
		var colNameBytes, dataTypeBytes, columnTypeBytes, isNullableBytes, columnDefaultBytes, extraBytes, columnCommentBytes []byte
		if err := colDefRows.Scan(&colNameBytes, &dataTypeBytes, &columnTypeBytes, &isNullableBytes, &columnDefaultBytes, &extraBytes, &columnCommentBytes); err != nil {
			beeLogger.Log.Fatal("Could not query INFORMATION_SCHEMA for column information")
		}
		colName, dataType, columnType, isNullable, columnDefault, extra, columnComment :=
			string(colNameBytes), string(dataTypeBytes), string(columnTypeBytes), string(isNullableBytes), string(columnDefaultBytes), string(extraBytes), string(columnCommentBytes)

		// create a column
		col := new(Column)
		col.Name = utils.CamelCase(colName)
		col.Type, err = mysqlDB.GetGoDataType(dataType)
		if err != nil {
			beeLogger.Log.Fatalf("%s", err)
		}

		// Tag info
		tag := new(OrmTag)
		tag.Column = colName
		tag.Comment = columnComment
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
				col.Name = utils.CamelCase(colName)
				col.Type = "*" + utils.CamelCase(refStructName)
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
						col.Type, err = mysqlDB.GetGoDataType(dataType + " " + sign)
						if err != nil {
							beeLogger.Log.Fatalf("%s", err)
						}
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
					// need to import time package
					table.ImportTimePkg = true
				}
				if isSQLDecimal(dataType) {
					tag.Digits, tag.Decimals = extractDecimal(columnType)
				}
				if isSQLBinaryType(dataType) {
					tag.Size = extractColSize(columnType)
				}
				if isSQLBitType(dataType) {
					tag.Size = extractColSize(columnType)
				}
			}
		}
		col.Tag = tag
		table.Columns = append(table.Columns, col)
	}
}

// GetGoDataType maps an SQL data type to Golang data type
func (*MysqlDB) GetGoDataType(sqlType string) (string, error) {
	if v, ok := typeMappingMysql[sqlType]; ok {
		return v, nil
	}
	return "", fmt.Errorf("data type '%s' not found", sqlType)
}

// GetTableNames for PostgreSQL
func (*PostgresDB) GetTableNames(db *sql.DB) (tables []string) {
	rows, err := db.Query(`
		SELECT table_name FROM information_schema.tables
		WHERE table_catalog = current_database() AND
		table_type = 'BASE TABLE' AND
		table_schema NOT IN ('pg_catalog', 'information_schema')`)
	if err != nil {
		beeLogger.Log.Fatalf("Could not show tables: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			beeLogger.Log.Fatalf("Could not show tables: %s", err)
		}
		tables = append(tables, name)
	}
	return
}

// GetConstraints for PostgreSQL
func (*PostgresDB) GetConstraints(db *sql.DB, table *Table, blackList map[string]bool) {
	rows, err := db.Query(
		`SELECT
			c.constraint_type,
			u.column_name,
			cu.table_catalog AS referenced_table_catalog,
			cu.table_name AS referenced_table_name,
			cu.column_name AS referenced_column_name,
			u.ordinal_position
		FROM
			information_schema.table_constraints c
		INNER JOIN
			information_schema.key_column_usage u ON c.constraint_name = u.constraint_name
		INNER JOIN
			information_schema.constraint_column_usage cu ON cu.constraint_name =  c.constraint_name
		WHERE
			c.table_catalog = current_database() AND c.table_schema NOT IN ('pg_catalog', 'information_schema')
			 AND c.table_name = $1
			AND u.table_catalog = current_database() AND u.table_schema NOT IN ('pg_catalog', 'information_schema')
			 AND u.table_name = $2`,
		table.Name, table.Name) //  u.position_in_unique_constraint,
	if err != nil {
		beeLogger.Log.Fatalf("Could not query INFORMATION_SCHEMA for PK/UK/FK information: %s", err)
	}

	for rows.Next() {
		var constraintTypeBytes, columnNameBytes, refTableSchemaBytes, refTableNameBytes, refColumnNameBytes, refOrdinalPosBytes []byte
		if err := rows.Scan(&constraintTypeBytes, &columnNameBytes, &refTableSchemaBytes, &refTableNameBytes, &refColumnNameBytes, &refOrdinalPosBytes); err != nil {
			beeLogger.Log.Fatalf("Could not read INFORMATION_SCHEMA for PK/UK/FK information: %s", err)
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

// GetColumns for PostgreSQL
func (postgresDB *PostgresDB) GetColumns(db *sql.DB, table *Table, blackList map[string]bool) {
	// retrieve columns
	colDefRows, err := db.Query(
		`SELECT
			column_name,
			data_type,
			data_type ||
			CASE
				WHEN data_type = 'character' THEN '('||character_maximum_length||')'
				WHEN data_type = 'numeric' THEN '(' || numeric_precision || ',' || numeric_scale ||')'
				ELSE ''
			END AS column_type,
			is_nullable,
			column_default,
			'' AS extra
		FROM
			information_schema.columns
		WHERE
			table_catalog = current_database() AND table_schema NOT IN ('pg_catalog', 'information_schema')
			 AND table_name = $1`,
		table.Name)
	if err != nil {
		beeLogger.Log.Fatalf("Could not query INFORMATION_SCHEMA for column information: %s", err)
	}
	defer colDefRows.Close()

	for colDefRows.Next() {
		// datatype as bytes so that SQL <null> values can be retrieved
		var colNameBytes, dataTypeBytes, columnTypeBytes, isNullableBytes, columnDefaultBytes, extraBytes []byte
		if err := colDefRows.Scan(&colNameBytes, &dataTypeBytes, &columnTypeBytes, &isNullableBytes, &columnDefaultBytes, &extraBytes); err != nil {
			beeLogger.Log.Fatalf("Could not query INFORMATION_SCHEMA for column information: %s", err)
		}
		colName, dataType, columnType, isNullable, columnDefault, extra :=
			string(colNameBytes), string(dataTypeBytes), string(columnTypeBytes), string(isNullableBytes), string(columnDefaultBytes), string(extraBytes)
		// Create a column
		col := new(Column)
		col.Name = utils.CamelCase(colName)
		col.Type, err = postgresDB.GetGoDataType(dataType)
		if err != nil {
			beeLogger.Log.Fatalf("%s", err)
		}

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
				col.Name = utils.CamelCase(colName)
				col.Type = "*" + utils.CamelCase(refStructName)
			} else {
				// if the name of column is Id, and it's not primary key
				if colName == "id" {
					col.Name = "Id_RENAME"
				}
				if isNullable == "YES" {
					tag.Null = true
				}
				if isSQLStringType(dataType) {
					tag.Size = extractColSize(columnType)
				}
				if isSQLTemporalType(dataType) || strings.HasPrefix(dataType, "timestamp") {
					tag.Type = dataType
					//check auto_now, auto_now_add
					if columnDefault == "CURRENT_TIMESTAMP" && extra == "on update CURRENT_TIMESTAMP" {
						tag.AutoNow = true
					} else if columnDefault == "CURRENT_TIMESTAMP" {
						tag.AutoNowAdd = true
					}
					// need to import time package
					table.ImportTimePkg = true
				}
				if isSQLDecimal(dataType) {
					tag.Digits, tag.Decimals = extractDecimal(columnType)
				}
				if isSQLBinaryType(dataType) {
					tag.Size = extractColSize(columnType)
				}
				if isSQLStrangeType(dataType) {
					tag.Type = dataType
				}
			}
		}
		col.Tag = tag
		table.Columns = append(table.Columns, col)
	}
}

// GetGoDataType returns the Go type from the mapped Postgres type
func (*PostgresDB) GetGoDataType(sqlType string) (string, error) {
	if v, ok := typeMappingPostgres[sqlType]; ok {
		return v, nil
	}
	return "", fmt.Errorf("data type '%s' not found", sqlType)
}

// deleteAndRecreatePaths removes several directories completely
func createPaths(mode byte, paths *MvcPath) {
	if (mode & OModel) == OModel {
		os.Mkdir(paths.ModelPath, 0777)
	}
	if (mode & OController) == OController {
		os.Mkdir(paths.ControllerPath, 0777)
	}
	if (mode & ORouter) == ORouter {
		os.Mkdir(paths.RouterPath, 0777)
	}
}

// writeSourceFiles generates source files for model/controller/router
// It will wipe the following directories and recreate them:./models, ./controllers, ./routers
// Newly geneated files will be inside these folders.
func writeSourceFiles(pkgPath string, tables []*Table, mode byte, paths *MvcPath) {
	if (OModel & mode) == OModel {
		beeLogger.Log.Info("Creating model files...")
		writeModelFiles(tables, paths.ModelPath)
	}
	if (OController & mode) == OController {
		beeLogger.Log.Info("Creating controller files...")
		writeControllerFiles(tables, paths.ControllerPath, pkgPath)
	}
	if (ORouter & mode) == ORouter {
		beeLogger.Log.Info("Creating router files...")
		writeRouterFile(tables, paths.RouterPath, pkgPath)
	}
}

// writeModelFiles generates model files
func writeModelFiles(tables []*Table, mPath string) {
	w := colors.NewColorWriter(os.Stdout)

	for _, tb := range tables {
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
			template = StructModelTPL
		} else {
			template = ModelTPL
		}
		fileStr := strings.Replace(template, "{{modelStruct}}", tb.String(), 1)
		fileStr = strings.Replace(fileStr, "{{modelName}}", utils.CamelCase(tb.Name), -1)
		fileStr = strings.Replace(fileStr, "{{tableName}}", tb.Name, -1)

		// If table contains time field, import time.Time package
		timePkg := ""
		importTimePkg := ""
		if tb.ImportTimePkg {
			timePkg = "\"time\"\n"
			importTimePkg = "import \"time\"\n"
		}
		fileStr = strings.Replace(fileStr, "{{timePkg}}", timePkg, -1)
		fileStr = strings.Replace(fileStr, "{{importTimePkg}}", importTimePkg, -1)
		if _, err := f.WriteString(fileStr); err != nil {
			beeLogger.Log.Fatalf("Could not write model file to '%s': %s", fpath, err)
		}
		utils.CloseFile(f)
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", fpath, "\x1b[0m")
		utils.FormatSourceCode(fpath)
	}
}

// writeControllerFiles generates controller files
func writeControllerFiles(tables []*Table, cPath string, pkgPath string) {
	w := colors.NewColorWriter(os.Stdout)

	for _, tb := range tables {
		if tb.Pk == "" {
			continue
		}
		filename := getFileName(tb.Name)
		fpath := path.Join(cPath, filename+".go")
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
		fileStr := strings.Replace(CtrlTPL, "{{ctrlName}}", utils.CamelCase(tb.Name), -1)
		fileStr = strings.Replace(fileStr, "{{pkgPath}}", pkgPath, -1)
		if _, err := f.WriteString(fileStr); err != nil {
			beeLogger.Log.Fatalf("Could not write controller file to '%s': %s", fpath, err)
		}
		utils.CloseFile(f)
		fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", fpath, "\x1b[0m")
		utils.FormatSourceCode(fpath)
	}
}

// writeRouterFile generates router file
func writeRouterFile(tables []*Table, rPath string, pkgPath string) {
	w := colors.NewColorWriter(os.Stdout)

	var nameSpaces []string
	for _, tb := range tables {
		if tb.Pk == "" {
			continue
		}
		// Add namespaces
		nameSpace := strings.Replace(NamespaceTPL, "{{nameSpace}}", tb.Name, -1)
		nameSpace = strings.Replace(nameSpace, "{{ctrlName}}", utils.CamelCase(tb.Name), -1)
		nameSpaces = append(nameSpaces, nameSpace)
	}
	// Add export controller
	fpath := filepath.Join(rPath, "router.go")
	routerStr := strings.Replace(RouterTPL, "{{nameSpaces}}", strings.Join(nameSpaces, ""), 1)
	routerStr = strings.Replace(routerStr, "{{pkgPath}}", pkgPath, 1)
	var f *os.File
	var err error
	if utils.IsExist(fpath) {
		beeLogger.Log.Warnf("'%s' already exists. Do you want to overwrite it? [Yes|No] ", fpath)
		if utils.AskForConfirmation() {
			f, err = os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0666)
			if err != nil {
				beeLogger.Log.Warnf("%s", err)
				return
			}
		} else {
			beeLogger.Log.Warnf("Skipped create file '%s'", fpath)
			return
		}
	} else {
		f, err = os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			beeLogger.Log.Warnf("%s", err)
			return
		}
	}
	if _, err := f.WriteString(routerStr); err != nil {
		beeLogger.Log.Fatalf("Could not write router file to '%s': %s", fpath, err)
	}
	utils.CloseFile(f)
	fmt.Fprintf(w, "\t%s%screate%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", fpath, "\x1b[0m")
	utils.FormatSourceCode(fpath)
}

func isSQLTemporalType(t string) bool {
	return t == "date" || t == "datetime" || t == "timestamp" || t == "time"
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

func isSQLBinaryType(t string) bool {
	return t == "binary" || t == "varbinary"
}

func isSQLBitType(t string) bool {
	return t == "bit"
}
func isSQLStrangeType(t string) bool {
	return t == "interval" || t == "uuid" || t == "json"
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

func getPackagePath(curpath string) (packpath string) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		beeLogger.Log.Fatal("GOPATH environment variable is not set or empty")
	}

	beeLogger.Log.Debugf("GOPATH: %s", utils.FILE(), utils.LINE(), gopath)

	appsrcpath := ""
	haspath := false
	wgopath := filepath.SplitList(gopath)

	for _, wg := range wgopath {
		wg, _ = filepath.EvalSymlinks(filepath.Join(wg, "src"))
		if strings.HasPrefix(strings.ToLower(curpath), strings.ToLower(wg)) {
			haspath = true
			appsrcpath = wg
			break
		}
	}

	if !haspath {
		beeLogger.Log.Fatalf("Cannot generate application code outside of GOPATH '%s' compare with CWD '%s'", gopath, curpath)
	}

	if curpath == appsrcpath {
		beeLogger.Log.Fatal("Cannot generate application code outside of application path")
	}

	packpath = strings.Join(strings.Split(curpath[len(appsrcpath)+1:], string(filepath.Separator)), "/")
	return
}

const (
	StructModelTPL = `package models
{{importTimePkg}}
{{modelStruct}}
`

	ModelTPL = `package models

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	{{timePkg}}
	"github.com/astaxie/beego/orm"
)

{{modelStruct}}

func (t *{{modelName}}) TableName() string {
	return "{{tableName}}"
}

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
		if strings.Contains(k, "isnull") {
			qs = qs.Filter(k, (v == "true" || v == "1"))
		} else {
			qs = qs.Filter(k, v)
		}
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
	CtrlTPL = `package controllers

import (
	"{{pkgPath}}/models"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/astaxie/beego"
)

// {{ctrlName}}Controller operations for {{ctrlName}}
type {{ctrlName}}Controller struct {
	beego.Controller
}

// URLMapping ...
func (c *{{ctrlName}}Controller) URLMapping() {
	c.Mapping("Post", c.Post)
	c.Mapping("GetOne", c.GetOne)
	c.Mapping("GetAll", c.GetAll)
	c.Mapping("Put", c.Put)
	c.Mapping("Delete", c.Delete)
}

// Post ...
// @Title Post
// @Description create {{ctrlName}}
// @Param	body		body 	models.{{ctrlName}}	true		"body for {{ctrlName}} content"
// @Success 201 {int} models.{{ctrlName}}
// @Failure 403 body is empty
// @router / [post]
func (c *{{ctrlName}}Controller) Post() {
	var v models.{{ctrlName}}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &v); err == nil {
		if _, err := models.Add{{ctrlName}}(&v); err == nil {
			c.Ctx.Output.SetStatus(201)
			c.Data["json"] = v
		} else {
			c.Data["json"] = err.Error()
		}
	} else {
		c.Data["json"] = err.Error()
	}
	c.ServeJSON()
}

// GetOne ...
// @Title Get One
// @Description get {{ctrlName}} by id
// @Param	id		path 	string	true		"The key for staticblock"
// @Success 200 {object} models.{{ctrlName}}
// @Failure 403 :id is empty
// @router /:id [get]
func (c *{{ctrlName}}Controller) GetOne() {
	idStr := c.Ctx.Input.Param(":id")
	id, _ := strconv.Atoi(idStr)
	v, err := models.Get{{ctrlName}}ById(id)
	if err != nil {
		c.Data["json"] = err.Error()
	} else {
		c.Data["json"] = v
	}
	c.ServeJSON()
}

// GetAll ...
// @Title Get All
// @Description get {{ctrlName}}
// @Param	query	query	string	false	"Filter. e.g. col1:v1,col2:v2 ..."
// @Param	fields	query	string	false	"Fields returned. e.g. col1,col2 ..."
// @Param	sortby	query	string	false	"Sorted-by fields. e.g. col1,col2 ..."
// @Param	order	query	string	false	"Order corresponding to each sortby field, if single value, apply to all sortby fields. e.g. desc,asc ..."
// @Param	limit	query	string	false	"Limit the size of result set. Must be an integer"
// @Param	offset	query	string	false	"Start position of result set. Must be an integer"
// @Success 200 {object} models.{{ctrlName}}
// @Failure 403
// @router / [get]
func (c *{{ctrlName}}Controller) GetAll() {
	var fields []string
	var sortby []string
	var order []string
	var query = make(map[string]string)
	var limit int64 = 10
	var offset int64

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
			kv := strings.SplitN(cond, ":", 2)
			if len(kv) != 2 {
				c.Data["json"] = errors.New("Error: invalid query key/value pair")
				c.ServeJSON()
				return
			}
			k, v := kv[0], kv[1]
			query[k] = v
		}
	}

	l, err := models.GetAll{{ctrlName}}(query, fields, sortby, order, offset, limit)
	if err != nil {
		c.Data["json"] = err.Error()
	} else {
		c.Data["json"] = l
	}
	c.ServeJSON()
}

// Put ...
// @Title Put
// @Description update the {{ctrlName}}
// @Param	id		path 	string	true		"The id you want to update"
// @Param	body		body 	models.{{ctrlName}}	true		"body for {{ctrlName}} content"
// @Success 200 {object} models.{{ctrlName}}
// @Failure 403 :id is not int
// @router /:id [put]
func (c *{{ctrlName}}Controller) Put() {
	idStr := c.Ctx.Input.Param(":id")
	id, _ := strconv.Atoi(idStr)
	v := models.{{ctrlName}}{Id: id}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &v); err == nil {
		if err := models.Update{{ctrlName}}ById(&v); err == nil {
			c.Data["json"] = "OK"
		} else {
			c.Data["json"] = err.Error()
		}
	} else {
		c.Data["json"] = err.Error()
	}
	c.ServeJSON()
}

// Delete ...
// @Title Delete
// @Description delete the {{ctrlName}}
// @Param	id		path 	string	true		"The id you want to delete"
// @Success 200 {string} delete success!
// @Failure 403 id is empty
// @router /:id [delete]
func (c *{{ctrlName}}Controller) Delete() {
	idStr := c.Ctx.Input.Param(":id")
	id, _ := strconv.Atoi(idStr)
	if err := models.Delete{{ctrlName}}(id); err == nil {
		c.Data["json"] = "OK"
	} else {
		c.Data["json"] = err.Error()
	}
	c.ServeJSON()
}
`
	RouterTPL = `// @APIVersion 1.0.0
// @Title beego Test API
// @Description beego has a very cool tools to autogenerate documents for your API
// @Contact astaxie@gmail.com
// @TermsOfServiceUrl http://beego.me/
// @License Apache 2.0
// @LicenseUrl http://www.apache.org/licenses/LICENSE-2.0.html
package routers

import (
	"{{pkgPath}}/controllers"

	"github.com/astaxie/beego"
)

func init() {
	ns := beego.NewNamespace("/v1",
		{{nameSpaces}}
	)
	beego.AddNamespace(ns)
}
`
	NamespaceTPL = `
		beego.NSNamespace("/{{nameSpace}}",
			beego.NSInclude(
				&controllers.{{ctrlName}}Controller{},
			),
		),
`
)

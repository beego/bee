package main

import (
	"fmt"
	"strings"
)

func generateScaffold(sname, fields, crupath, driver, conn string) {
	// generate model
	ColorLog("[INFO] Do you want to create a %v model? [yes|no]]  ", sname)
	if askForConfirmation() {
		generateModel(sname, fields, crupath)
	}

	// generate controller
	ColorLog("[INFO] Do you want to create a %v controller? [yes|no]]  ", sname)
	if askForConfirmation() {
		generateController(sname, crupath)
	}
	// generate view
	ColorLog("[INFO] Do you want to create views for this %v resource? [yes|no]]  ", sname)
	if askForConfirmation() {
		generateView(sname, crupath)
	}
	// generate migration
	ColorLog("[INFO] Do you want to create a %v migration and schema for this resource? [yes|no]]  ", sname)
	if askForConfirmation() {
		upsql := ""
		downsql := ""
		if fields != "" {
			upsql = `m.SQL("CREATE TABLE ` + sname + "(" + generateSQLFromFields(fields) + `)");`
			downsql = `m.SQL("DROP TABLE ` + "`" + sname + "`" + `")`
			if driver == "" {
				downsql = strings.Replace(downsql, "`", "", -1)
			}
		}
		generateMigration(sname, upsql, downsql, crupath)
	}
	// run migration
	ColorLog("[INFO] Do you want to migrate the database? [yes|no]]  ")
	if askForConfirmation() {
		migrateUpdate(crupath, driver, conn)
	}
	ColorLog("[INFO] All done! Don't forget to add  beego.Router(\"/%v\" ,&controllers.%vController{}) to routers/route.go\n", sname, strings.Title(sname))
}

func generateSQLFromFields(fields string) string {
	sql := ""
	tags := ""
	fds := strings.Split(fields, ",")
	for i, v := range fds {
		kv := strings.SplitN(v, ":", 2)
		if len(kv) != 2 {
			ColorLog("[ERRO] Fields format is wrong. Should be: key:type,key:type " + v + "\n")
			return ""
		}
		typ, tag := "", ""
		switch driver {
		case "mysql":
			typ, tag = getSQLTypeMysql(kv[1])
		case "postgres":
			typ, tag = getSQLTypePostgresql(kv[1])
		default:
			typ, tag = getSQLTypeMysql(kv[1])
		}
		if typ == "" {
			ColorLog("[ERRO] Fields format is wrong. Should be: key:type,key:type " + v + "\n")
			return ""
		}
		if i == 0 && strings.ToLower(kv[0]) != "id" {
			switch driver {
			case "mysql":
				sql = sql + "`id` int(11) NOT NULL AUTO_INCREMENT,"
				tags = tags + "PRIMARY KEY (`id`),"
			case "postgres":
				sql = sql + "id interger serial primary key,"
			default:
				sql = sql + "`id` int(11) NOT NULL AUTO_INCREMENT,"
				tags = tags + "PRIMARY KEY (`id`),"
			}
		}

		sql = sql + "`" + snakeString(kv[0]) + "` " + typ + ","
		if tag != "" {
			tags = tags + fmt.Sprintf(tag, "`"+snakeString(kv[0])+"`") + ","
		}
	}
	if driver == "postgres" {
		sql = strings.Replace(sql, "`", "", -1)
		tags = strings.Replace(tags, "`", "", -1)
	}
	sql = strings.TrimRight(sql+tags, ",")
	return sql
}

func getSQLTypeMysql(ktype string) (tp, tag string) {
	kv := strings.SplitN(ktype, ":", 2)
	switch kv[0] {
	case "string":
		if len(kv) == 2 {
			return "varchar(" + kv[1] + ") NOT NULL", ""
		}
		return "varchar(128) NOT NULL", ""
	case "text":
		return "longtext  NOT NULL", ""
	case "auto":
		return "int(11) NOT NULL AUTO_INCREMENT", ""
	case "pk":
		return "int(11) NOT NULL", "PRIMARY KEY (%s)"
	case "datetime":
		return "datetime NOT NULL", ""
	case "int", "int8", "int16", "int32", "int64":
		fallthrough
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "int(11) DEFAULT NULL", ""
	case "bool":
		return "tinyint(1) NOT NULL", ""
	case "float32", "float64":
		return "float NOT NULL", ""
	case "float":
		return "float NOT NULL", ""
	}
	return "", ""
}

func getSQLTypePostgresql(ktype string) (tp, tag string) {
	kv := strings.SplitN(ktype, ":", 2)
	switch kv[0] {
	case "string":
		if len(kv) == 2 {
			return "char(" + kv[1] + ") NOT NULL", ""
		}
		return "TEXT NOT NULL", ""
	case "text":
		return "TEXT NOT NULL", ""
	case "auto", "pk":
		return "serial primary key", ""
	case "datetime":
		return "TIMESTAMP WITHOUT TIME ZONE NOT NULL", ""
	case "int", "int8", "int16", "int32", "int64":
		fallthrough
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "integer DEFAULT NULL", ""
	case "bool":
		return "boolean NOT NULL", ""
	case "float32", "float64", "float":
		return "numeric NOT NULL", ""
	}
	return "", ""
}

package main

import (
	"fmt"
	"strings"
)

func generateScaffold(sname, fields, crupath, driver, conn string) {
	// generate model
	ColorLog("[INFO] Do you want me to create a %v model? [yes|no]]  ", sname)
	if askForConfirmation() {
		generateModel(sname, fields, crupath)
	}

	// generate controller
	ColorLog("[INFO] Do you want me to create a %v controller? [yes|no]]  ", sname)
	if askForConfirmation() {
		generateController(sname, crupath)
	}
	// generate view
	ColorLog("[INFO] Do you want me to create views for this %v resource? [yes|no]]  ", sname)
	if askForConfirmation() {
		generateView(sname, crupath)
	}
	// generate migration
	ColorLog("[INFO] Do you want me to create a %v migration and schema for this resource? [yes|no]]  ", sname)
	if askForConfirmation() {
		upsql := ""
		downsql := ""
		if fields != "" {
			upsql = `m.Sql("CREATE TABLE ` + sname + "(" + generateSQLFromFields(fields) + `)");`
			downsql = `m.Sql("DROP TABLE ` + "`" + sname + "`" + `")`
		}
		generateMigration(sname, upsql, downsql, crupath)
	}
	// run migration
	ColorLog("[INFO] Do you want to go ahead and migrate the database? [yes|no]]  ")
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
			ColorLog("[ERRO] the filds format is wrong. should key:type,key:type " + v)
			return ""
		}
		typ, tag := getSqlType(kv[1])
		if typ == "" {
			ColorLog("[ERRO] the filds format is wrong. should key:type,key:type " + v)
			return ""
		}
		if i == 0 && strings.ToLower(kv[0]) != "id" {
			sql = sql + "`id` int(11) NOT NULL AUTO_INCREMENT,"
			tags = tags + "PRIMARY KEY (`id`),"
		}
		sql = sql + "`" + snakeString(kv[0]) + "` " + typ + ","
		if tag != "" {
			tags = tags + fmt.Sprintf(tag, "`"+snakeString(kv[0])+"`") + ","
		}
	}
	sql = strings.TrimRight(sql+tags, ",")
	return sql
}

func getSqlType(ktype string) (tp, tag string) {
	kv := strings.SplitN(ktype, ":", 2)
	switch kv[0] {
	case "string":
		if len(kv) == 2 {
			return "varchar(" + kv[1] + ") NOT NULL", ""
		} else {
			return "varchar(128) NOT NULL", ""
		}
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

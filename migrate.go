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
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

var cmdMigrate = &Command{
	UsageLine: "migrate [Command]",
	Short:     "run database migrations",
	Long: `
bee migrate
    run all outstanding migrations
    -driver: [mysql | postgresql | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test

bee migrate rollback
    rollback the last migration operation
    -driver: [mysql | postgresql | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test

bee migrate reset
    rollback all migrations
    -driver: [mysql | postgresql | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test

bee migrate refresh
    rollback all migrations and run them all again
    -driver: [mysql | postgresql | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test
`,
}

var mDriver docValue
var mConn docValue

func init() {
	cmdMigrate.Run = runMigration
	cmdMigrate.Flag.Var(&mDriver, "driver", "database driver: mysql, postgresql, etc.")
	cmdMigrate.Flag.Var(&mConn, "conn", "connection string used by the driver to connect to a database instance")
}

func runMigration(cmd *Command, args []string) {
	crupath, _ := os.Getwd()

	gopath := os.Getenv("GOPATH")
	Debugf("gopath:%s", gopath)
	if gopath == "" {
		ColorLog("[ERRO] $GOPATH not found\n")
		ColorLog("[HINT] Set $GOPATH in your environment vairables\n")
		os.Exit(2)
	}
	// getting command line arguments
	if len(args) != 0 {
		cmd.Flag.Parse(args[1:])
	}
	if mDriver == "" {
		mDriver = "mysql"
	}
	if mConn == "" {
		mConn = "root:@tcp(127.0.0.1:3306)/test"
	}
	ColorLog("[INFO] Using '%s' as 'driver'\n", mDriver)
	ColorLog("[INFO] Using '%s' as 'conn'\n", mConn)
	driverStr, connStr := string(mDriver), string(mConn)
	if len(args) == 0 {
		// run all outstanding migrations
		ColorLog("[INFO] Running all outstanding migrations\n")
		migrateUpdate(crupath, driverStr, connStr)
	} else {
		mcmd := args[0]
		switch mcmd {
		case "rollback":
			ColorLog("[INFO] Rolling back the last migration operation\n")
			migrateRollback(crupath, driverStr, connStr)
		case "reset":
			ColorLog("[INFO] Reseting all migrations\n")
			migrateReset(crupath, driverStr, connStr)
		case "refresh":
			ColorLog("[INFO] Refreshing all migrations\n")
			migrateReset(crupath, driverStr, connStr)
		default:
			ColorLog("[ERRO] Command is missing\n")
			os.Exit(2)
		}
	}
	ColorLog("[SUCC] Migration successful!\n")
}

func migrateUpdate(crupath, driver, connStr string) {
	migrate("upgrade", crupath, driver, connStr)
}

func migrateRollback(crupath, driver, connStr string) {
	migrate("rollback", crupath, driver, connStr)
}

func migrateReset(crupath, driver, connStr string) {
	migrate("reset", crupath, driver, connStr)
}

func migrateRefresh(crupath, driver, connStr string) {
	migrate("refresh", crupath, driver, connStr)
}

func migrate(goal, crupath, driver, connStr string) {
	dir := path.Join(crupath, "database", "migrations")
	binary := "m"
	source := binary + ".go"
	// connect to database
	db, err := sql.Open(driver, connStr)
	if err != nil {
		ColorLog("[ERRO] Could not connect to %s: %s\n", driver, connStr)
		os.Exit(2)
	}
	defer db.Close()
	checkForSchemaUpdateTable(db)
	latestName, latestTime := getLatestMigration(db)
	writeMigrationSourceFile(dir, source, driver, connStr, latestTime, latestName, goal)
	buildMigrationBinary(dir, binary)
	runMigrationBinary(dir, binary)
	removeTempFile(dir, source)
	removeTempFile(dir, binary)
}

func checkForSchemaUpdateTable(db *sql.DB) {
	if rows, err := db.Query("SHOW TABLES LIKE 'migrations'"); err != nil {
		ColorLog("[ERRO] Could not show migrations table: %s\n", err)
		os.Exit(2)
	} else if !rows.Next() {
		// no migrations table, create anew
		ColorLog("[INFO] Creating 'migrations' table...\n")
		if _, err := db.Query(MYSQL_MIGRATION_DDL); err != nil {
			ColorLog("[ERRO] Could not create migrations table: %s\n", err)
			os.Exit(2)
		}
	}
	// checking that migrations table schema are expected
	if rows, err := db.Query("DESC migrations"); err != nil {
		ColorLog("[ERRO] Could not show columns of migrations table: %s\n", err)
		os.Exit(2)
	} else {
		for rows.Next() {
			var fieldBytes, typeBytes, nullBytes, keyBytes, defaultBytes, extraBytes []byte
			if err := rows.Scan(&fieldBytes, &typeBytes, &nullBytes, &keyBytes, &defaultBytes, &extraBytes); err != nil {
				ColorLog("[ERRO] Could not read column information: %s\n", err)
				os.Exit(2)
			}
			fieldStr, typeStr, nullStr, keyStr, defaultStr, extraStr :=
				string(fieldBytes), string(typeBytes), string(nullBytes), string(keyBytes), string(defaultBytes), string(extraBytes)
			if fieldStr == "id_migration" {
				if keyStr != "PRI" || extraStr != "auto_increment" {
					ColorLog("[ERRO] Column migration.id_migration type mismatch: KEY: %s, EXTRA: %s\n", keyStr, extraStr)
					ColorLog("[HINT] Expecting KEY: PRI, EXTRA: auto_increment\n")
					os.Exit(2)
				}
			} else if fieldStr == "name" {
				if !strings.HasPrefix(typeStr, "varchar") || nullStr != "YES" {
					ColorLog("[ERRO] Column migration.name type mismatch: TYPE: %s, NULL: %s\n", typeStr, nullStr)
					ColorLog("[HINT] Expecting TYPE: varchar, NULL: YES\n")
					os.Exit(2)
				}

			} else if fieldStr == "created_at" {
				if typeStr != "timestamp" || defaultStr != "CURRENT_TIMESTAMP" {
					ColorLog("[ERRO] Column migration.timestamp type mismatch: TYPE: %s, DEFAULT: %s\n", typeStr, defaultStr)
					ColorLog("[HINT] Expecting TYPE: timestamp, DEFAULT: CURRENT_TIMESTAMP\n")
					os.Exit(2)
				}
			}
		}
	}
}

func getLatestMigration(db *sql.DB) (file string, createdAt int64) {
	sql := "SELECT name, created_at FROM migrations where status = 'update' ORDER BY id_migration DESC LIMIT 1"
	if rows, err := db.Query(sql); err != nil {
		ColorLog("[ERRO] Could not retrieve migrations: %s\n", err)
		os.Exit(2)
	} else {
		var createdAtStr string
		if rows.Next() {
			if err := rows.Scan(&file, &createdAtStr); err != nil {
				ColorLog("[ERRO] Could not read migrations in database: %s\n", err)
				os.Exit(2)
			}
			if t, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err != nil {
				ColorLog("[ERRO] Could not parse time: %s\n", err)
				os.Exit(2)
			} else {
				createdAt = t.Unix()
			}
		} else {
			file, createdAt = "", 0
		}
	}
	return
}

func writeMigrationSourceFile(dir, source, driver, connStr string, latestTime int64, latestName string, task string) {
	os.Chdir(dir)
	if f, err := os.OpenFile(source, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err != nil {
		ColorLog("[ERRO] Could not create file: %s\n", err)
		os.Exit(2)
	} else {
		content := strings.Replace(MIGRATION_MAIN_TPL, "{{DBDriver}}", driver, -1)
		content = strings.Replace(content, "{{ConnStr}}", connStr, -1)
		content = strings.Replace(content, "{{LatestTime}}", strconv.FormatInt(latestTime, 10), -1)
		content = strings.Replace(content, "{{LatestName}}", latestName, -1)
		content = strings.Replace(content, "{{Task}}", task, -1)
		if _, err := f.WriteString(content); err != nil {
			ColorLog("[ERRO] Could not write to file: %s\n", err)
			os.Exit(2)
		}
		f.Close()
	}
}

func buildMigrationBinary(dir, binary string) {
	os.Chdir(dir)
	cmd := exec.Command("go", "build", "-o", binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		ColorLog("[ERRO] Could not build migration binary: %s\n", err)
		formatShellErrOutput(string(out))
		os.Exit(2)
	}
}

func runMigrationBinary(dir, binary string) {
	os.Chdir(dir)
	cmd := exec.Command("./" + binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		formatShellOutput(string(out))
		ColorLog("[ERRO] Could not run migration binary: %s\n", err)
		os.Exit(2)
	} else {
		formatShellOutput(string(out))
	}
}

func removeTempFile(dir, file string) {
	os.Chdir(dir)
	if err := os.Remove(file); err != nil {
		ColorLog("[ERRO] Could not remove temporary migration files: %s\n", err)
		os.Exit(2)
	}
}

func formatShellErrOutput(o string) {
	for _, line := range strings.Split(o, "\n") {
		if line != "" {
			ColorLog("[ERRO] -| %s\n", line)
		}
	}
}

func formatShellOutput(o string) {
	for _, line := range strings.Split(o, "\n") {
		if line != "" {
			ColorLog("[INFO] -| %s\n", line)
		}
	}
}

const (
	MIGRATION_MAIN_TPL = `package main

import(
	"os"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/migration"

	_ "github.com/go-sql-driver/mysql"
)

func init(){
	orm.RegisterDataBase("default", "{{DBDriver}}","{{ConnStr}}")
}

func main(){
	task := "{{Task}}"
	switch task {
	case "upgrade":
		if err := migration.Upgrade({{LatestTime}}); err != nil {
			os.Exit(2)
		}
	case "rollback":
		if err := migration.Rollback("{{LatestName}}"); err != nil {
			os.Exit(2)
		}
	case "reset":
		if err := migration.Reset(); err != nil {
			os.Exit(2)
		}
	case "refresh":
		if err := migration.Refresh(); err != nil {
			os.Exit(2)
		}
	}
}

`
	MYSQL_MIGRATION_DDL = `
CREATE TABLE migrations (
	id_migration int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'surrogate key',
	name varchar(255) DEFAULT NULL COMMENT 'migration name, unique',
	created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'date migrated or rolled back',
	statements longtext COMMENT 'SQL statements for this migration',
	rollback_statements longtext COMMENT 'SQL statment for rolling back migration',
	status ENUM('update', 'rollback') COMMENT 'update indicates it is a normal migration while rollback means this migration is rolled back',
	PRIMARY KEY (id_migration),
	UNIQUE KEY (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 
`
)

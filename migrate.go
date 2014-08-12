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
	"strings"
)

var cmdMigrate = &Command{
	UsageLine: "migrate [Command]",
	Short:     "run database migrations",
	Long: `
bee migrate
    run all outstanding migrations

bee migrate rollback
    rollback the last migration operation

bee migrate reset
    rollback all migrations

bee migrate refresh
    rollback all migrations and run them all again
`,
}

const (
	TMP_DIR = "temp"
)

func init() {
	cmdMigrate.Run = runMigration
}

func runMigration(cmd *Command, args []string) {
	//curpath, _ := os.Getwd()

	gopath := os.Getenv("GOPATH")
	Debugf("gopath:%s", gopath)
	if gopath == "" {
		ColorLog("[ERRO] $GOPATH not found\n")
		ColorLog("[HINT] Set $GOPATH in your environment vairables\n")
		os.Exit(2)
	}
	// getting command line arguments
	connStr := "root:@tcp(127.0.0.1:3306)/sgfas?charset=utf8"
	driver := "mysql"
	if len(args) == 0 {
		// run all outstanding migrations
		ColorLog("[INFO] Running all outstanding migrations\n")
		migrateUpdate(driver, connStr)
	} else {
		mcmd := args[0]
		switch mcmd {
		case "rollback":
			ColorLog("[INFO] Rolling back the last migration operation\n")
			migrateRollback(driver, connStr)
		case "reset":
			ColorLog("[INFO] Reseting all migrations\n")
			migrateReset(driver, connStr)
		case "refresh":
			ColorLog("[INFO] Refreshing all migrations\n")
			migrateReset(driver, connStr)
		default:
			ColorLog("[ERRO] Command is missing\n")
			os.Exit(2)
		}
		ColorLog("[SUCC] Migration successful!\n")
	}
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

func getLatestMigration(db *sql.DB) (file string, createdAt string) {
	sql := "SELECT name, created_at FROM migrations where status = 'update' ORDER BY id_migration DESC LIMIT 1"
	if rows, err := db.Query(sql); err != nil {
		ColorLog("[ERRO] Could not retrieve migrations: %s\n", err)
		os.Exit(2)
	} else {
		var fileBytes, createdAtBytes []byte
		if rows.Next() {
			if err := rows.Scan(&fileBytes, &createdAtBytes); err != nil {
				ColorLog("[ERRO] Could not read migrations in database: %s\n", err)
				os.Exit(2)
			}
			file, createdAt = string(fileBytes), string(createdAtBytes)
		} else {
			file, createdAt = "", "0"
		}
	}
	return
}

func createTempMigrationDir(path string) {
	if err := os.MkdirAll(path, 0777); err != nil {
		ColorLog("[ERRO] Could not create path: %s\n", err)
		os.Exit(2)
	}
}

func writeMigrationSourceFile(filename string, driver string, connStr string, latestTime string, latestName string, task string) {
	if f, err := os.OpenFile(filename+".go", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err != nil {
		ColorLog("[ERRO] Could not create file: %s\n", err)
		os.Exit(2)
	} else {
		content := strings.Replace(MIGRATION_MAIN_TPL, "{{DBDriver}}", driver, -1)
		content = strings.Replace(content, "{{ConnStr}}", connStr, -1)
		content = strings.Replace(content, "{{LatestTime}}", latestTime, -1)
		content = strings.Replace(content, "{{LatestName}}", latestName, -1)
		content = strings.Replace(content, "{{Task}}", task, -1)
		if _, err := f.WriteString(content); err != nil {
			ColorLog("[ERRO] Could not write to file: %s\n", err)
			os.Exit(2)
		}
		f.Close()
	}
}

func buildMigrationBinary(filename string) {
	cmd := exec.Command("go", "build", "-o", filename, filename+".go")
	if err := cmd.Run(); err != nil {
		ColorLog("[ERRO] Could not build migration binary: %s\n", err)
		os.Exit(2)
	}
}

func runMigrationBinary(filename string) {
	cmd := exec.Command("./" + filename)
	if out, err := cmd.CombinedOutput(); err != nil {
		ColorLog("[ERRO] Could not run migration binary\n")
		os.Exit(2)
	} else {
		ColorLog("[INFO] %s\n", string(out))
	}
}

func cleanUpMigrationFiles(tmpPath string) {
	if err := os.RemoveAll(tmpPath); err != nil {
		ColorLog("[ERRO] Could not remove temporary migration directory: %s\n", err)
		os.Exit(2)
	}
}

func migrateUpdate(driver, connStr string) {
	migrate("upgrade", driver, connStr)
}

func migrateRollback(driver, connStr string) {
	migrate("rollback", driver, connStr)
}

func migrateReset(driver, connStr string) {
	migrate("reset", driver, connStr)
}

func migrateRefresh(driver, connStr string) {
	migrate("refresh", driver, connStr)
}

func migrate(goal, driver, connStr string) {
	filename := path.Join(TMP_DIR, "migrate")
	// connect to database
	db, err := sql.Open(driver, connStr)
	if err != nil {
		ColorLog("[ERRO] Could not connect to %s: %s\n", driver, connStr)
		os.Exit(2)
	}
	defer db.Close()
	checkForSchemaUpdateTable(db)
	latestName, latestTime := getLatestMigration(db)
	createTempMigrationDir(TMP_DIR)
	writeMigrationSourceFile(filename, driver, connStr, latestTime, latestName, goal)
	buildMigrationBinary(filename)
	runMigrationBinary(filename)
	cleanUpMigrationFiles(TMP_DIR)
}

const (
	MIGRATION_MAIN_TPL = `package main

import(
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
		migration.Upgrade({{LatestTime}})
	case "rollback":
		migration.Rollback("{{LatestName}}")
	case "reset":
		migration.Reset()
	case "refresh":
		migration.Refresh()
	}
}

`
	MYSQL_MIGRATION_DDL = `
CREATE TABLE migrations (
	id_migration int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'surrogate key',
	name varchar(255) DEFAULT NULL COMMENT 'migration name, unique',
	created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'date migrated or rolled back',
	statements longtext COMMENT 'SQL statements for this migration',
	status ENUM('update', 'rollback') COMMENT 'update indicates it is a normal migration while rollback means this migration is rolled back',
	PRIMARY KEY (id_migration),
	UNIQUE KEY (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 
`
)

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
	"strconv"
	"strings"
	"time"
)

var cmdMigrate = &Command{
	UsageLine: "migrate [Command]",
	Short:     "run database migrations",
	Long: `
bee migrate [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    run all outstanding migrations
    -driver: [mysql | postgres | sqlite] (default: mysql)
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test

bee migrate rollback [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    rollback the last migration operation
    -driver: [mysql | postgres | sqlite] (default: mysql)
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test

bee migrate reset [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    rollback all migrations
    -driver: [mysql | postgres | sqlite] (default: mysql)
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test

bee migrate refresh [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    rollback all migrations and run them all again
    -driver: [mysql | postgres | sqlite] (default: mysql)
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test
`,
}

var mDriver docValue
var mConn docValue

func init() {
	cmdMigrate.Run = runMigration
	cmdMigrate.Flag.Var(&mDriver, "driver", "database driver: mysql, postgres, sqlite, etc.")
	cmdMigrate.Flag.Var(&mConn, "conn", "connection string used by the driver to connect to a database instance")
}

// runMigration is the entry point for starting a migration
func runMigration(cmd *Command, args []string) int {
	crupath, _ := os.Getwd()

	gopath := os.Getenv("GOPATH")
	Debugf("gopath:%s", gopath)
	if gopath == "" {
		ColorLog("[ERRO] $GOPATH not found\n")
		ColorLog("[HINT] Set $GOPATH in your environment vairables\n")
		os.Exit(2)
	}
	// load config
	err := loadConfig()
	if err != nil {
		ColorLog("[ERRO] Fail to parse bee.json[ %s ]\n", err)
	}
	// getting command line arguments
	if len(args) != 0 {
		cmd.Flag.Parse(args[1:])
	}
	if mDriver == "" {
		mDriver = docValue(conf.Database.Driver)
		if mDriver == "" {
			mDriver = "mysql"
		}
	}
	if mConn == "" {
		mConn = docValue(conf.Database.Conn)
		if mConn == "" {
			mConn = "root:@tcp(127.0.0.1:3306)/test"
		}
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
			migrateRefresh(crupath, driverStr, connStr)
		default:
			ColorLog("[ERRO] Command is missing\n")
			os.Exit(2)
		}
	}
	ColorLog("[SUCC] Migration successful!\n")
	return 0
}

// migrateUpdate does the schema update
func migrateUpdate(crupath, driver, connStr string) {
	migrate("upgrade", crupath, driver, connStr)
}

// migrateRollback rolls back the latest migration
func migrateRollback(crupath, driver, connStr string) {
	migrate("rollback", crupath, driver, connStr)
}

// migrateReset rolls back all migrations
func migrateReset(crupath, driver, connStr string) {
	migrate("reset", crupath, driver, connStr)
}

// migrationRefresh rolls back all migrations and start over again
func migrateRefresh(crupath, driver, connStr string) {
	migrate("refresh", crupath, driver, connStr)
}

// migrate generates source code, build it, and invoke the binary who does the actual migration
func migrate(goal, crupath, driver, connStr string) {
	dir := path.Join(crupath, "database", "migrations")
	binary := "m"
	source := binary + ".go"
	// connect to database
	db, err := sql.Open(driver, connStr)
	if err != nil {
		ColorLog("[ERRO] Could not connect to %s: %s\n", driver, connStr)
		ColorLog("[ERRO] Error: %v", err.Error())
		os.Exit(2)
	}
	defer db.Close()
	checkForSchemaUpdateTable(db, driver)
	latestName, latestTime := getLatestMigration(db, goal)
	writeMigrationSourceFile(dir, source, driver, connStr, latestTime, latestName, goal)
	buildMigrationBinary(dir, binary)
	runMigrationBinary(dir, binary)
	removeTempFile(dir, source)
	removeTempFile(dir, binary)
}

// checkForSchemaUpdateTable checks the existence of migrations table.
// It checks for the proper table structures and creates the table using MYSQL_MIGRATION_DDL if it does not exist.
func checkForSchemaUpdateTable(db *sql.DB, driver string) {
	showTableSql := showMigrationsTableSql(driver)
	if rows, err := db.Query(showTableSql); err != nil {
		ColorLog("[ERRO] Could not show migrations table: %s\n", err)
		os.Exit(2)
	} else if !rows.Next() {
		// no migrations table, create anew
		createTableSql := createMigrationsTableSql(driver)
		ColorLog("[INFO] Creating 'migrations' table...\n")
		if _, err := db.Query(createTableSql); err != nil {
			ColorLog("[ERRO] Could not create migrations table: %s\n", err)
			os.Exit(2)
		}
	}

	// checking that migrations table schema are expected
	selectTableSql := selectMigrationsTableSql(driver)
	if rows, err := db.Query(selectTableSql); err != nil {
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

func showMigrationsTableSql(driver string) string {
	switch driver {
	case "mysql":
		return "SHOW TABLES LIKE 'migrations'"
	case "postgres":
		return "SELECT * FROM pg_catalog.pg_tables WHERE tablename = 'migrations';"
	default:
		return "SHOW TABLES LIKE 'migrations'"
	}
}

func createMigrationsTableSql(driver string) string {
	switch driver {
	case "mysql":
		return MYSQL_MIGRATION_DDL
	case "postgres":
		return POSTGRES_MIGRATION_DDL
	default:
		return MYSQL_MIGRATION_DDL
	}
}

func selectMigrationsTableSql(driver string) string {
	switch driver {
	case "mysql":
		return "DESC migrations"
	case "postgres":
		return "SELECT * FROM migrations ORDER BY id_migration;"
	default:
		return "DESC migrations"
	}
}

// getLatestMigration retrives latest migration with status 'update'
func getLatestMigration(db *sql.DB, goal string) (file string, createdAt int64) {
	sql := "SELECT name FROM migrations where status = 'update' ORDER BY id_migration DESC LIMIT 1"
	if rows, err := db.Query(sql); err != nil {
		ColorLog("[ERRO] Could not retrieve migrations: %s\n", err)
		os.Exit(2)
	} else {
		if rows.Next() {
			if err := rows.Scan(&file); err != nil {
				ColorLog("[ERRO] Could not read migrations in database: %s\n", err)
				os.Exit(2)
			}
			createdAtStr := file[len(file)-15:]
			if t, err := time.Parse("20060102_150405", createdAtStr); err != nil {
				ColorLog("[ERRO] Could not parse time: %s\n", err)
				os.Exit(2)
			} else {
				createdAt = t.Unix()
			}
		} else {
			// migration table has no 'update' record, no point rolling back
			if goal == "rollback" {
				ColorLog("[ERRO] There is nothing to rollback\n")
				os.Exit(2)
			}
			file, createdAt = "", 0
		}
	}
	return
}

// writeMigrationSourceFile create the source file based on MIGRATION_MAIN_TPL
func writeMigrationSourceFile(dir, source, driver, connStr string, latestTime int64, latestName string, task string) {
	changeDir(dir)
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

// buildMigrationBinary changes directory to database/migrations folder and go-build the source
func buildMigrationBinary(dir, binary string) {
	changeDir(dir)
	cmd := exec.Command("go", "build", "-o", binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		ColorLog("[ERRO] Could not build migration binary: %s\n", err)
		formatShellErrOutput(string(out))
		removeTempFile(dir, binary)
		removeTempFile(dir, binary+".go")
		os.Exit(2)
	}
}

// runMigrationBinary runs the migration program who does the actual work
func runMigrationBinary(dir, binary string) {
	changeDir(dir)
	cmd := exec.Command("./" + binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		formatShellOutput(string(out))
		ColorLog("[ERRO] Could not run migration binary: %s\n", err)
		removeTempFile(dir, binary)
		removeTempFile(dir, binary+".go")
		os.Exit(2)
	} else {
		formatShellOutput(string(out))
	}
}

// changeDir changes working directory to dir.
// It exits the system when encouter an error
func changeDir(dir string) {
	if err := os.Chdir(dir); err != nil {
		ColorLog("[ERRO] Could not find migration directory: %s\n", err)
		os.Exit(2)
	}
}

// removeTempFile removes a file in dir
func removeTempFile(dir, file string) {
	changeDir(dir)
	if err := os.Remove(file); err != nil {
		ColorLog("[WARN] Could not remove temporary file: %s\n", err)
	}
}

// formatShellErrOutput formats the error shell output
func formatShellErrOutput(o string) {
	for _, line := range strings.Split(o, "\n") {
		if line != "" {
			ColorLog("[ERRO] -| ")
			fmt.Println(line)
		}
	}
}

// formatShellOutput formats the normal shell output
func formatShellOutput(o string) {
	for _, line := range strings.Split(o, "\n") {
		if line != "" {
			ColorLog("[INFO] -| ")
			fmt.Println(line)
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
	_ "github.com/lib/pq"
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
	PRIMARY KEY (id_migration)
) ENGINE=InnoDB DEFAULT CHARSET=utf8
`

	POSTGRES_MIGRATION_DDL = `
CREATE TYPE migrations_status AS ENUM('update', 'rollback');

CREATE TABLE migrations (
	id_migration SERIAL PRIMARY KEY,
	name varchar(255) DEFAULT NULL,
	created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	statements text,
	rollback_statements text,
	status migrations_status
)`
)

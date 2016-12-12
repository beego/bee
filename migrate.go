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
	"runtime"
	"strconv"
	"strings"
	"time"
)

var cmdMigrate = &Command{
	UsageLine: "migrate [Command]",
	Short:     "Runs database migrations",
	Long: `The command 'migrate' allows you to run database migrations to keep it up-to-date.

  ▶ {{"To run all the migrations:"|bold}}

    $ bee migrate [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"]

  ▶ {{"To rollback the last migration:"|bold}}

    $ bee migrate rollback [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"]

  ▶ {{"To do a reset, which will rollback all the migrations:"|bold}}

    $ bee migrate reset [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"]

  ▶ {{"To update your schema:"|bold}}

    $ bee migrate refresh [-driver=mysql] [-conn="root:@tcp(127.0.0.1:3306)/test"]
`,
	PreRun: func(cmd *Command, args []string) { ShowShortVersionBanner() },
	Run:    runMigration,
}

var mDriver docValue
var mConn docValue

func init() {
	cmdMigrate.Flag.Var(&mDriver, "driver", "Database driver. Either mysql, postgres or sqlite.")
	cmdMigrate.Flag.Var(&mConn, "conn", "Connection string used by the driver to connect to a database instance.")
}

// runMigration is the entry point for starting a migration
func runMigration(cmd *Command, args []string) int {
	currpath, _ := os.Getwd()

	gps := GetGOPATHs()
	if len(gps) == 0 {
		logger.Fatal("GOPATH environment variable is not set or empty")
	}

	gopath := gps[0]

	logger.Debugf("GOPATH: %s", __FILE__(), __LINE__(), gopath)

	// Load the configuration
	err := loadConfig()
	if err != nil {
		logger.Errorf("Failed to load configuration: %s", err)
	}

	// Getting command line arguments
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
	logger.Infof("Using '%s' as 'driver'", mDriver)
	logger.Infof("Using '%s' as 'conn'", mConn)
	driverStr, connStr := string(mDriver), string(mConn)
	if len(args) == 0 {
		// run all outstanding migrations
		logger.Info("Running all outstanding migrations")
		migrateUpdate(currpath, driverStr, connStr)
	} else {
		mcmd := args[0]
		switch mcmd {
		case "rollback":
			logger.Info("Rolling back the last migration operation")
			migrateRollback(currpath, driverStr, connStr)
		case "reset":
			logger.Info("Reseting all migrations")
			migrateReset(currpath, driverStr, connStr)
		case "refresh":
			logger.Info("Refreshing all migrations")
			migrateRefresh(currpath, driverStr, connStr)
		default:
			logger.Fatal("Command is missing")
		}
	}
	logger.Success("Migration successful!")
	return 0
}

// migrateUpdate does the schema update
func migrateUpdate(currpath, driver, connStr string) {
	migrate("upgrade", currpath, driver, connStr)
}

// migrateRollback rolls back the latest migration
func migrateRollback(currpath, driver, connStr string) {
	migrate("rollback", currpath, driver, connStr)
}

// migrateReset rolls back all migrations
func migrateReset(currpath, driver, connStr string) {
	migrate("reset", currpath, driver, connStr)
}

// migrationRefresh rolls back all migrations and start over again
func migrateRefresh(currpath, driver, connStr string) {
	migrate("refresh", currpath, driver, connStr)
}

// migrate generates source code, build it, and invoke the binary who does the actual migration
func migrate(goal, currpath, driver, connStr string) {
	dir := path.Join(currpath, "database", "migrations")
	postfix := ""
	if runtime.GOOS == "windows" {
		postfix = ".exe"
	}
	binary := "m" + postfix
	source := binary + ".go"

	// Connect to database
	db, err := sql.Open(driver, connStr)
	if err != nil {
		logger.Fatalf("Could not connect to database using '%s': %s", connStr, err)
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
	showTableSQL := showMigrationsTableSQL(driver)
	if rows, err := db.Query(showTableSQL); err != nil {
		logger.Fatalf("Could not show migrations table: %s", err)
	} else if !rows.Next() {
		// No migrations table, create new ones
		createTableSQL := createMigrationsTableSQL(driver)

		logger.Infof("Creating 'migrations' table...")

		if _, err := db.Query(createTableSQL); err != nil {
			logger.Fatalf("Could not create migrations table: %s", err)
		}
	}

	// Checking that migrations table schema are expected
	selectTableSQL := selectMigrationsTableSQL(driver)
	if rows, err := db.Query(selectTableSQL); err != nil {
		logger.Fatalf("Could not show columns of migrations table: %s", err)
	} else {
		for rows.Next() {
			var fieldBytes, typeBytes, nullBytes, keyBytes, defaultBytes, extraBytes []byte
			if err := rows.Scan(&fieldBytes, &typeBytes, &nullBytes, &keyBytes, &defaultBytes, &extraBytes); err != nil {
				logger.Fatalf("Could not read column information: %s", err)
			}
			fieldStr, typeStr, nullStr, keyStr, defaultStr, extraStr :=
				string(fieldBytes), string(typeBytes), string(nullBytes), string(keyBytes), string(defaultBytes), string(extraBytes)
			if fieldStr == "id_migration" {
				if keyStr != "PRI" || extraStr != "auto_increment" {
					logger.Hint("Expecting KEY: PRI, EXTRA: auto_increment")
					logger.Fatalf("Column migration.id_migration type mismatch: KEY: %s, EXTRA: %s", keyStr, extraStr)
				}
			} else if fieldStr == "name" {
				if !strings.HasPrefix(typeStr, "varchar") || nullStr != "YES" {
					logger.Hint("Expecting TYPE: varchar, NULL: YES")
					logger.Fatalf("Column migration.name type mismatch: TYPE: %s, NULL: %s", typeStr, nullStr)
				}
			} else if fieldStr == "created_at" {
				if typeStr != "timestamp" || defaultStr != "CURRENT_TIMESTAMP" {
					logger.Hint("Expecting TYPE: timestamp, DEFAULT: CURRENT_TIMESTAMP")
					logger.Fatalf("Column migration.timestamp type mismatch: TYPE: %s, DEFAULT: %s", typeStr, defaultStr)
				}
			}
		}
	}
}

func showMigrationsTableSQL(driver string) string {
	switch driver {
	case "mysql":
		return "SHOW TABLES LIKE 'migrations'"
	case "postgres":
		return "SELECT * FROM pg_catalog.pg_tables WHERE tablename = 'migrations';"
	default:
		return "SHOW TABLES LIKE 'migrations'"
	}
}

func createMigrationsTableSQL(driver string) string {
	switch driver {
	case "mysql":
		return MYSQLMigrationDDL
	case "postgres":
		return POSTGRESMigrationDDL
	default:
		return MYSQLMigrationDDL
	}
}

func selectMigrationsTableSQL(driver string) string {
	switch driver {
	case "mysql":
		return "DESC migrations"
	case "postgres":
		return "SELECT * FROM migrations WHERE false ORDER BY id_migration;"
	default:
		return "DESC migrations"
	}
}

// getLatestMigration retrives latest migration with status 'update'
func getLatestMigration(db *sql.DB, goal string) (file string, createdAt int64) {
	sql := "SELECT name FROM migrations where status = 'update' ORDER BY id_migration DESC LIMIT 1"
	if rows, err := db.Query(sql); err != nil {
		logger.Fatalf("Could not retrieve migrations: %s", err)
	} else {
		if rows.Next() {
			if err := rows.Scan(&file); err != nil {
				logger.Fatalf("Could not read migrations in database: %s", err)
			}
			createdAtStr := file[len(file)-15:]
			if t, err := time.Parse("20060102_150405", createdAtStr); err != nil {
				logger.Fatalf("Could not parse time: %s", err)
			} else {
				createdAt = t.Unix()
			}
		} else {
			// migration table has no 'update' record, no point rolling back
			if goal == "rollback" {
				logger.Fatal("There is nothing to rollback")
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
		logger.Fatalf("Could not create file: %s", err)
	} else {
		content := strings.Replace(MigrationMainTPL, "{{DBDriver}}", driver, -1)
		content = strings.Replace(content, "{{ConnStr}}", connStr, -1)
		content = strings.Replace(content, "{{LatestTime}}", strconv.FormatInt(latestTime, 10), -1)
		content = strings.Replace(content, "{{LatestName}}", latestName, -1)
		content = strings.Replace(content, "{{Task}}", task, -1)
		if _, err := f.WriteString(content); err != nil {
			logger.Fatalf("Could not write to file: %s", err)
		}
		CloseFile(f)
	}
}

// buildMigrationBinary changes directory to database/migrations folder and go-build the source
func buildMigrationBinary(dir, binary string) {
	changeDir(dir)
	cmd := exec.Command("go", "build", "-o", binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		logger.Errorf("Could not build migration binary: %s", err)
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
		logger.Errorf("Could not run migration binary: %s", err)
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
		logger.Fatalf("Could not find migration directory: %s", err)
	}
}

// removeTempFile removes a file in dir
func removeTempFile(dir, file string) {
	changeDir(dir)
	if err := os.Remove(file); err != nil {
		logger.Warnf("Could not remove temporary file: %s", err)
	}
}

// formatShellErrOutput formats the error shell output
func formatShellErrOutput(o string) {
	for _, line := range strings.Split(o, "\n") {
		if line != "" {
			logger.Errorf("|> %s", line)
		}
	}
}

// formatShellOutput formats the normal shell output
func formatShellOutput(o string) {
	for _, line := range strings.Split(o, "\n") {
		if line != "" {
			logger.Infof("|> %s", line)
		}
	}
}

const (
	// MigrationMainTPL migration main template
	MigrationMainTPL = `package main

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
	// MYSQLMigrationDDL MySQL migration SQL
	MYSQLMigrationDDL = `
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
	// POSTGRESMigrationDDL Postgres migration SQL
	POSTGRESMigrationDDL = `
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

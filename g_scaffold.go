package main

import "strings"

func generateScaffold(sname, fields, currpath, driver, conn string) {
	ColorLog("[INFO] Do you want to create a '%v' model? [Yes|No] ", sname)

	// Generate the model
	if askForConfirmation() {
		generateModel(sname, fields, currpath)
	}

	// Generate the controller
	ColorLog("[INFO] Do you want to create a '%v' controller? [Yes|No] ", sname)
	if askForConfirmation() {
		generateController(sname, currpath)
	}

	// Generate the views
	ColorLog("[INFO] Do you want to create views for this '%v' resource? [Yes|No] ", sname)
	if askForConfirmation() {
		generateView(sname, currpath)
	}

	// Generate a migration
	ColorLog("[INFO] Do you want to create a '%v' migration and schema for this resource? [Yes|No] ", sname)
	if askForConfirmation() {
		upsql := ""
		downsql := ""
		if fields != "" {
			dbMigrator := newDBDriver()
			upsql = dbMigrator.generateCreateUp(sname)
			downsql = dbMigrator.generateCreateDown(sname)
			//todo remove
			//if driver == "" {
			//	downsql = strings.Replace(downsql, "`", "", -1)
			//}
		}
		generateMigration(sname, upsql, downsql, currpath)
	}

	// Run the migration
	ColorLog("[INFO] Do you want to migrate the database? [Yes|No] ")
	if askForConfirmation() {
		migrateUpdate(currpath, driver, conn)
	}
	ColorLog("[INFO] All done! Don't forget to add  beego.Router(\"/%v\" ,&controllers.%vController{}) to routers/route.go\n", sname, strings.Title(sname))
}

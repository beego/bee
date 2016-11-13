package main

import "strings"

func generateScaffold(sname, fields, currpath, driver, conn string) {
	logger.Infof("Do you want to create a '%s' model? [Yes|No] ", sname)

	// Generate the model
	if askForConfirmation() {
		generateModel(sname, fields, currpath)
	}

	// Generate the controller
	logger.Infof("Do you want to create a '%s' controller? [Yes|No] ", sname)
	if askForConfirmation() {
		generateController(sname, currpath)
	}

	// Generate the views
	logger.Infof("Do you want to create views for this '%s' resource? [Yes|No] ", sname)
	if askForConfirmation() {
		generateView(sname, currpath)
	}

	// Generate a migration
	logger.Infof("Do you want to create a '%s' migration and schema for this resource? [Yes|No] ", sname)
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
	logger.Infof("Do you want to migrate the database? [Yes|No] ")
	if askForConfirmation() {
		migrateUpdate(currpath, driver, conn)
	}
	logger.Successf("All done! Don't forget to add  beego.Router(\"/%s\" ,&controllers.%sController{}) to routers/route.go\n", sname, strings.Title(sname))
}

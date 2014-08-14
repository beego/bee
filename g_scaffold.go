package main

import "strings"

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
		generateMigration(sname, crupath)
	}
	// run migration
	ColorLog("[INFO] Do you want to go ahead and migrate the database? [yes|no]]  ")
	if askForConfirmation() {
		migrateUpdate(crupath, driver, conn)
	}
	ColorLog("[INFO] All done! Don't forget to add  beego.Router(\"/%v\" ,&controllers.%vController{}) to routers/route.go\n", sname, strings.Title(sname))
}

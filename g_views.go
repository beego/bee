package main

import (
	"os"
	"path"
)

// recipe
// admin/recipe
func generateView(vpath, crupath string) {
	absvpath := path.Join(crupath, "views", vpath)
	os.MkdirAll(absvpath, os.ModePerm)
	cfile := path.Join(absvpath, "index.tpl")
	if f, err := os.OpenFile(cfile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer f.Close()
		f.WriteString(cfile)
		ColorLog("[INFO] Created: %v\n", cfile)
	} else {
		ColorLog("[ERRO] Could not create view file: %s\n", err)
		os.Exit(2)
	}
	cfile = path.Join(absvpath, "show.tpl")
	if f, err := os.OpenFile(cfile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer f.Close()
		f.WriteString(cfile)
		ColorLog("[INFO] Created: %v\n", cfile)
	} else {
		ColorLog("[ERRO] Could not create view file: %s\n", err)
		os.Exit(2)
	}
	cfile = path.Join(absvpath, "create.tpl")
	if f, err := os.OpenFile(cfile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer f.Close()
		f.WriteString(cfile)
		ColorLog("[INFO] Created: %v\n", cfile)
	} else {
		ColorLog("[ERRO] Could not create view file: %s\n", err)
		os.Exit(2)
	}
	cfile = path.Join(absvpath, "edit.tpl")
	if f, err := os.OpenFile(cfile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer f.Close()
		f.WriteString(cfile)
		ColorLog("[INFO] Created: %v\n", cfile)
	} else {
		ColorLog("[ERRO] Could not create view file: %s\n", err)
		os.Exit(2)
	}
}

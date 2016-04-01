package main

import (
	"os"
	"path"
	"strings"
)

func generateHelper(cname, crupath string) {
	p, f := path.Split(cname)
	helperName := strings.Title(f)
	packageName := "helpers"
	if p != "" {
		i := strings.LastIndex(p[:len(p)-1], "/")
		packageName = p[i+1 : len(p)-1]
	}
	ColorLog("[INFO] Using '%s' as helpers name\n", helperName)
	ColorLog("[INFO] Using '%s' as package name\n", packageName)
	fp := path.Join(crupath, "helpers", p)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		// create controller directory
		if err := os.MkdirAll(fp, 0777); err != nil {
			ColorLog("[ERRO] Could not create helpers directory: %s\n", err)
			os.Exit(2)
		}
	}
	fpath := path.Join(fp, strings.ToLower(helperName)+"_helper.go")
	if f, err := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666); err == nil {
		defer f.Close()
		content := strings.Replace(BASE_HELPER_TPL, "{{packageName}}", packageName, -1)
		content = strings.Replace(content, "{{helperName}}", helperName, -1)
		f.WriteString(content)
		// gofmt generated source code
		formatSourceCode(fpath)
		ColorLog("[INFO] helpers file generated: %s\n", fpath)
	} else {
		// error creating file
		ColorLog("[ERRO] Could not create helper file: %s\n", err)
		os.Exit(2)
	}
}

const (
	BASE_HELPER_TPL = `package {{packageName}}

	func {{helperName}}() {

	}
	`
)

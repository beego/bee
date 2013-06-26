package main

import (
	"fmt"
	"os"
	"path"
	"strings"
)

var cmdCreate = &Command{
	UsageLine: "create [appname]",
	Short:     "create an application base on beego framework",
	Long: `
create an application base on beego framework

In the current path, will create a folder named [appname]

In the appname folder has the follow struct:

	|- main.go
	|- conf
	    |-  app.conf
	|- controllers
	     |- default.go
	|- models
	|- static
	     |- js
		 |- css
	     |- img				
	|- views
	    index.tpl					

`,
}

func init() {
	cmdCreate.Run = createapp
}

func createapp(cmd *Command, args []string) {
	crupath, _ := os.Getwd()
	if len(args) != 1 {
		fmt.Println("error args")
		os.Exit(2)
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		fmt.Println("you should set GOPATH in the env")
		os.Exit(2)
	}
	haspath := false
	appsrcpath := ""
	if crupath != path.Join(gopath, "src") {
		wgopath := strings.Split(gopath, ";")
		if len(wgopath) >= 1 {
			for _, wg := range wgopath {
				wg = wg + `\src`
				if strings.HasPrefix(crupath, wg) {
					haspath = true
					appsrcpath = path.Join(strings.TrimPrefix(crupath, wg), args[0])
					break
				}
			}
		}
		if !haspath {
			lgopath := strings.Split(gopath, ":")
			if len(lgopath) >= 1 {
				for _, wg := range lgopath {
					if strings.HasPrefix(crupath, path.Join(wg, "src")) {
						haspath = true
						appsrcpath = path.Join(strings.TrimPrefix(crupath, path.Join(wg, "src")), args[0])
						break
					}
				}
			}
		}

	} else {
		haspath = true
		appsrcpath = args[0]
	}
	if !haspath {
		fmt.Println("can't create application outside of GOPATH")
		fmt.Println("you first should `cd $GOPATH/src` then use create")
		os.Exit(2)
	}
	apppath := path.Join(crupath, args[0])
	os.Mkdir(apppath, 0755)
	fmt.Println("create app folder:", apppath)
	os.Mkdir(path.Join(apppath, "conf"), 0755)
	fmt.Println("create conf:", path.Join(apppath, "conf"))
	os.Mkdir(path.Join(apppath, "controllers"), 0755)
	fmt.Println("create controllers:", path.Join(apppath, "controllers"))
	os.Mkdir(path.Join(apppath, "models"), 0755)
	fmt.Println("create models:", path.Join(apppath, "models"))
	os.Mkdir(path.Join(apppath, "static"), 0755)
	fmt.Println("create static:", path.Join(apppath, "static"))
	os.Mkdir(path.Join(apppath, "static", "js"), 0755)
	fmt.Println("create static js:", path.Join(apppath, "static", "js"))
	os.Mkdir(path.Join(apppath, "static", "css"), 0755)
	fmt.Println("create static css:", path.Join(apppath, "static", "css"))
	os.Mkdir(path.Join(apppath, "static", "img"), 0755)
	fmt.Println("create static img:", path.Join(apppath, "static", "img"))
	fmt.Println("create views:", path.Join(apppath, "views"))
	os.Mkdir(path.Join(apppath, "views"), 0755)
	fmt.Println("create conf app.conf:", path.Join(apppath, "conf", "app.conf"))
	writetofile(path.Join(apppath, "conf", "app.conf"), strings.Replace(appconf, "{{.Appname}}", args[0], -1))

	fmt.Println("create controllers default.go:", path.Join(apppath, "controllers", "default.go"))
	writetofile(path.Join(apppath, "controllers", "default.go"), controllers)

	fmt.Println("create views index.tpl:", path.Join(apppath, "views", "index.tpl"))
	writetofile(path.Join(apppath, "views", "index.tpl"), indextpl)

	fmt.Println("create main.go:", path.Join(apppath, "main.go"))
	writetofile(path.Join(apppath, "main.go"), strings.Replace(maingo, "{{.Appname}}", strings.TrimPrefix(strings.Replace(appsrcpath, "\\", "/", -1), "/"), -1))
}

var appconf = `
appname = {{.Appname}}
httpport = 8080
runmode = dev
`

var maingo = `package main

import (
	"{{.Appname}}/controllers"
	"github.com/astaxie/beego"
)

func main() {
	beego.Router("/", &controllers.MainController{})
	beego.Run()
}

`
var controllers = `package controllers

import (
	"github.com/astaxie/beego"
)

type MainController struct {
	beego.Controller
}

func (this *MainController) Get() {
	this.Data["Username"] = "astaxie"
	this.Data["Email"] = "astaxie@gmail.com"
	this.TplNames = "index.tpl"
}
`

var indextpl = `<!DOCTYPE html>
<html>
  <head>
    <title>beego welcome template</title>
  </head>
  <body>
    <h1>Hello, world!{{.Username}},{{.Email}}</h1>
  </body>
</html>
`

func writetofile(filename, content string) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(content)
}

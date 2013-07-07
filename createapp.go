package main

import (
	"fmt"
	"os"
	path "path/filepath"
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
	curpath, _ := os.Getwd()
	if len(args) != 1 {
		fmt.Println("error args")
		os.Exit(2)
	}

	gopath := os.Getenv("GOPATH")
	Debugf("gopath:%s", gopath)
	if gopath == "" {
		fmt.Println("you should set GOPATH in the env")
		os.Exit(2)
	}
	haspath := false
	appsrcpath := ""

	wgopath := path.SplitList(gopath)
	for _, wg := range wgopath {
		wg = path.Join(wg, "src")

		if path.HasPrefix(strings.ToLower(curpath), strings.ToLower(wg)) {
			haspath = true
			appsrcpath = wg
			break
		}
	}

	if !haspath {
		fmt.Printf("can't create application outside of GOPATH `%s`\n", gopath)
		fmt.Printf("you first should `cd $GOPATH%ssrc` then use create\n", string(path.Separator))
		os.Exit(2)
	}

	apppath := path.Join(curpath, args[0])

	if _, err := os.Stat(apppath); os.IsNotExist(err) == false {
		fmt.Printf("path `%s` exists, can not create app without remove it\n", apppath)
		os.Exit(2)
	}

	os.MkdirAll(apppath, 0755)
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
	writetofile(path.Join(apppath, "main.go"), strings.Replace(maingo, "{{.Appname}}", strings.Join(strings.Split(apppath[len(appsrcpath)+1:], string(path.Separator)), "/"), -1))
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

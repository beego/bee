package main

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/astaxie/beego"
)

var cmdVersion = &Command{
	UsageLine: "version",
	Short:     "show the bee & beego version",
	Long: `
show the bee & beego version                  

bee version
    bee: 1.1.1
	beego: 1.2
`,
}

func init() {
	cmdVersion.Run = versionCmd
}

func versionCmd(cmd *Command, args []string) {
	fmt.Println("bee   :" + version)
	fmt.Println("beego :" + beego.VERSION)
	goversion, err := exec.Command("go", "version").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Go    :" + string(goversion))
}

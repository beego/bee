package main

import (
	"encoding/json"
	"fmt"
	"os"
	path "path/filepath"
	"runtime"
)

var cmdRun = &Command{
	UsageLine: "run [appname]",
	Short:     "run the app which can hot compile",
	Long: `
start the appname throw exec.Command

then start a inotify watch for current dir
										
when the file has changed bee will auto go build and restart the app

	file changed
	     |
  check if it's go file
	     |
     yes     no
      |       |
 go build    do nothing
     |
 restart app   
`,
}

var defaultJson = `
{
	"go_install": false,
	"dir_structure":{
		"controllers": "",
		"models": "",
		"others": []
	},
	"main_files":{
		"main.go": "",
		"others": []
	}
}
`

func init() {
	cmdRun.Run = runApp
}

var appname string
var conf struct {
	// Indicates whether execute "go install" before "go build".
	GoInstall bool `json:"go_install"`

	DirStruct struct {
		Controllers string
		Models      string
		Others      []string // Other directories.
	} `json:"dir_structure"`

	MainFiles struct {
		Main   string   `json:"main.go"`
		Others []string // Others files of package main.
	} `json:"main_files"`
}

func runApp(cmd *Command, args []string) {
	exit := make(chan bool)
	if len(args) != 1 {
		fmt.Println("[ERRO] Argument [appname] is missing")
		os.Exit(2)
	}
	crupath, _ := os.Getwd()
	Debugf("current path:%s\n", crupath)

	err := loadConfig()
	if err != nil {
		fmt.Println("[ERRO] Fail to parse bee.json:", err)
	}
	var paths []string
	paths = append(paths,
		path.Join(crupath, conf.DirStruct.Controllers),
		path.Join(crupath, conf.DirStruct.Models),
		path.Join(crupath, "./")) // Current path.
	// Because monitor files has some issues, we watch current directory
	// and ignore non-go files.
	paths = append(paths, conf.DirStruct.Others...)
	paths = append(paths, conf.MainFiles.Others...)

	NewWatcher(paths)
	appname = args[0]
	Autobuild()
	for {
		select {
		case <-exit:
			runtime.Goexit()
		}
	}
}

// loadConfig loads customized configuration.
func loadConfig() error {
	f, err := os.Open("bee.json")
	if err != nil {
		// Use default.
		err = json.Unmarshal([]byte(defaultJson), &conf)
		if err != nil {
			return err
		}
	} else {
		defer f.Close()
		fmt.Println("[INFO] Detected bee.json")
		d := json.NewDecoder(f)
		err = d.Decode(&conf)
		if err != nil {
			return err
		}
	}
	// Set variables.
	if len(conf.DirStruct.Controllers) == 0 {
		conf.DirStruct.Controllers = "controllers"
	}
	if len(conf.DirStruct.Models) == 0 {
		conf.DirStruct.Models = "models"
	}
	if len(conf.MainFiles.Main) == 0 {
		conf.MainFiles.Main = "main.go"
	}
	return nil
}

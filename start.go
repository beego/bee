package main

import (
	"fmt"
	"os"
	path "path/filepath"
	"runtime"
)

var cmdStart = &Command{
	UsageLine: "start [appname]",
	Short:     "start the app which can hot compile",
	Long: `
start the appname throw exec.Command

then start a inotify watch for current dir
										
when the file has changed bee will auto go build and restart the app

	file changed
	     |
  checked is go file
	     |
     yes     no
      |       |
 go build    do nothing
     |
 restart app   
`,
}

func init() {
	cmdStart.Run = startapp
}

var appname string

func startapp(cmd *Command, args []string) {
	if len(args) != 1 {
		fmt.Println("error args")
		os.Exit(2)
	}
	crupath, _ := os.Getwd()
	Debugf("current path:%s\n", crupath)

	var paths []string
	paths = append(paths, path.Join(crupath, "controllers"), path.Join(crupath, "models"))
	NewWatcher(paths)
	appname = args[0]
	Autobuild()
	for {
		runtime.Gosched()
	}
}

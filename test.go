package main

import (
	"bytes"
	"os"
	"os/exec"
	path "path/filepath"
	"time"

	"github.com/Unknwon/com"
)

var cmdTest = &Command{
	UsageLine: "test [appname]",
	Short:     "test the app",
	Long:      ``,
}

func init() {
	cmdTest.Run = testApp
}

var started = make(chan bool)

func testApp(cmd *Command, args []string) {
	if len(args) != 1 {
		com.ColorLog("[ERRO] Cannot start running[ %s ]\n",
			"argument 'appname' is missing")
		os.Exit(2)
	}
	crupath, _ := os.Getwd()
	Debugf("current path:%s\n", crupath)

	err := loadConfig()
	if err != nil {
		com.ColorLog("[ERRO] Fail to parse bee.json[ %s ]", err)
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
		case <-started:
			runTest()
			Kill()
			os.Exit(0)
		}
	}
}

func runTest() {
	com.ColorLog("[INFO] Start testing...\n")
	time.Sleep(time.Second * 5)
	path, _ := os.Getwd()
	os.Chdir(path + "/tests")

	var err error
	icmd := exec.Command("go", "test")
	var out, errbuffer bytes.Buffer
	icmd.Stdout = &out
	icmd.Stderr = &errbuffer
	com.ColorLog("[INFO] ============== Test Begin ===================\n")
	err = icmd.Run()
	com.ColorLog(out.String())
	com.ColorLog(errbuffer.String())
	com.ColorLog("[INFO] ============== Test End ===================\n")

	if err != nil {
		com.ColorLog("[ERRO] ============== Test failed ===================\n")
		com.ColorLog("[ERRO] ", err)
		return
	}
	com.ColorLog("[SUCC] Test finish\n")
}

package main

import (
	"os"
	"os/exec"
	path "path/filepath"
	"time"

//	"bytes"
)

var cmdTest = &Command{
	UsageLine: "test [appname]",
	Short:     "test the app",
	Long:      ``,
}

func init() {
	cmdTest.Run = testApp
}

func safePathAppend(arr []string, paths ...string) []string {
	for _, path := range paths {
		if pathExists(path) {
			arr = append(arr, path)
		}
	}
	return arr
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

var started = make(chan bool)

func testApp(cmd *Command, args []string) {
	if len(args) != 1 {
		colorLog("[ERRO] Cannot start running[ %s ]\n",
			"argument 'appname' is missing")
		os.Exit(2)
	}
	crupath, _ := os.Getwd()
	Debugf("current path:%s\n", crupath)

	err := loadConfig()
	if err != nil {
		colorLog("[ERRO] Fail to parse bee.json[ %s ]", err)
	}
	var paths []string
	paths = safePathAppend(paths,
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
			//Kill()
			//os.Exit(0)
		}
	}
}

func runTest() {
	colorLog("[INFO] Start testing...\n")
	time.Sleep(time.Second * 1)
	crupwd, _ := os.Getwd()
	testDir := path.Join(crupwd, "tests")
	if pathExists(testDir) {
		os.Chdir(testDir)
	}

	var err error
	icmd := exec.Command("go", "test")
	//var out,errbuffer bytes.Buffer
	//icmd.Stdout = &out
	//icmd.Stderr = &errbuffer
	icmd.Stdout = os.Stdout
	icmd.Stderr = os.Stderr
	colorLog("[TRAC] ============== Test Begin ===================\n")
	err = icmd.Run()
	//colorLog(out.String())
	//colorLog(errbuffer.String())
	colorLog("[TRAC] ============== Test End ===================\n")

	if err != nil {
		colorLog("[ERRO] ============== Test failed ===================\n")
		colorLog("[ERRO] ", err)
		return
	}
	colorLog("[SUCC] Test finish\n")
}

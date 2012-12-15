package main

import (
	"bytes"
	"fmt"
	"github.com/howeyc/fsnotify"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
)

var (
	builderror chan string
	restart    chan bool
	cmd        *exec.Cmd
)

func init() {
	builderror = make(chan string)
	restart = make(chan bool)
}

func NewWatcher(paths []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case e := <-watcher.Event:
				fmt.Println(e)
				go Autobuild()
			case err := <-watcher.Error:
				log.Fatal("error:", err)
			}
		}
	}()
	for _, path := range paths {
		fmt.Println(path)
		err = watcher.Watch(path)
		if err != nil {
			log.Fatal(err)
		}
	}

}

func Autobuild() {
	defer func() {
		if err := recover(); err != nil {
			str := ""
			for i := 1; ; i += 1 {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				str = str + fmt.Sprintf("%v,%v", file, line)
			}
			builderror <- str

		}
	}()
	fmt.Println("Autobuild")
	path, _ := os.Getwd()
	os.Chdir(path)
	bcmd := exec.Command("go", "build")
	var out bytes.Buffer
	var berr bytes.Buffer
	bcmd.Stdout = &out
	bcmd.Stderr = &berr
	err := bcmd.Run()
	if err != nil {
		fmt.Println("run error", err)
	}
	if out.String() == "" {
		Kill()
	} else {
		builderror <- berr.String()
	}
}

func Kill() {
	err := cmd.Process.Kill()
	if err != nil {
		panic(err)
	}
}

func Start(appname string) {
	fmt.Println("start", appname)
	cmd = exec.Command(appname)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("stdout:", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("stdin:", err)
	}
	r := io.MultiReader(stdout, stderr)
	err = cmd.Start()
	if err != nil {
		fmt.Println("cmd start:", err)
	}
	for {
		buf := make([]byte, 1024)
		count, err := r.Read(buf)
		if err != nil || count == 0 {
			fmt.Println("process exit")
			restart <- true
			return
		} else {
			fmt.Println("result:", string(buf))
		}
	}
}

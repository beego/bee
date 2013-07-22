package main

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
	"strings"
)

var (
	cmd       *exec.Cmd
	state     sync.Mutex
	eventTime = make(map[string]time.Time)
)

func NewWatcher(paths []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case e := <-watcher.Event:
				isbuild := true
				if t, ok := eventTime[e.String()]; ok {
					// if 500ms change many times, then ignore it.
					// for liteide often gofmt code after save.
					if t.Add(time.Millisecond * 500).After(time.Now()) {
						isbuild = false
					}
				}
				eventTime[e.String()] = time.Now()

				if isbuild {
					fmt.Println(e)
					go Autobuild()
				}
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
	state.Lock()
	defer state.Unlock()

	fmt.Println("start autobuild")
	path, _ := os.Getwd()
	os.Chdir(path)
	bcmd := exec.Command("go", "build")
	bcmd.Stdout = os.Stdout
	bcmd.Stderr = os.Stderr
	err := bcmd.Run()

	if err != nil {
		fmt.Println("============== build failed ===================")
		return
	}
	fmt.Println("build success")
	Restart(appname)
}

func Kill() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Kill -> ", e)
		}
	}()
	if cmd != nil {
		cmd.Process.Kill()
	}
}

func Restart(appname string) {
	Debugf("kill running process")
	Kill()
	go Start(appname)
}

func Start(appname string) {
	fmt.Println("start", appname)
	
	if strings.Index(appname, "./") == -1 {
		appname = "./" + appname
	}

	cmd = exec.Command(appname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go cmd.Run()
}

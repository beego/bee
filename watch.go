package main

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	//"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

var (
	builderror chan string
	restart    chan bool
	cmd        *exec.Cmd
	state      sync.Mutex
	running    bool
)

func init() {
	builderror = make(chan string)
	//restart = make(chan bool)
	running = false
}

var eventTime = make(map[string]time.Time)

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
					//Debugf("%s pk %s", t, time.Now())
					if t.Add(time.Millisecond * 500).After(time.Now()) {
						isbuild = false
					}
				}
				Debugf("isbuild:%v", isbuild)
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
	fmt.Println("autobuild")
	path, _ := os.Getwd()
	os.Chdir(path)
	bcmd := exec.Command("go", "build")
	bcmd.Stdout = os.Stdout
	bcmd.Stderr = os.Stderr
	err := bcmd.Run()
	if err != nil {
		builderror <- err.Error()
		return
	}
	Restart(appname)
}

func Kill() {
	err := cmd.Process.Kill()
	if err != nil {
		panic(err)
	}
}

func Restart(appname string) {
	Kill()
	go Start()
}
func Start(appname string) {
	fmt.Println("start", appname)

	cmd = exec.Command(appname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ch := Go(func() error {
		state.Lock()
		defer state.Unlock()
		running = true
		return cmd.Run()
	})
	for {
		select {
		case err := <-ch:
			fmt.Println("cmd start error: ", err)
			state.Lock()
			defer state.Unlock()
			running = false
			return
		case <-time.After(2 * time.Second):
		}
	}

	//stdout, err := cmd.StdoutPipe()

	//if err != nil {
	//fmt.Println("stdout:", err)
	//}
	//stderr, err := cmd.StderrPipe()
	//if err != nil {
	//fmt.Println("stdin:", err)
	//}
	//r := io.MultiReader(stdout, stderr)
	//err = cmd.Start()
	//if err != nil {
	//	fmt.Println("cmd start:", err)
	//}
	//for {
	//	buf := make([]byte, 1024)
	//	count, err := r.Read(buf)
	//	if err != nil || count == 0 {
	//		fmt.Println("process exit")
	//		restart <- true
	//		return
	//	} else {
	//		fmt.Println("result:", string(buf))
	//	}
	//}
}

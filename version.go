package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	path "path/filepath"
	"regexp"
	"strings"
)

var cmdVersion = &Command{
	UsageLine: "version",
	Short:     "Prints the current Bee version",
	Long: `
Prints the current Bee, Beego and Go version alongside the platform information.

`,
}

const verboseVersionBanner string = `%s%s______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v{{ .BeeVersion }}%s
%s%s
├── Beego     : {{ .BeegoVersion }}
├── GoVersion : {{ .GoVersion }}
├── GOOS      : {{ .GOOS }}
├── GOARCH    : {{ .GOARCH }}
├── NumCPU    : {{ .NumCPU }}
├── GOPATH    : {{ .GOPATH }}
├── GOROOT    : {{ .GOROOT }}
├── Compiler  : {{ .Compiler }}
└── Date      : {{ Now "Monday, 2 Jan 2006" }}%s
`

const shortVersionBanner = `%s%s______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v{{ .BeeVersion }}%s
`

func init() {
	cmdVersion.Run = versionCmd
}

func versionCmd(cmd *Command, args []string) int {
	ShowVerboseVersionBanner()
	return 0
}

// ShowVerboseVersionBanner prints the verbose version banner
func ShowVerboseVersionBanner() {
	w := NewColorWriter(os.Stdout)
	coloredBanner := fmt.Sprintf(verboseVersionBanner, "\x1b[35m", "\x1b[1m", "\x1b[0m",
		"\x1b[32m", "\x1b[1m", "\x1b[0m")
	InitBanner(w, bytes.NewBufferString(coloredBanner))
}

// ShowShortVersionBanner prints the short version banner
func ShowShortVersionBanner() {
	w := NewColorWriter(os.Stdout)
	coloredBanner := fmt.Sprintf(shortVersionBanner, "\x1b[35m", "\x1b[1m", "\x1b[0m")
	InitBanner(w, bytes.NewBufferString(coloredBanner))
}

func getBeegoVersion() string {
	gopath := os.Getenv("GOPATH")
	re, err := regexp.Compile(`VERSION = "([0-9.]+)"`)
	if err != nil {
		return ""
	}
	if gopath == "" {
		err = fmt.Errorf("You need to set GOPATH environment variable")
		return ""
	}
	wgopath := path.SplitList(gopath)
	for _, wg := range wgopath {
		wg, _ = path.EvalSymlinks(path.Join(wg, "src", "github.com", "astaxie", "beego"))
		filename := path.Join(wg, "beego.go")
		_, err := os.Stat(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			logger.Error("Error while getting stats of 'beego.go'")
		}
		fd, err := os.Open(filename)
		if err != nil {
			logger.Error("Error while reading 'beego.go'")
			continue
		}
		reader := bufio.NewReader(fd)
		for {
			byteLine, _, er := reader.ReadLine()
			if er != nil && er != io.EOF {
				return ""
			}
			if er == io.EOF {
				break
			}
			line := string(byteLine)
			s := re.FindStringSubmatch(line)
			if len(s) >= 2 {
				return s[1]
			}
		}

	}
	return "Beego is not installed. Please do consider installing it first: https://github.com/astaxie/beego"
}

func getGoVersion() string {
	var (
		cmdOut []byte
		err    error
	)

	if cmdOut, err = exec.Command("go", "version").Output(); err != nil {
		logger.Fatalf("There was an error running 'go version' command: %s", err)
	}
	return strings.Split(string(cmdOut), " ")[2]
}

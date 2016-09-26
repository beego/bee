// Copyright 2013 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Go is a basic promise implementation: it wraps calls a function in a goroutine
// and returns a channel which will later return the function's return value.
func Go(f func() error) chan error {
	ch := make(chan error)
	go func() {
		ch <- f()
	}()
	return ch
}

// Debugf outputs a formtted debug message, when os.env DEBUG is set.
func Debugf(format string, a ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "<unknown>"
			line = -1
		} else {
			file = filepath.Base(file)
		}
		fmt.Fprintf(os.Stderr, fmt.Sprintf("[debug] %s:%d %s\n", file, line, format), a...)
	}
}

const (
	Gray = uint8(iota + 90)
	Red
	Green
	Yellow
	Blue
	Magenta
	//NRed      = uint8(31) // Normal
	EndColor = "\033[0m"

	INFO = "INFO"
	TRAC = "TRAC"
	ERRO = "ERRO"
	WARN = "WARN"
	SUCC = "SUCC"
)

// ColorLog colors log and print to stdout.
// See color rules in function 'ColorLogS'.
func ColorLog(format string, a ...interface{}) {
	fmt.Print(ColorLogS(format, a...))
}

// ColorLogS colors log and return colored content.
// Log format: <level> <content [highlight][path]> [ error ].
// Level: TRAC -> blue; ERRO -> red; WARN -> Magenta; SUCC -> green; others -> default.
// Content: default; path: yellow; error -> red.
// Level has to be surrounded by "[" and "]".
// Highlights have to be surrounded by "# " and " #"(space), "#" will be deleted.
// Paths have to be surrounded by "( " and " )"(space).
// Errors have to be surrounded by "[ " and " ]"(space).
// Note: it hasn't support windows yet, contribute is welcome.
func ColorLogS(format string, a ...interface{}) string {
	log := fmt.Sprintf(format, a...)

	var clog string

	if runtime.GOOS != "windows" {
		// Level.
		i := strings.Index(log, "]")
		if log[0] == '[' && i > -1 {
			clog += "[" + getColorLevel(log[1:i]) + "]"
		}

		log = log[i+1:]

		// Error.
		log = strings.Replace(log, "[ ", fmt.Sprintf("[\033[%dm", Red), -1)
		log = strings.Replace(log, " ]", EndColor+"]", -1)

		// Path.
		log = strings.Replace(log, "( ", fmt.Sprintf("(\033[%dm", Yellow), -1)
		log = strings.Replace(log, " )", EndColor+")", -1)

		// Highlights.
		log = strings.Replace(log, "# ", fmt.Sprintf("\033[%dm", Gray), -1)
		log = strings.Replace(log, " #", EndColor, -1)

		log = clog + log

	} else {
		// Level.
		i := strings.Index(log, "]")
		if log[0] == '[' && i > -1 {
			clog += "[" + log[1:i] + "]"
		}

		log = log[i+1:]

		// Error.
		log = strings.Replace(log, "[ ", "[", -1)
		log = strings.Replace(log, " ]", "]", -1)

		// Path.
		log = strings.Replace(log, "( ", "(", -1)
		log = strings.Replace(log, " )", ")", -1)

		// Highlights.
		log = strings.Replace(log, "# ", "", -1)
		log = strings.Replace(log, " #", "", -1)

		log = clog + log
	}

	return time.Now().Format("2006/01/02 15:04:05 ") + log
}

// getColorLevel returns colored level string by given level.
func getColorLevel(level string) string {
	level = strings.ToUpper(level)
	switch level {
	case INFO:
		return fmt.Sprintf("\033[%dm%s\033[0m", Blue, level)
	case TRAC:
		return fmt.Sprintf("\033[%dm%s\033[0m", Blue, level)
	case ERRO:
		return fmt.Sprintf("\033[%dm%s\033[0m", Red, level)
	case WARN:
		return fmt.Sprintf("\033[%dm%s\033[0m", Magenta, level)
	case SUCC:
		return fmt.Sprintf("\033[%dm%s\033[0m", Green, level)
	default:
		return level
	}
}

// IsExist returns whether a file or directory exists.
func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// GetGOPATHs returns all paths in GOPATH variable.
func GetGOPATHs() []string {
	gopath := os.Getenv("GOPATH")
	var paths []string
	if runtime.GOOS == "windows" {
		gopath = strings.Replace(gopath, "\\", "/", -1)
		paths = strings.Split(gopath, ";")
	} else {
		paths = strings.Split(gopath, ":")
	}
	return paths
}

func isBeegoProject(thePath string) bool {
	mainFiles := []string{}
	hasBeegoRegex := regexp.MustCompile(`(?s)package main.*?import.*?\(.*?github.com/astaxie/beego".*?\).*func main()`)
	// Walk the application path tree to look for main files.
	// Main files must satisfy the 'hasBeegoRegex' regular expression.
	err := filepath.Walk(thePath, func(fpath string, f os.FileInfo, err error) error {
		if !f.IsDir() { // Skip sub-directories
			data, _err := ioutil.ReadFile(fpath)
			if _err != nil {
				return _err
			}
			if len(hasBeegoRegex.Find(data)) > 0 {
				mainFiles = append(mainFiles, fpath)
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Unable to walk '%s' tree: %v", thePath, err)
		return false
	}

	if len(mainFiles) > 0 {
		return true
	}
	return false
}

// SearchGOPATHs searchs the user GOPATH(s) for the specified application name.
// It returns a boolean, the application's GOPATH and its full path.
func SearchGOPATHs(app string) (bool, string, string) {
	gps := GetGOPATHs()
	if len(gps) == 0 {
		ColorLog("[ERRO] Fail to start [ %s ]\n", "GOPATH environment variable is not set or empty")
		os.Exit(2)
	}

	// Lookup the application inside the user workspace(s)
	for _, gopath := range gps {
		var currentPath string

		if !strings.Contains(app, "src") {
			gopathsrc := path.Join(gopath, "src")
			currentPath = path.Join(gopathsrc, app)
		} else {
			currentPath = app
		}

		if isExist(currentPath) {
			if !isBeegoProject(currentPath) {
				continue
			}
			return true, gopath, currentPath
		}
	}
	return false, "", ""
}

// askForConfirmation uses Scanln to parse user input. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user. Typically, you should use fmt to print out a question
// before calling askForConfirmation. E.g. fmt.Println("WARNING: Are you sure? (yes/no)")
func askForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return askForConfirmation()
	}
}

func containsString(slice []string, element string) bool {
	for _, elem := range slice {
		if elem == element {
			return true
		}
	}
	return false
}

// snake string, XxYy to xx_yy
func snakeString(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '_')
		}
		if d != '_' {
			j = true
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:len(data)]))
}

func camelString(s string) string {
	data := make([]byte, 0, len(s))
	j := false
	k := false
	num := len(s) - 1
	for i := 0; i <= num; i++ {
		d := s[i]
		if k == false && d >= 'A' && d <= 'Z' {
			k = true
		}
		if d >= 'a' && d <= 'z' && (j || k == false) {
			d = d - 32
			j = false
			k = true
		}
		if k && d == '_' && num > i && s[i+1] >= 'a' && s[i+1] <= 'z' {
			j = true
			continue
		}
		data = append(data, d)
	}
	return string(data[:len(data)])
}

// The string flag list, implemented flag.Value interface
type strFlags []string

func (s *strFlags) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *strFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// CloseFile attempts to close the passed file
// or panics with the actual error
func CloseFile(f *os.File) {
	if err := f.Close(); err != nil {
		panic(err)
	}
}

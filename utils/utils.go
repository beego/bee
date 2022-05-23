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

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/beego/bee/v2/config"
	"github.com/beego/bee/v2/internal/pkg/system"
	beeLogger "github.com/beego/bee/v2/logger"
	"github.com/beego/bee/v2/logger/colors"
)

type tagName struct {
	Name string `json:"name"`
}

type Repos struct {
	UpdatedAt time.Time `json:"updated_at"`
	PushedAt  time.Time `json:"pushed_at"`
}

type Releases struct {
	PublishedAt time.Time `json:"published_at"`
	TagName     string    `json:"tag_name"`
}

func GetBeeWorkPath() string {
	curpath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return curpath
}

// Go is a basic promise implementation: it wraps calls a function in a goroutine
// and returns a channel which will later return the function's return value.
func Go(f func() error) chan error {
	ch := make(chan error)
	go func() {
		ch <- f()
	}()
	return ch
}

// IsExist returns whether a file or directory exists.
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// GetGOPATHs returns all paths in GOPATH variable.
func GetGOPATHs() []string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" && strings.Compare(runtime.Version(), "go1.8") >= 0 {
		gopath = defaultGOPATH()
	}
	return filepath.SplitList(gopath)
}

// IsInGOPATH checks whether the path is inside of any GOPATH or not
func IsInGOPATH(thePath string) bool {
	for _, gopath := range GetGOPATHs() {
		if strings.Contains(thePath, filepath.Join(gopath, "src")) {
			return true
		}
	}
	return false
}

// IsBeegoProject checks whether the current path is a Beego application or not
func IsBeegoProject(thePath string) bool {
	mainFiles := []string{}
	hasBeegoRegex := regexp.MustCompile(`(?s)package main.*?import.*?\(.*?github.com/beego/beego/v2".*?\).*func main()`)
	c := make(chan error)
	// Walk the application path tree to look for main files.
	// Main files must satisfy the 'hasBeegoRegex' regular expression.
	go func() {
		filepath.Walk(thePath, func(fpath string, f os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			// Skip sub-directories
			if !f.IsDir() {
				var data []byte
				data, err = ioutil.ReadFile(fpath)
				if err != nil {
					c <- err
					return nil
				}

				if len(hasBeegoRegex.Find(data)) > 0 {
					mainFiles = append(mainFiles, fpath)
				}
			}
			return nil
		})
		close(c)
	}()

	if err := <-c; err != nil {
		beeLogger.Log.Fatalf("Unable to walk '%s' tree: %s", thePath, err)
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
		beeLogger.Log.Fatal("GOPATH environment variable is not set or empty")
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

		if IsExist(currentPath) {
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
func AskForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		beeLogger.Log.Fatalf("%s", err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return AskForConfirmation()
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
func SnakeString(s string) string {
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
	return strings.ToLower(string(data[:]))
}

func CamelString(s string) string {
	data := make([]byte, 0, len(s))
	j := false
	k := false
	num := len(s) - 1
	for i := 0; i <= num; i++ {
		d := s[i]
		if !k && d >= 'A' && d <= 'Z' {
			k = true
		}
		if d >= 'a' && d <= 'z' && (j || !k) {
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
	return string(data[:])
}

// camelCase converts a _ delimited string to camel case
// e.g. very_important_person => VeryImportantPerson
func CamelCase(in string) string {
	tokens := strings.Split(in, "_")
	for i := range tokens {
		tokens[i] = strings.Title(strings.Trim(tokens[i], " "))
	}
	return strings.Join(tokens, "")
}

// formatSourceCode formats source files
func FormatSourceCode(filename string) {
	cmd := exec.Command("gofmt", "-w", filename)
	if err := cmd.Run(); err != nil {
		beeLogger.Log.Warnf("Error while running gofmt: %s", err)
	}
}

// CloseFile attempts to close the passed file
// or panics with the actual error
func CloseFile(f *os.File) {
	err := f.Close()
	MustCheck(err)
}

// MustCheck panics when the error is not nil
func MustCheck(err error) {
	if err != nil {
		panic(err)
	}
}

// WriteToFile creates a file and writes content to it
func WriteToFile(filename, content string) {
	f, err := os.Create(filename)
	MustCheck(err)
	defer CloseFile(f)
	_, err = f.WriteString(content)
	MustCheck(err)
}

// __FILE__ returns the file name in which the function was invoked
func FILE() string {
	_, file, _, _ := runtime.Caller(1)
	return file
}

// __LINE__ returns the line number at which the function was invoked
func LINE() int {
	_, _, line, _ := runtime.Caller(1)
	return line
}

// BeeFuncMap returns a FuncMap of functions used in different templates.
func BeeFuncMap() template.FuncMap {
	return template.FuncMap{
		"trim":       strings.TrimSpace,
		"bold":       colors.Bold,
		"headline":   colors.MagentaBold,
		"foldername": colors.RedBold,
		"endline":    EndLine,
		"tmpltostr":  TmplToString,
	}
}

// TmplToString parses a text template and return the result as a string.
func TmplToString(tmpl string, data interface{}) string {
	t := template.New("tmpl").Funcs(BeeFuncMap())
	template.Must(t.Parse(tmpl))

	var doc bytes.Buffer
	err := t.Execute(&doc, data)
	MustCheck(err)

	return doc.String()
}

// EndLine returns the a newline escape character
func EndLine() string {
	return "\n"
}

func Tmpl(text string, data interface{}) {
	output := colors.NewColorWriter(os.Stderr)

	t := template.New("Usage").Funcs(BeeFuncMap())
	template.Must(t.Parse(text))

	err := t.Execute(output, data)
	if err != nil {
		beeLogger.Log.Error(err.Error())
	}
}

func CheckEnv(appname string) (apppath, packpath string, err error) {
	gps := GetGOPATHs()
	if len(gps) == 0 {
		beeLogger.Log.Error("if you want new a go module project,please add param `-gopath=false`.")
		beeLogger.Log.Fatal("GOPATH environment variable is not set or empty")
	}
	currpath, _ := os.Getwd()
	currpath = filepath.Join(currpath, appname)
	for _, gpath := range gps {
		gsrcpath := filepath.Join(gpath, "src")
		if strings.HasPrefix(strings.ToLower(currpath), strings.ToLower(gsrcpath)) {
			packpath = strings.Replace(currpath[len(gsrcpath)+1:], string(filepath.Separator), "/", -1)
			return currpath, packpath, nil
		}
	}

	// In case of multiple paths in the GOPATH, by default
	// we use the first path
	gopath := gps[0]

	beeLogger.Log.Warn("You current workdir is not inside $GOPATH/src.")
	beeLogger.Log.Debugf("GOPATH: %s", FILE(), LINE(), gopath)

	gosrcpath := filepath.Join(gopath, "src")
	apppath = filepath.Join(gosrcpath, appname)

	if _, e := os.Stat(apppath); !os.IsNotExist(e) {
		err = fmt.Errorf("cannot create application without removing '%s' first", apppath)
		beeLogger.Log.Errorf("Path '%s' already exists", apppath)
		return
	}
	packpath = strings.Join(strings.Split(apppath[len(gosrcpath)+1:], string(filepath.Separator)), "/")
	return
}

func PrintErrorAndExit(message, errorTemplate string) {
	Tmpl(fmt.Sprintf(errorTemplate, message), nil)
	os.Exit(2)
}

// GoCommand executes the passed command using Go tool
func GoCommand(command string, args ...string) error {
	allargs := []string{command}
	allargs = append(allargs, args...)
	goBuild := exec.Command("go", allargs...)
	goBuild.Stderr = os.Stderr
	return goBuild.Run()
}

// SplitQuotedFields is like strings.Fields but ignores spaces
// inside areas surrounded by single quotes.
// To specify a single quote use backslash to escape it: '\''
func SplitQuotedFields(in string) []string {
	type stateEnum int
	const (
		inSpace stateEnum = iota
		inField
		inQuote
		inQuoteEscaped
	)
	state := inSpace
	r := []string{}
	var buf bytes.Buffer

	for _, ch := range in {
		switch state {
		case inSpace:
			if ch == '\'' {
				state = inQuote
			} else if !unicode.IsSpace(ch) {
				buf.WriteRune(ch)
				state = inField
			}

		case inField:
			if ch == '\'' {
				state = inQuote
			} else if unicode.IsSpace(ch) {
				r = append(r, buf.String())
				buf.Reset()
			} else {
				buf.WriteRune(ch)
			}

		case inQuote:
			if ch == '\'' {
				state = inField
			} else if ch == '\\' {
				state = inQuoteEscaped
			} else {
				buf.WriteRune(ch)
			}

		case inQuoteEscaped:
			buf.WriteRune(ch)
			state = inQuote
		}
	}

	if buf.Len() != 0 {
		r = append(r, buf.String())
	}

	return r
}

// GetFileModTime returns unix timestamp of `os.File.ModTime` for the given path.
func GetFileModTime(path string) int64 {
	path = strings.Replace(path, "\\", "/", -1)
	f, err := os.Open(path)
	if err != nil {
		beeLogger.Log.Errorf("Failed to open file on '%s': %s", path, err)
		return time.Now().Unix()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		beeLogger.Log.Errorf("Failed to get file stats: %s", err)
		return time.Now().Unix()
	}

	return fi.ModTime().Unix()
}

func defaultGOPATH() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
		env = "home"
	}
	if home := os.Getenv(env); home != "" {
		return filepath.Join(home, "go")
	}
	return ""
}

func GetGoVersionSkipMinor() string {
	strArray := strings.Split(runtime.Version()[2:], `.`)
	return strArray[0] + `.` + strArray[1]
}

func IsGOMODULE() bool {
	if combinedOutput, e := exec.Command(`go`, `env`).CombinedOutput(); e != nil {
		beeLogger.Log.Errorf("i cann't find go.")
	} else {
		regex := regexp.MustCompile(`GOMOD="?(.+go.mod)"?`)
		stringSubmatch := regex.FindStringSubmatch(string(combinedOutput))
		return len(stringSubmatch) == 2
	}
	return false
}

func NoticeUpdateBee() {
	cmd := exec.Command("go", "version")
	cmd.Output()
	if cmd.Process == nil || cmd.Process.Pid <= 0 {
		beeLogger.Log.Warn("There is no go environment")
		return
	}
	beeHome := system.BeegoHome
	if !IsExist(beeHome) {
		if err := os.MkdirAll(beeHome, 0755); err != nil {
			beeLogger.Log.Fatalf("Could not create the directory: %s", err)
			return
		}
	}
	fp := beeHome + "/.noticeUpdateBee"
	timeNow := time.Now().Unix()
	var timeOld int64
	if !IsExist(fp) {
		f, err := os.Create(fp)
		if err != nil {
			beeLogger.Log.Warnf("Create noticeUpdateBee file err: %s", err)
			return
		}
		defer f.Close()
	}
	oldContent, err := ioutil.ReadFile(fp)
	if err != nil {
		beeLogger.Log.Warnf("Read noticeUpdateBee file err: %s", err)
		return
	}
	timeOld, _ = strconv.ParseInt(string(oldContent), 10, 64)
	if timeNow-timeOld < 24*60*60 {
		return
	}
	w, err := os.OpenFile(fp, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		beeLogger.Log.Warnf("Open noticeUpdateBee file err: %s", err)
		return
	}
	defer w.Close()
	timeNowStr := strconv.FormatInt(timeNow, 10)
	if _, err := w.WriteString(timeNowStr); err != nil {
		beeLogger.Log.Warnf("Update noticeUpdateBee file err: %s", err)
		return
	}
	beeLogger.Log.Info("Getting bee latest version...")
	versionLast := BeeLastVersion()
	versionNow := config.Version
	if versionLast == "" {
		beeLogger.Log.Warn("Get latest version err")
		return
	}
	if versionNow != versionLast {
		beeLogger.Log.Warnf("Update available %s ==> %s", versionNow, versionLast)
		beeLogger.Log.Warn("Run `bee update` to update")
	}
	beeLogger.Log.Info("Your bee are up to date")
}

func BeeLastVersion() (version string) {
	var url = "https://api.github.com/repos/beego/bee/tags"
	resp, err := http.Get(url)
	if err != nil {
		beeLogger.Log.Warnf("Get bee tags from github error: %s", err)
		return
	}
	defer resp.Body.Close()
	bodyContent, _ := ioutil.ReadAll(resp.Body)
	var tags []tagName
	if err = json.Unmarshal(bodyContent, &tags); err != nil {
		beeLogger.Log.Warnf("Unmarshal tags body error: %s", err)
		return
	}
	if len(tags) < 1 {
		beeLogger.Log.Warn("There is no tags！")
		return
	}
	last := tags[0]
	re, _ := regexp.Compile(`[0-9.]+`)
	versionList := re.FindStringSubmatch(last.Name)
	if len(versionList) > 0 {
		return versionList[0]
	}
	beeLogger.Log.Warn("There is no tags！")
	return
}

// get info of bee repos
func BeeReposInfo() (repos Repos) {
	var url = "https://api.github.com/repos/beego/bee"
	resp, err := http.Get(url)
	if err != nil {
		beeLogger.Log.Warnf("Get bee repos from github error: %s", err)
		return
	}
	defer resp.Body.Close()
	bodyContent, _ := ioutil.ReadAll(resp.Body)
	if err = json.Unmarshal(bodyContent, &repos); err != nil {
		beeLogger.Log.Warnf("Unmarshal repos body error: %s", err)
		return
	}
	return
}

// get info of bee releases
func BeeReleasesInfo() (repos []Releases) {
	var url = "https://api.github.com/repos/beego/bee/releases"
	resp, err := http.Get(url)
	if err != nil {
		beeLogger.Log.Warnf("Get bee releases from github error: %s", err)
		return
	}
	defer resp.Body.Close()
	bodyContent, _ := ioutil.ReadAll(resp.Body)
	if err = json.Unmarshal(bodyContent, &repos); err != nil {
		beeLogger.Log.Warnf("Unmarshal releases body error: %s", err)
		return
	}
	return
}

//TODO merge UpdateLastPublishedTime and NoticeUpdateBee
func UpdateLastPublishedTime() {
	info := BeeReleasesInfo()
	if len(info) == 0 {
		beeLogger.Log.Warn("Has no releases")
		return
	}
	createdAt := info[0].PublishedAt.Format("2006-01-02")
	beeHome := system.BeegoHome
	if !IsExist(beeHome) {
		if err := os.MkdirAll(beeHome, 0755); err != nil {
			beeLogger.Log.Fatalf("Could not create the directory: %s", err)
			return
		}
	}
	fp := beeHome + "/.lastPublishedAt"
	w, err := os.OpenFile(fp, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		beeLogger.Log.Warnf("Open .lastPublishedAt file err: %s", err)
		return
	}
	defer w.Close()
	if _, err := w.WriteString(createdAt); err != nil {
		beeLogger.Log.Warnf("Update .lastPublishedAt file err: %s", err)
		return
	}
}

func GetLastPublishedTime() string {
	fp := system.BeegoHome + "/.lastPublishedAt"
	if !IsExist(fp) {
		UpdateLastPublishedTime()
	}
	w, err := os.OpenFile(fp, os.O_RDONLY, 0644)
	if err != nil {
		beeLogger.Log.Warnf("Open .lastPublishedAt file err: %s", err)
		return "unknown"
	}
	t := make([]byte, 1024)
	read, err := w.Read(t)
	if err != nil {
		beeLogger.Log.Warnf("read .lastPublishedAt file err: %s", err)
		return "unknown"
	}
	return string(t[:read])
}

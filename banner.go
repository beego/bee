package main

import (
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"text/template"
	"time"
)

type vars struct {
	GoVersion    string
	GOOS         string
	GOARCH       string
	NumCPU       int
	GOPATH       string
	GOROOT       string
	Compiler     string
	BeeVersion   string
	BeegoVersion string
}

// Now returns the current local time in the specified layout
func Now(layout string) string {
	return time.Now().Format(layout)
}

// InitBanner loads the banner and prints it to output
// All errors are ignored, the application will not
// print the banner in case of error.
func InitBanner(out io.Writer, in io.Reader) {
	if in == nil {
		ColorLog("[ERRO] The input is nil\n")
		os.Exit(2)
	}

	banner, err := ioutil.ReadAll(in)
	if err != nil {
		ColorLog("[ERRO] Error trying to read the banner\n")
		ColorLog("[HINT] %v\n", err)
		os.Exit(2)
	}

	show(out, string(banner))
}

func show(out io.Writer, content string) {
	t, err := template.New("banner").
		Funcs(template.FuncMap{"Now": Now}).
		Parse(content)

	if err != nil {
		ColorLog("[ERRO] Cannot parse the banner template\n")
		ColorLog("[HINT] %v\n", err)
		os.Exit(2)
	}

	err = t.Execute(out, vars{
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumCPU(),
		os.Getenv("GOPATH"),
		runtime.GOROOT(),
		runtime.Compiler,
		version,
		getBeegoVersion(),
	})
	if err != nil {
		panic(err)
	}
}

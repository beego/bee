package main

import (
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"text/template"
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

// InitBanner loads the banner and prints it to output
// All errors are ignored, the application will not
// print the banner in case of error.
func InitBanner(out io.Writer, in io.Reader) {
	if in == nil {
		logger.Fatal("The input is nil")
	}

	banner, err := ioutil.ReadAll(in)
	if err != nil {
		logger.Fatalf("Error while trying to read the banner: %s", err)
	}

	show(out, string(banner))
}

func show(out io.Writer, content string) {
	t, err := template.New("banner").
		Funcs(template.FuncMap{"Now": Now}).
		Parse(content)

	if err != nil {
		logger.Fatalf("Cannot parse the banner template: %s", err)
	}

	err = t.Execute(out, vars{
		getGoVersion(),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumCPU(),
		os.Getenv("GOPATH"),
		runtime.GOROOT(),
		runtime.Compiler,
		version,
		getBeegoVersion(),
	})
	MustCheck(err)
}

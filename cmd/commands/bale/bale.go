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
package bale

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	"github.com/beego/bee/config"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
)

var CmdBale = &commands.Command{
	UsageLine: "bale",
	Short:     "Transforms non-Go files to Go source files",
	Long: `Bale command compress all the static files in to a single binary file.

  This is useful to not have to carry static files including js, css, images and
  views when deploying a Web application.

  It will auto-generate an unpack function to the main package then run it during the runtime.
  This is mainly used for zealots who are requiring 100% Go code.
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    runBale,
}

func init() {
	commands.AvailableCommands = append(commands.AvailableCommands, CmdBale)
}

func runBale(cmd *commands.Command, args []string) int {
	os.RemoveAll("bale")
	os.Mkdir("bale", os.ModePerm)

	// Pack and compress data
	for _, p := range config.Conf.Bale.Dirs {
		if !utils.IsExist(p) {
			beeLogger.Log.Warnf("Skipped directory: %s", p)
			continue
		}
		beeLogger.Log.Infof("Packaging directory: %s", p)
		filepath.Walk(p, walkFn)
	}

	// Generate auto-uncompress function.
	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf(BaleHeader, config.Conf.Bale.Import,
		strings.Join(resFiles, "\",\n\t\t\""),
		strings.Join(resFiles, ",\n\t\tbale.R")))

	fw, err := os.Create("bale.go")
	if err != nil {
		beeLogger.Log.Fatalf("Failed to create file: %s", err)
	}
	defer fw.Close()

	_, err = fw.Write(buf.Bytes())
	if err != nil {
		beeLogger.Log.Fatalf("Failed to write data: %s", err)
	}

	beeLogger.Log.Success("Baled resources successfully!")
	return 0
}

const (
	// BaleHeader ...
	BaleHeader = `package main

import(
	"os"
	"strings"
	"path"

	"%s"
)

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func init() {
	files := []string{
		"%s",
	}

	funcs := []func() []byte{
		bale.R%s,
	}

	for i, f := range funcs {
		fp := getFilePath(files[i])
		if !isExist(fp) {
			saveFile(fp, f())
		}
	}
}

func getFilePath(name string) string {
	name = strings.Replace(name, "_4_", "/", -1)
	name = strings.Replace(name, "_3_", " ", -1)
	name = strings.Replace(name, "_2_", "-", -1)
	name = strings.Replace(name, "_1_", ".", -1)
	name = strings.Replace(name, "_0_", "_", -1)
	return name
}

func saveFile(filePath string, b []byte) (int, error) {
	os.MkdirAll(path.Dir(filePath), os.ModePerm)
	fw, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer fw.Close()
	return fw.Write(b)
}
`
)

var resFiles = make([]string, 0, 10)

func walkFn(resPath string, info os.FileInfo, _ error) error {
	if info.IsDir() || filterSuffix(resPath) {
		return nil
	}

	// Open resource files
	fr, err := os.Open(resPath)
	if err != nil {
		beeLogger.Log.Fatalf("Failed to read file: %s", err)
	}

	// Convert path
	resPath = strings.Replace(resPath, "_", "_0_", -1)
	resPath = strings.Replace(resPath, ".", "_1_", -1)
	resPath = strings.Replace(resPath, "-", "_2_", -1)
	resPath = strings.Replace(resPath, " ", "_3_", -1)
	sep := "/"
	if runtime.GOOS == "windows" {
		sep = "\\"
	}
	resPath = strings.Replace(resPath, sep, "_4_", -1)

	// Create corresponding Go source files
	os.MkdirAll(path.Dir(resPath), os.ModePerm)
	fw, err := os.Create("bale/" + resPath + ".go")
	if err != nil {
		beeLogger.Log.Fatalf("Failed to create file: %s", err)
	}
	defer fw.Close()

	// Write header
	fmt.Fprintf(fw, Header, resPath)

	// Copy and compress data
	gz := gzip.NewWriter(&ByteWriter{Writer: fw})
	io.Copy(gz, fr)
	gz.Close()

	// Write footer.
	fmt.Fprint(fw, Footer)

	resFiles = append(resFiles, resPath)
	return nil
}

func filterSuffix(name string) bool {
	for _, s := range config.Conf.Bale.IngExt {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

const (
	// Header ...
	Header = `package bale

import(
	"bytes"
	"compress/gzip"
	"io"
)

func R%s() []byte {
	gz, err := gzip.NewReader(bytes.NewBuffer([]byte{`
	// Footer ...
	Footer = `
	}))

	if err != nil {
		panic("Unpack resources failed: " + err.Error())
	}

	var b bytes.Buffer
	io.Copy(&b, gz)
	gz.Close()

	return b.Bytes()
}`
)

var newline = []byte{'\n'}

// ByteWriter ...
type ByteWriter struct {
	io.Writer
	c int
}

func (w *ByteWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	for n = range p {
		if w.c%12 == 0 {
			w.Writer.Write(newline)
			w.c = 0
		}
		fmt.Fprintf(w.Writer, "0x%02x,", p[n])
		w.c++
	}
	n++
	return
}

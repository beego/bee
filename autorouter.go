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
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	gobuild "go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

var cmdRouter = &Command{
	UsageLine: "router",
	Short:     "auto-generate routers for the app controllers",
	Long: `
  
`,
}

func init() {
	cmdRouter.Run = autoRouter
}

func autoRouter(cmd *Command, args []string) int {
	fmt.Println("[INFO] Starting auto-generating routers...")
	return 0
}

// getControllerInfo returns controllers that embeded "beego.controller"
// and their methods of package in given path.
func getControllerInfo(path string) (map[string][]string, error) {
	now := time.Now()
	path = strings.TrimSuffix(path, "/")
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	fis, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}

	files := make([]*source, 0, len(fis))
	for _, fi := range fis {
		// Only load go files.
		if strings.HasSuffix(fi.Name(), ".go") {
			f, err := os.Open(path + "/" + fi.Name())
			if err != nil {
				return nil, err
			}

			p := make([]byte, fi.Size())
			_, err = f.Read(p)
			if err != nil {
				return nil, err
			}

			f.Close()
			files = append(files, &source{
				name: path + "/" + fi.Name(),
				data: p,
			})
		}
	}

	rw := &routerWalker{
		pdoc: &Package{
			ImportPath: path,
		},
	}

	cm := make(map[string][]string)
	pdoc, err := rw.build(files)
	for _, t := range pdoc.Types {
		// Check if embeded "beego.Controller".
		if strings.Index(t.Decl, "beego.Controller") > -1 {
			for _, f := range t.Methods {
				cm[t.Name] = append(cm[t.Name], f.Name)
			}
		}
	}
	fmt.Println(time.Since(now))
	return cm, nil
}

// A source describles a source code file.
type source struct {
	name string
	data []byte
}

func (s *source) Name() string       { return s.name }
func (s *source) Size() int64        { return int64(len(s.data)) }
func (s *source) Mode() os.FileMode  { return 0 }
func (s *source) ModTime() time.Time { return time.Time{} }
func (s *source) IsDir() bool        { return false }
func (s *source) Sys() interface{}   { return nil }

// A routerWalker holds the state used when building the documentation.
type routerWalker struct {
	pdoc *Package
	srcs map[string]*source // Source files.
	fset *token.FileSet
	buf  []byte // scratch space for printNode method.
}

// Package represents full information and documentation for a package.
type Package struct {
	ImportPath string

	// Top-level declarations.
	Types []*Type
}

// Type represents structs and interfaces.
type Type struct {
	Name    string // Type name.
	Decl    string
	Methods []*Func // Exported methods.
}

// Func represents functions
type Func struct {
	Name string
}

// build generates data from source files.
func (w *routerWalker) build(srcs []*source) (*Package, error) {
	// Add source files to walker, I skipped references here.
	w.srcs = make(map[string]*source)
	for _, src := range srcs {
		w.srcs[src.name] = src
	}

	w.fset = token.NewFileSet()

	// Find the package and associated files.
	ctxt := gobuild.Context{
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		CgoEnabled:    true,
		JoinPath:      path.Join,
		IsAbsPath:     path.IsAbs,
		SplitPathList: func(list string) []string { return strings.Split(list, ":") },
		IsDir:         func(path string) bool { panic("unexpected") },
		HasSubdir:     func(root, dir string) (rel string, ok bool) { panic("unexpected") },
		ReadDir:       func(dir string) (fi []os.FileInfo, err error) { return w.readDir(dir) },
		OpenFile:      func(path string) (r io.ReadCloser, err error) { return w.openFile(path) },
		Compiler:      "gc",
	}

	bpkg, err := ctxt.ImportDir(w.pdoc.ImportPath, 0)
	// Continue if there are no Go source files; we still want the directory info.
	_, nogo := err.(*gobuild.NoGoError)
	if err != nil {
		if nogo {
			err = nil
		} else {
			return nil, errors.New("routerWalker.build -> " + err.Error())
		}
	}

	// Parse the Go files
	files := make(map[string]*ast.File)
	for _, name := range append(bpkg.GoFiles, bpkg.CgoFiles...) {
		file, err := parser.ParseFile(w.fset, name, w.srcs[name].data, parser.ParseComments)
		if err != nil {
			return nil, errors.New("routerWalker.build -> parse go files: " + err.Error())
		}
		files[name] = file
	}

	apkg, _ := ast.NewPackage(w.fset, files, simpleImporter, nil)

	mode := doc.Mode(0)
	if w.pdoc.ImportPath == "builtin" {
		mode |= doc.AllDecls
	}

	pdoc := doc.New(apkg, w.pdoc.ImportPath, mode)

	w.pdoc.Types = w.types(pdoc.Types)

	return w.pdoc, err
}

func (w *routerWalker) funcs(fdocs []*doc.Func) []*Func {
	var result []*Func
	for _, d := range fdocs {
		result = append(result, &Func{
			Name: d.Name,
		})
	}
	return result
}

func (w *routerWalker) types(tdocs []*doc.Type) []*Type {
	var result []*Type
	for _, d := range tdocs {
		result = append(result, &Type{
			Decl:    w.printDecl(d.Decl),
			Name:    d.Name,
			Methods: w.funcs(d.Methods),
		})
	}
	return result
}

func (w *routerWalker) printDecl(decl ast.Node) string {
	var d Code
	d, w.buf = printDecl(decl, w.fset, w.buf)
	return d.Text
}

func (w *routerWalker) readDir(dir string) ([]os.FileInfo, error) {
	if dir != w.pdoc.ImportPath {
		panic("unexpected")
	}
	fis := make([]os.FileInfo, 0, len(w.srcs))
	for _, src := range w.srcs {
		fis = append(fis, src)
	}
	return fis, nil
}

func (w *routerWalker) openFile(path string) (io.ReadCloser, error) {
	if strings.HasPrefix(path, w.pdoc.ImportPath+"/") {
		if src, ok := w.srcs[path[len(w.pdoc.ImportPath)+1:]]; ok {
			return ioutil.NopCloser(bytes.NewReader(src.data)), nil
		}
	}
	panic("unexpected")
}

func simpleImporter(imports map[string]*ast.Object, path string) (*ast.Object, error) {
	pkg := imports[path]
	if pkg == nil {
		// Guess the package name without importing it. Start with the last
		// element of the path.
		name := path[strings.LastIndex(path, "/")+1:]

		// Trim commonly used prefixes and suffixes containing illegal name
		// runes.
		name = strings.TrimSuffix(name, ".go")
		name = strings.TrimSuffix(name, "-go")
		name = strings.TrimPrefix(name, "go.")
		name = strings.TrimPrefix(name, "go-")
		name = strings.TrimPrefix(name, "biogo.")

		// It's also common for the last element of the path to contain an
		// extra "go" prefix, but not always. TODO: examine unresolved ids to
		// detect when trimming the "go" prefix is appropriate.

		pkg = ast.NewObj(ast.Pkg, name)
		pkg.Data = ast.NewScope(nil)
		imports[path] = pkg
	}
	return pkg, nil
}

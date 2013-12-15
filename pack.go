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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	path "path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var cmdPack = &Command{
	CustomFlags: true,
	UsageLine:   "pack",
	Short:       "compress an beego project",
	Long: `
compress an beego project

-p        app path. default is current path
-b        build specify platform app. default true
-o        compressed file output dir. default use current path
-f        format. [ tar.gz / zip ]. default tar.gz. note: zip doesn't support embed symlink, skip it
-exp      path exclude prefix. default: .
-exs      path exclude suffix. default: .go:.DS_Store:.tmp
          all path use : as separator
-fs       follow symlink. default false
-ss       skip symlink. default false
          default embed symlink into compressed file
-v        verbose
`,
}

var (
	appPath  string
	excludeP string
	excludeS string
	outputP  string
	fsym     bool
	ssym     bool
	build    bool
	verbose  bool
	format   string
)

func init() {
	fs := flag.NewFlagSet("pack", flag.ContinueOnError)
	fs.StringVar(&appPath, "p", "", "")
	fs.StringVar(&excludeP, "exp", ".", "")
	fs.StringVar(&excludeS, "exs", ".go:.DS_Store:.tmp", "")
	fs.StringVar(&outputP, "o", "", "")
	fs.BoolVar(&build, "b", true, "")
	fs.BoolVar(&fsym, "fs", false, "")
	fs.BoolVar(&ssym, "ss", false, "")
	fs.BoolVar(&verbose, "v", false, "")
	fs.StringVar(&format, "f", "tar.gz", "")
	cmdPack.Flag = *fs
	cmdPack.Run = packApp
}

func exitPrint(con string) {
	fmt.Fprintln(os.Stderr, con)
	os.Exit(2)
}

type walker interface {
	isExclude(string) bool
	isEmpty(string) bool
	relName(string) string
	virPath(string) string
	compress(string, string, os.FileInfo) (bool, error)
	walkRoot(string) error
}

type byName []os.FileInfo

func (f byName) Len() int           { return len(f) }
func (f byName) Less(i, j int) bool { return f[i].Name() < f[j].Name() }
func (f byName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }

type walkFileTree struct {
	wak           walker
	prefix        string
	excludePrefix []string
	excludeSuffix []string
	allfiles      map[string]bool
}

func (wft *walkFileTree) setPrefix(prefix string) {
	wft.prefix = prefix
}

func (wft *walkFileTree) isExclude(name string) bool {
	if name == "" {
		return true
	}
	for _, prefix := range wft.excludePrefix {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	for _, suffix := range wft.excludeSuffix {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func (wft *walkFileTree) isEmpty(fpath string) bool {
	fh, _ := os.Open(fpath)
	defer fh.Close()
	infos, _ := fh.Readdir(-1)
	for _, fi := range infos {
		fp := path.Join(fpath, fi.Name())
		if wft.isExclude(wft.virPath(fp)) {
			continue
		}
		if fi.Mode()&os.ModeSymlink > 0 {
			continue
		}
		if fi.IsDir() && wft.isEmpty(fp) {
			continue
		}
		return false
	}
	return true
}

func (wft *walkFileTree) relName(fpath string) string {
	name, _ := path.Rel(wft.prefix, fpath)
	return name
}

func (wft *walkFileTree) virPath(fpath string) string {
	name := fpath[len(wft.prefix):]
	if name == "" {
		return ""
	}
	name = name[1:]
	return name
}

func (wft *walkFileTree) readDir(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Sort(byName(list))
	return list, nil
}

func (wft *walkFileTree) walkLeaf(fpath string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if fpath == outputP {
		return nil
	}

	if fi.IsDir() {
		return nil
	}

	name := wft.virPath(fpath)
	if wft.isExclude(name) {
		return nil
	}

	if ssym && fi.Mode()&os.ModeSymlink > 0 {
		return nil
	}

	if wft.allfiles[name] {
		return nil
	}

	if added, err := wft.wak.compress(name, fpath, fi); added {
		if verbose {
			fmt.Printf("Compressed: %s\n", name)
		}
		wft.allfiles[name] = true
		return err
	} else {
		return err
	}
}

func (wft *walkFileTree) iterDirectory(fpath string, fi os.FileInfo) error {
	doFSym := fsym && fi.Mode()&os.ModeSymlink > 0
	if doFSym {
		nfi, err := os.Stat(fpath)
		if os.IsNotExist(err) {
			return nil
		}
		fi = nfi
	}

	err := wft.walkLeaf(fpath, fi, nil)
	if err != nil {
		if fi.IsDir() && err == path.SkipDir {
			return nil
		}
		return err
	}

	if !fi.IsDir() {
		return nil
	}

	list, err := wft.readDir(fpath)
	if err != nil {
		return wft.walkLeaf(fpath, fi, err)
	}

	for _, fileInfo := range list {
		err = wft.iterDirectory(path.Join(fpath, fileInfo.Name()), fileInfo)
		if err != nil {
			if !fileInfo.IsDir() || err != path.SkipDir {
				return err
			}
		}
	}
	return nil
}

func (wft *walkFileTree) walkRoot(root string) error {
	wft.prefix = root
	fi, err := os.Stat(root)
	if err != nil {
		return err
	}
	return wft.iterDirectory(root, fi)
}

type tarWalk struct {
	walkFileTree
	tw *tar.Writer
}

func (wft *tarWalk) compress(name, fpath string, fi os.FileInfo) (bool, error) {
	isSym := fi.Mode()&os.ModeSymlink > 0
	link := ""
	if isSym {
		link, _ = os.Readlink(fpath)
	}

	hdr, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return false, err
	}
	hdr.Name = name

	tw := wft.tw
	err = tw.WriteHeader(hdr)
	if err != nil {
		return false, err
	}

	if isSym == false {
		fr, err := os.Open(fpath)
		if err != nil {
			return false, err
		}
		defer fr.Close()
		_, err = io.Copy(tw, fr)
		if err != nil {
			return false, err
		}
		tw.Flush()
	}

	return true, nil
}

type zipWalk struct {
	walkFileTree
	zw *zip.Writer
}

func (wft *zipWalk) compress(name, fpath string, fi os.FileInfo) (bool, error) {
	isSym := fi.Mode()&os.ModeSymlink > 0
	if isSym {
		// golang1.1 doesn't support embed symlink
		// what i miss something?
		return false, nil
	}

	hdr, err := zip.FileInfoHeader(fi)
	if err != nil {
		return false, err
	}
	hdr.Name = name

	zw := wft.zw
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return false, err
	}

	if isSym == false {
		fr, err := os.Open(fpath)
		if err != nil {
			return false, err
		}
		defer fr.Close()
		_, err = io.Copy(w, fr)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func packDirectory(excludePrefix []string, excludeSuffix []string, includePath ...string) (err error) {
	fmt.Printf("exclude prefix: %s\n", strings.Join(excludePrefix, ":"))
	fmt.Printf("exclude suffix: %s\n", strings.Join(excludeSuffix, ":"))

	w, err := os.OpenFile(outputP, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	var wft walker

	if format == "zip" {
		walk := new(zipWalk)
		zw := zip.NewWriter(w)
		defer func() {
			zw.Close()
		}()
		walk.allfiles = make(map[string]bool)
		walk.zw = zw
		walk.wak = walk
		walk.excludePrefix = excludePrefix
		walk.excludeSuffix = excludeSuffix
		wft = walk
	} else {
		walk := new(tarWalk)
		cw := gzip.NewWriter(w)
		tw := tar.NewWriter(cw)

		defer func() {
			tw.Flush()
			cw.Flush()
			tw.Close()
			cw.Close()
		}()
		walk.allfiles = make(map[string]bool)
		walk.tw = tw
		walk.wak = walk
		walk.excludePrefix = excludePrefix
		walk.excludeSuffix = excludeSuffix
		wft = walk
	}

	for _, p := range includePath {
		err = wft.walkRoot(p)
		if err != nil {
			return
		}
	}

	return
}

func isBeegoProject(thePath string) bool {
	fh, _ := os.Open(thePath)
	fis, _ := fh.Readdir(-1)
	regex := regexp.MustCompile(`(?s)package main.*?import.*?\(.*?"github.com/astaxie/beego".*?\).*func main()`)
	for _, fi := range fis {
		if fi.IsDir() == false && strings.HasSuffix(fi.Name(), ".go") {
			data, err := ioutil.ReadFile(path.Join(thePath, fi.Name()))
			if err != nil {
				continue
			}
			if len(regex.Find(data)) > 0 {
				return true
			}
		}
	}
	return false
}

func packApp(cmd *Command, args []string) {
	curPath, _ := os.Getwd()
	thePath := ""

	nArgs := []string{}
	has := false
	for _, a := range args {
		if a != "" && a[0] == '-' {
			has = true
		}
		if has {
			nArgs = append(nArgs, a)
		}
	}
	cmdPack.Flag.Parse(nArgs)

	if path.IsAbs(appPath) == false {
		appPath = path.Join(curPath, appPath)
	}

	thePath, err := path.Abs(appPath)
	if err != nil {
		exitPrint(fmt.Sprintf("wrong app path: %s", thePath))
	}
	if stat, err := os.Stat(thePath); os.IsNotExist(err) || stat.IsDir() == false {
		exitPrint(fmt.Sprintf("not exist app path: %s", thePath))
	}

	if isBeegoProject(thePath) == false {
		exitPrint(fmt.Sprintf("not support non beego project"))
	}

	fmt.Printf("app path: %s\n", thePath)

	appName := path.Base(thePath)

	goos := runtime.GOOS
	if v, found := syscall.Getenv("GOOS"); found {
		goos = v
	}
	goarch := runtime.GOARCH
	if v, found := syscall.Getenv("GOARCH"); found {
		goarch = v
	}

	str := strconv.FormatInt(time.Now().UnixNano(), 10)[9:]

	gobin := path.Join(runtime.GOROOT(), "bin", "go")
	tmpdir := path.Join(os.TempDir(), "beePack-"+str)

	os.Mkdir(tmpdir, 0700)

	if build {
		fmt.Println("GOOS", goos, "GOARCH", goarch)
		fmt.Println("build", appName)

		os.Setenv("GOOS", goos)
		os.Setenv("GOARCH", goarch)

		binPath := path.Join(tmpdir, appName)
		if goos == "windows" {
			binPath += ".exe"
		}

		execmd := exec.Command(gobin, "build", "-o", binPath)
		execmd.Stdout = os.Stdout
		execmd.Stderr = os.Stderr
		execmd.Dir = thePath
		err = execmd.Run()
		if err != nil {
			exitPrint(err.Error())
		}

		fmt.Println("build success")
	}

	switch format {
	case "zip":
	default:
		format = "tar.gz"
	}

	outputN := appName + "." + format

	if outputP == "" || path.IsAbs(outputP) == false {
		outputP = path.Join(curPath, outputP)
	}

	if _, err := os.Stat(outputP); err != nil {
		err = os.MkdirAll(outputP, 0755)
		if err != nil {
			exitPrint(err.Error())
		}
	}

	outputP = path.Join(outputP, outputN)

	var exp, exs []string
	for _, p := range strings.Split(excludeP, ":") {
		if len(p) > 0 {
			exp = append(exp, p)
		}
	}
	for _, p := range strings.Split(excludeS, ":") {
		if len(p) > 0 {
			exs = append(exs, p)
		}
	}

	err = packDirectory(exp, exs, tmpdir, thePath)
	if err != nil {
		exitPrint(err.Error())
	}

	fmt.Printf("file write to `%s`\n", outputP)
}

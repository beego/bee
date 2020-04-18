package pack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
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

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
)

var CmdPack = &commands.Command{
	CustomFlags: true,
	UsageLine:   "pack",
	Short:       "Compresses a Beego application into a single file",
	Long: `Pack is used to compress Beego applications into a tarball/zip file.
  This eases the deployment by directly extracting the file to a server.

  {{"Example:"|bold}}
    $ bee pack -v -ba="-ldflags '-s -w'"
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    packApp,
}

var (
	appPath   string
	excludeP  string
	excludeS  string
	outputP   string
	excludeR  utils.ListOpts
	fsym      bool
	ssym      bool
	build     bool
	buildArgs string
	buildEnvs utils.ListOpts
	verbose   bool
	format    string
)

func init() {
	fs := flag.NewFlagSet("pack", flag.ContinueOnError)
	fs.StringVar(&appPath, "p", "", "Set the application path. Defaults to the current path.")
	fs.BoolVar(&build, "b", true, "Tell the command to do a build for the current platform. Defaults to true.")
	fs.StringVar(&buildArgs, "ba", "", "Specify additional args for Go build.")
	fs.Var(&buildEnvs, "be", "Specify additional env variables for Go build. e.g. GOARCH=arm.")
	fs.StringVar(&outputP, "o", "", "Set the compressed file output path. Defaults to the current path.")
	fs.StringVar(&format, "f", "tar.gz", "Set file format. Either tar.gz or zip. Defaults to tar.gz.")
	fs.StringVar(&excludeP, "exp", ".", "Set prefixes of paths to be excluded. Uses a column (:) as separator.")
	fs.StringVar(&excludeS, "exs", ".go:.DS_Store:.tmp", "Set suffixes of paths to be excluded. Uses a column (:) as separator.")
	fs.Var(&excludeR, "exr", "Set a regular expression of files to be excluded.")
	fs.BoolVar(&fsym, "fs", false, "Tell the command to follow symlinks. Defaults to false.")
	fs.BoolVar(&ssym, "ss", false, "Tell the command to skip symlinks. Defaults to false.")
	fs.BoolVar(&verbose, "v", false, "Be more verbose during the operation. Defaults to false.")
	CmdPack.Flag = *fs
	commands.AvailableCommands = append(commands.AvailableCommands, CmdPack)
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
	excludeRegexp []*regexp.Regexp
	excludeSuffix []string
	allfiles      map[string]bool
	output        *io.Writer
}

func (wft *walkFileTree) isExclude(fPath string) bool {
	if fPath == "" {
		return true
	}

	for _, prefix := range wft.excludePrefix {
		if strings.HasPrefix(fPath, prefix) {
			return true
		}
	}
	for _, suffix := range wft.excludeSuffix {
		if strings.HasSuffix(fPath, suffix) {
			return true
		}
	}
	return false
}

func (wft *walkFileTree) isExcludeName(name string) bool {
	for _, r := range wft.excludeRegexp {
		if r.MatchString(name) {
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
		fn := fi.Name()
		fp := path.Join(fpath, fn)
		if wft.isExclude(wft.virPath(fp)) {
			continue
		}
		if wft.isExcludeName(fn) {
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
	name = path.ToSlash(name)
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

	if ssym && fi.Mode()&os.ModeSymlink > 0 {
		return nil
	}

	name := wft.virPath(fpath)

	if wft.allfiles[name] {
		return nil
	}

	if added, err := wft.wak.compress(name, fpath, fi); added {
		if verbose {
			fmt.Fprintf(*wft.output, "\t%s%scompressed%s\t %s%s\n", "\x1b[32m", "\x1b[1m", "\x1b[21m", name, "\x1b[0m")
		}
		wft.allfiles[name] = true
		return err
	}
	return err
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

	relPath := wft.virPath(fpath)

	if len(relPath) > 0 {
		if wft.isExcludeName(fi.Name()) {
			return nil
		}

		if wft.isExclude(relPath) {
			return nil
		}
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

	if !isSym {
		fr, err := os.Open(fpath)
		if err != nil {
			return false, err
		}
		defer utils.CloseFile(fr)
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

	if !isSym {
		fr, err := os.Open(fpath)
		if err != nil {
			return false, err
		}
		defer utils.CloseFile(fr)
		_, err = io.Copy(w, fr)
		if err != nil {
			return false, err
		}
	} else {
		var link string
		if link, err = os.Readlink(fpath); err != nil {
			return false, err
		}
		_, err = w.Write([]byte(link))
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func packDirectory(output io.Writer, excludePrefix []string, excludeSuffix []string,
	excludeRegexp []*regexp.Regexp, includePath ...string) (err error) {

	beeLogger.Log.Infof("Excluding relpath prefix: %s", strings.Join(excludePrefix, ":"))
	beeLogger.Log.Infof("Excluding relpath suffix: %s", strings.Join(excludeSuffix, ":"))
	if len(excludeRegexp) > 0 {
		beeLogger.Log.Infof("Excluding filename regex: `%s`", strings.Join(excludeR, "`, `"))
	}

	w, err := os.OpenFile(outputP, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	var wft walker

	if format == "zip" {
		walk := new(zipWalk)
		walk.output = &output
		zw := zip.NewWriter(w)
		defer func() {
			zw.Close()
		}()
		walk.allfiles = make(map[string]bool)
		walk.zw = zw
		walk.wak = walk
		walk.excludePrefix = excludePrefix
		walk.excludeSuffix = excludeSuffix
		walk.excludeRegexp = excludeRegexp
		wft = walk
	} else {
		walk := new(tarWalk)
		walk.output = &output
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
		walk.excludeRegexp = excludeRegexp
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

func packApp(cmd *commands.Command, args []string) int {
	output := cmd.Out()
	curPath, _ := os.Getwd()
	var thePath string

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
	cmd.Flag.Parse(nArgs)

	if !path.IsAbs(appPath) {
		appPath = path.Join(curPath, appPath)
	}

	thePath, err := path.Abs(appPath)
	if err != nil {
		beeLogger.Log.Fatalf("Wrong application path: %s", thePath)
	}
	if stat, err := os.Stat(thePath); os.IsNotExist(err) || !stat.IsDir() {
		beeLogger.Log.Fatalf("Application path does not exist: %s", thePath)
	}

	beeLogger.Log.Infof("Packaging application on '%s'...", thePath)

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

	tmpdir := path.Join(os.TempDir(), "beePack-"+str)

	os.Mkdir(tmpdir, 0700)
	defer func() {
		// Remove the tmpdir once bee pack is done
		err := os.RemoveAll(tmpdir)
		if err != nil {
			beeLogger.Log.Error("Failed to remove the generated temp dir")
		}
	}()

	if build {
		beeLogger.Log.Info("Building application...")
		var envs []string
		for _, env := range buildEnvs {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
				if len(k) > 0 && len(v) > 0 {
					switch k {
					case "GOOS":
						goos = v
					case "GOARCH":
						goarch = v
					default:
						envs = append(envs, fmt.Sprintf("%s=%s", k, v))
					}
				}
			}
		}

		os.Setenv("GOOS", goos)
		os.Setenv("GOARCH", goarch)

		beeLogger.Log.Infof("Using: GOOS=%s GOARCH=%s", goos, goarch)

		binPath := path.Join(tmpdir, appName)
		if goos == "windows" {
			binPath += ".exe"
		}

		args := []string{"build", "-o", binPath}
		if len(buildArgs) > 0 {
			args = append(args, strings.Fields(buildArgs)...)
		}

		if verbose {
			fmt.Fprintf(output, "\t%s%s+ go %s%s%s\n", "\x1b[32m", "\x1b[1m", strings.Join(args, " "), "\x1b[21m", "\x1b[0m")
		}

		execmd := exec.Command("go", args...)
		execmd.Env = append(os.Environ(), envs...)
		execmd.Stdout = os.Stdout
		execmd.Stderr = os.Stderr
		execmd.Dir = thePath
		err = execmd.Run()
		if err != nil {
			beeLogger.Log.Fatal(err.Error())
		}

		beeLogger.Log.Success("Build Successful!")
	}

	switch format {
	case "zip":
	default:
		format = "tar.gz"
	}

	outputN := appName + "." + format

	if outputP == "" || !path.IsAbs(outputP) {
		outputP = path.Join(curPath, outputP)
	}

	if _, err := os.Stat(outputP); err != nil {
		err = os.MkdirAll(outputP, 0755)
		if err != nil {
			beeLogger.Log.Fatal(err.Error())
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

	var exr []*regexp.Regexp
	for _, r := range excludeR {
		if len(r) > 0 {
			if re, err := regexp.Compile(r); err != nil {
				beeLogger.Log.Fatal(err.Error())
			} else {
				exr = append(exr, re)
			}
		}
	}

	beeLogger.Log.Infof("Writing to output: %s", outputP)

	err = packDirectory(output, exp, exs, exr, tmpdir, thePath)
	if err != nil {
		beeLogger.Log.Fatal(err.Error())
	}

	beeLogger.Log.Success("Application packed!")
	return 0
}

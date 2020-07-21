package beegopro

import (
	"fmt"
	"github.com/beego/bee/internal/pkg/command"
	"github.com/beego/bee/internal/pkg/system"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
	"github.com/flosch/pongo2"
	"github.com/smartwalle/pongo2render"
	"path/filepath"
	"strings"
	"sync"
)

// store all data
type Container struct {
	BeegoProFile     string                 // beego pro toml
	TimestampFile    string                 // store ts file
	GoModFile        string                 // go mod file
	UserOption       UserOption             // user option
	TmplOption       TmplOption             // tmpl option
	CurPath          string                 // user current path
	EnableModules    map[string]interface{} // beego pro provider a collection of module
	FunctionOnce     map[string]sync.Once   // exec function once
	Timestamp        Timestamp
	GenerateTime     string
	GenerateTimeUnix int64
	Parser           Parser
}

// user option
type UserOption struct {
	Debug          bool                 `json:"debug"`
	ContextDebug   bool                 `json:"contextDebug"`
	Dsn            string               `json:"dsn"`
	Driver         string               `json:"driver"`
	ProType        string               `json:"proType"`
	ApiPrefix      string               `json:"apiPrefix"`
	EnableModule   []string             `json:"enableModule"`
	Models         map[string]TextModel `json:"models"`
	GitRemotePath  string               `json:"gitRemotePath"`
	Branch         string               `json:"branch"`
	GitLocalPath   string               `json:"gitLocalPath"`
	EnableFormat   bool                 `json:"enableFormat"`
	SourceGen      string               `json:"sourceGen"`
	EnableGitPull  bool                 `json:"enbaleGitPull"`
	Path           map[string]string    `json:"path"`
	EnableGomod    bool                 `json:"enableGomod"`
	RefreshGitTime int64                `json:"refreshGitTime"`
	Extend         map[string]string    `json:"extend"` // extend user data
}

// tmpl option
type TmplOption struct {
	RenderPath string `toml:"renderPath"`
	Descriptor []Descriptor
}

type Descriptor struct {
	Module  string `toml:"module"`
	SrcName string `toml:"srcName"`
	DstPath string `toml:"dstPath"`
	Once    bool   `toml:"once"`
	Script  string `toml:"script"`
}

func (descriptor Descriptor) Parse(modelName string, paths map[string]string) (newDescriptor Descriptor, ctx pongo2.Context) {
	var (
		err             error
		relativeDstPath string
		absFile         string
		relPath         string
	)

	newDescriptor = descriptor
	render := pongo2render.NewRender("")
	ctx = make(pongo2.Context)
	for key, value := range paths {
		absFile, err = filepath.Abs(value)
		if err != nil {
			beeLogger.Log.Fatalf("absolute path error %s from key %s and value %s", err, key, value)
		}
		relPath, err = filepath.Rel(system.CurrentDir, absFile)
		if err != nil {
			beeLogger.Log.Fatalf("Could not get the relative path: %s", err)
		}
		// user input path
		ctx["path"+utils.CamelCase(key)] = value
		// relativePath
		ctx["pathRel"+utils.CamelCase(key)] = relPath
	}
	ctx["modelName"] = lowerFirst(utils.CamelString(modelName))
	relativeDstPath, err = render.TemplateFromString(descriptor.DstPath).Execute(ctx)
	if err != nil {
		beeLogger.Log.Fatalf("beego tmpl exec error, err: %s", err)
		return
	}

	newDescriptor.DstPath, err = filepath.Abs(relativeDstPath)
	if err != nil {
		beeLogger.Log.Fatalf("absolute path error %s from flush file %s", err, relativeDstPath)
	}

	newDescriptor.Script, err = render.TemplateFromString(descriptor.Script).Execute(ctx)
	if err != nil {
		beeLogger.Log.Fatalf("parse script %s, error %s", descriptor.Script, err)
	}
	return
}

func (descriptor Descriptor) IsExistScript() bool {
	return descriptor.Script != ""
}

func (d Descriptor) ExecScript(path string) (err error) {
	arr := strings.Split(d.Script, " ")
	if len(arr) == 0 {
		return
	}

	stdout, stderr, err := command.ExecCmdDir(path, arr[0], arr[1:]...)
	if err != nil {
		return concatenateError(err, stderr)
	}

	beeLogger.Log.Info(stdout)
	return nil
}

type Timestamp struct {
	GitCacheLastRefresh int64 `toml:"gitCacheLastRefresh"`
	Generate            int64 `toml:"generate"`
}

func concatenateError(err error, stderr string) error {
	if len(stderr) == 0 {
		return err
	}
	return fmt.Errorf("%v: %s", err, stderr)
}

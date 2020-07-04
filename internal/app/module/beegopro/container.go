package beegopro

import (
	"fmt"
	"github.com/beego/bee/internal/pkg/git"
	"github.com/beego/bee/internal/pkg/system"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
	"github.com/davecgh/go-spew/spew"
	"github.com/pelletier/go-toml"
	"github.com/spf13/viper"
	"io/ioutil"
	"sync"
	"time"
)

const MDateFormat = "20060102_150405"

var DefaultBeegoPro = &Container{
	BeegoProFile:  system.CurrentDir + "/beegopro.toml",
	TimestampFile: system.CurrentDir + "/beegopro.timestamp",
	GoModFile:     system.CurrentDir + "/go.mod",
	Option: Option{
		Debug:          false,
		ContextDebug:   false,
		Dsn:            "",
		Driver:         "mysql",
		ProType:        "default",
		ApiPrefix:      "/",
		EnableModule:   nil,
		Models:         make(map[string]ModelContent, 0),
		GitRemotePath:  "https://github.com/beego-dev/beego-pro.git",
		Branch:         "master",
		GitLocalPath:   system.BeegoHome + "/beego-pro",
		EnableFormat:   true,
		SourceGen:      "text",
		EnableGitPull:  true,
		RefreshGitTime: 24 * 3600,
		Path: map[string]string{
			"beego": ".",
		},
		EnableGomod: true,
	},
	GenerateTime:     time.Now().Format(MDateFormat),
	GenerateTimeUnix: time.Now().Unix(),
	Tmpl:             Tmpl{},
	CurPath:          system.CurrentDir,
	EnableModules:    make(map[string]interface{}, 0), // get the user configuration, get the enable module result
	FunctionOnce:     make(map[string]sync.Once, 0),   // get the tmpl configuration, get the function once result
}

func (c *Container) Run() {
	// init git refresh cache time
	c.initTimestamp()
	c.initBeegoPro()
	c.initBeegoTmpl()
	c.initRender()
	c.flushTimestamp()
}

func (c *Container) initBeegoPro() {
	if !utils.IsExist(c.BeegoProFile) {
		beeLogger.Log.Fatalf("beego pro config is not exist, beego json path: %s", c.BeegoProFile)
		return
	}
	viper.SetConfigFile(c.BeegoProFile)
	err := viper.ReadInConfig()
	if err != nil {
		beeLogger.Log.Fatalf("read beego pro config content, err: %s", err.Error())
		return
	}

	err = viper.Unmarshal(&c.Option)
	if err != nil {
		beeLogger.Log.Fatalf("beego pro config unmarshal error, err: %s", err.Error())
		return
	}
	if c.Option.Debug {
		viper.Debug()
	}

	if c.Option.EnableGomod {
		if !utils.IsExist(c.GoModFile) {
			beeLogger.Log.Fatalf("go mod not exist, please create go mod file")
			return
		}
	}

	for _, value := range c.Option.EnableModule {
		c.EnableModules[value] = struct{}{}
	}

	if len(c.EnableModules) == 0 {
		c.EnableModules["*"] = struct{}{}
	}

	if c.Option.Debug {
		fmt.Println("c.modules", c.EnableModules)
	}
}

func (c *Container) initBeegoTmpl() {
	if c.Option.EnableGitPull && (c.GenerateTimeUnix-c.Timestamp.GitCacheLastRefresh > c.Option.RefreshGitTime) {
		err := git.CloneORPullRepo(c.Option.GitRemotePath, c.Option.GitLocalPath)
		if err != nil {
			beeLogger.Log.Fatalf("beego pro git clone or pull repo error, err: %s", err)
			return
		}
		c.Timestamp.GitCacheLastRefresh = c.GenerateTimeUnix
	}

	tree, err := toml.LoadFile(c.Option.GitLocalPath + "/" + c.Option.ProType + "/bee.toml")

	if err != nil {
		beeLogger.Log.Fatalf("beego tmpl exec error, err: %s", err)
		return
	}
	err = tree.Unmarshal(&c.Tmpl)
	if err != nil {
		beeLogger.Log.Fatalf("beego tmpl parse error, err: %s", err)
		return
	}

	if c.Option.Debug {
		spew.Dump("tmpl", c.Tmpl)
	}

	for _, value := range c.Tmpl.Descriptor {
		if value.Once == true {
			c.FunctionOnce[value.SrcName] = sync.Once{}
		}
	}
}

type modelInfo struct {
	Module       string
	ModelName    string
	Option       Option
	Content      ModelContent
	Descriptor   Descriptor
	TmplPath     string
	GenerateTime string
}

func (c *Container) initRender() {
	for _, desc := range c.Tmpl.Descriptor {
		_, allFlag := c.EnableModules["*"]
		_, moduleFlag := c.EnableModules[desc.Module]
		if !allFlag && !moduleFlag {
			continue
		}

		// model table name, model table schema
		for modelName, content := range c.Option.Models {
			m := modelInfo{
				Module:       desc.Module,
				ModelName:    modelName,
				Content:      content,
				Option:       c.Option,
				Descriptor:   desc,
				TmplPath:     c.Tmpl.RenderPath,
				GenerateTime: c.GenerateTime,
			}

			// some render exec once
			syncOnce, flag := c.FunctionOnce[desc.SrcName]
			if flag {
				syncOnce.Do(func() {
					c.renderModel(m)
				})
				continue
			}
			c.renderModel(m)
		}
	}
}

func (c *Container) renderModel(m modelInfo) {
	render := NewRender(m)
	render.Exec(m.Descriptor.SrcName)
	if render.Descriptor.IsExistScript() {
		err := render.Descriptor.ExecScript(c.CurPath)
		if err != nil {
			beeLogger.Log.Fatalf("beego exec shell error, err: %s", err)
		}
	}
}

func (c *Container) initTimestamp() {
	if utils.IsExist(c.TimestampFile) {
		tree, err := toml.LoadFile(c.TimestampFile)
		if err != nil {
			beeLogger.Log.Fatalf("beego timestamp tmpl exec error, err: %s", err)
			return
		}
		err = tree.Unmarshal(&c.Timestamp)
		if err != nil {
			beeLogger.Log.Fatalf("beego timestamp tmpl parse error, err: %s", err)
			return
		}
	}
	c.Timestamp.Generate = c.GenerateTimeUnix
}

func (c *Container) flushTimestamp() {
	tomlByte, err := toml.Marshal(c.Timestamp)
	if err != nil {
		beeLogger.Log.Fatalf("marshal timestamp tmpl parse error, err: %s", err)
	}
	err = ioutil.WriteFile(c.TimestampFile, tomlByte, 0644)
	if err != nil {
		beeLogger.Log.Fatalf("flush timestamp tmpl parse error, err: %s", err)
	}
}

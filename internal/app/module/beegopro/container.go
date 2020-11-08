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
	TimestampFile: system.CurrentDir + "/.beegopro.timestamp",
	GoModFile:     system.CurrentDir + "/go.mod",
	UserOption: UserOption{
		Debug:         false,
		ContextDebug:  false,
		Dsn:           "",
		Driver:        "mysql",
		ProType:       "default",
		ApiPrefix:     "/api",
		EnableModule:  nil,
		Models:        make(map[string]TextModel),
		GitRemotePath: "https://github.com/beego/beego-pro.git",
		Branch:        "master",
		GitLocalPath:  system.BeegoHome + "/beego-pro",
		EnableFormat:  true,
		SourceGen:     "text",
		EnableGitPull: true,
		Path: map[string]string{
			"beego": ".",
		},
		EnableGomod:    true,
		RefreshGitTime: 24 * 3600,
		Extend:         nil,
	},
	GenerateTime:     time.Now().Format(MDateFormat),
	GenerateTimeUnix: time.Now().Unix(),
	TmplOption:       TmplOption{},
	CurPath:          system.CurrentDir,
	EnableModules:    make(map[string]interface{}), // get the user configuration, get the enable module result
	FunctionOnce:     make(map[string]sync.Once),   // get the tmpl configuration, get the function once result
}

func (c *Container) Run() {
	// init git refresh cache time
	c.initTimestamp()
	c.initUserOption()
	c.initTemplateOption()
	c.initParser()
	c.initRender()
	c.flushTimestamp()
}

func (c *Container) initUserOption() {
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

	err = viper.Unmarshal(&c.UserOption)
	if err != nil {
		beeLogger.Log.Fatalf("beego pro config unmarshal error, err: %s", err.Error())
		return
	}
	if c.UserOption.Debug {
		viper.Debug()
	}

	if c.UserOption.EnableGomod {
		if !utils.IsExist(c.GoModFile) {
			beeLogger.Log.Fatalf("go mod not exist, please create go mod file")
			return
		}
	}

	for _, value := range c.UserOption.EnableModule {
		c.EnableModules[value] = struct{}{}
	}

	if len(c.EnableModules) == 0 {
		c.EnableModules["*"] = struct{}{}
	}

	if c.UserOption.Debug {
		fmt.Println("c.modules", c.EnableModules)
	}

}

func (c *Container) initTemplateOption() {
	if c.UserOption.EnableGitPull && (c.GenerateTimeUnix-c.Timestamp.GitCacheLastRefresh > c.UserOption.RefreshGitTime) {
		err := git.CloneORPullRepo(c.UserOption.GitRemotePath, c.UserOption.GitLocalPath)
		if err != nil {
			beeLogger.Log.Fatalf("beego pro git clone or pull repo error, err: %s", err)
			return
		}
		c.Timestamp.GitCacheLastRefresh = c.GenerateTimeUnix
	}

	tree, err := toml.LoadFile(c.UserOption.GitLocalPath + "/" + c.UserOption.ProType + "/bee.toml")

	if err != nil {
		beeLogger.Log.Fatalf("beego tmpl exec error, err: %s", err)
		return
	}
	err = tree.Unmarshal(&c.TmplOption)
	if err != nil {
		beeLogger.Log.Fatalf("beego tmpl parse error, err: %s", err)
		return
	}

	if c.UserOption.Debug {
		spew.Dump("tmpl", c.TmplOption)
	}

	for _, value := range c.TmplOption.Descriptor {
		if value.Once {
			c.FunctionOnce[value.SrcName] = sync.Once{}
		}
	}
}

func (c *Container) initParser() {
	driver, flag := ParserDriver[c.UserOption.SourceGen]
	if !flag {
		beeLogger.Log.Fatalf("parse driver not exit, source gen %s", c.UserOption.SourceGen)
	}
	driver.RegisterOption(c.UserOption, c.TmplOption)
	c.Parser = driver
}

func (c *Container) initRender() {
	for _, desc := range c.TmplOption.Descriptor {
		_, allFlag := c.EnableModules["*"]
		_, moduleFlag := c.EnableModules[desc.Module]
		if !allFlag && !moduleFlag {
			continue
		}

		models := c.Parser.GetRenderInfos(desc)
		// model table name, model table schema
		for _, m := range models {
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

func (c *Container) renderModel(m RenderInfo) {
	// todo optimize
	m.GenerateTime = c.GenerateTime
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

package beegopro

import (
	"github.com/beego/bee/internal/pkg/utils"
	beeLogger "github.com/beego/bee/logger"
	"io/ioutil"
)
var CompareExcept = []string{"@BeeGenerateTime"}
func (c *Container) GenConfig() {
	if utils.IsExist(c.BeegoProFile) {
		beeLogger.Log.Fatalf("beego pro toml exist")
		return
	}

	err := ioutil.WriteFile("beegopro.toml", []byte(BeegoToml), 0644)
	if err != nil {
		beeLogger.Log.Fatalf("write beego pro toml err: %s", err)
		return
	}
}

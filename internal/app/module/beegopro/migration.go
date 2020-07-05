package beegopro

import (
	"database/sql"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
	"io/ioutil"
	"path/filepath"
)

var SQL utils.DocValue

func (c *Container) Migration(args []string) {
	c.initUserOption()
	db, err := sql.Open(c.UserOption.Driver, c.UserOption.Dsn)
	if err != nil {
		beeLogger.Log.Fatalf("Could not connect to '%s' database using '%s': %s", c.UserOption.Driver, c.UserOption.Dsn, err)
		return
	}

	defer db.Close()

	absFile, _ := filepath.Abs(SQL.String())
	content, err := ioutil.ReadFile(SQL.String())
	if err != nil {
		beeLogger.Log.Errorf("read file err %s, abs file %s", err, absFile)
	}

	result, err := db.Exec(string(content))
	if err != nil {
		beeLogger.Log.Errorf("db exec err %s", err)
	}
	beeLogger.Log.Infof("db exec info %v", result)

}

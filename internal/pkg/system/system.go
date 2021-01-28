package system

import (
	"os"
	"os/user"
	"path/filepath"
)

// Bee System Params ...
var (
	Usr, _     = user.Current()
	BeegoHome  = filepath.Join(Usr.HomeDir, "/.beego")
	CurrentDir = getCurrentDirectory()
	GoPath     = os.Getenv("GOPATH")
)

func getCurrentDirectory() string {
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return ""
}

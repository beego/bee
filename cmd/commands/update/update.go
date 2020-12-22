package update

import (
	"flag"
	"os"
	"os/exec"

	"github.com/beego/bee/v2/cmd/commands"
	"github.com/beego/bee/v2/config"
	beeLogger "github.com/beego/bee/v2/logger"
	"github.com/beego/bee/v2/utils"
)

var CmdUpdate = &commands.Command{
	UsageLine: "update",
	Short:     "Update Bee",
	Long: `
Automatic run command "go get -u github.com/beego/bee/v2" for selfupdate
`,
	Run: updateBee,
}

func init() {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	CmdUpdate.Flag = *fs
	commands.AvailableCommands = append(commands.AvailableCommands, CmdUpdate)
}

func updateBee(cmd *commands.Command, args []string) int {
	beeLogger.Log.Info("Updating")
	beePath := config.GitRemotePath
	cmdUp := exec.Command("go", "get", "-u", beePath)
	cmdUp.Stdout = os.Stdout
	cmdUp.Stderr = os.Stderr
	if err := cmdUp.Run(); err != nil {
		beeLogger.Log.Warnf("Run cmd err:%s", err)
	}
	// update the Time when updateBee every time
	utils.UpdateLastPublishedTime()
	return 0
}

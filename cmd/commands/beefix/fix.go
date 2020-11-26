package beefix

import (
	"strings"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	beeLogger "github.com/beego/bee/logger"
	"github.com/beego/bee/utils"
)

var CmdFix = &commands.Command{
	UsageLine: "fix",
	Short:     "Fixes your application by making it compatible with newer versions of Beego",
	Long: `
  The command 'fix' will try to solve those issues by upgrading your code base
  to be compatible  with Beego old version
  -s source version
  -t target version

  example: bee fix -s 1 -t 2 means that upgrade Beego version from v1.x to v2.x
`,
}

var (
	source, target utils.DocValue
)

func init() {
	CmdFix.Run = runFix
	CmdFix.PreRun = func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() }
	CmdFix.Flag.Var(&source, "s", "source version")
	CmdFix.Flag.Var(&target, "t", "target version")
	commands.AvailableCommands = append(commands.AvailableCommands, CmdFix)
}

func runFix(cmd *commands.Command, args []string) int {
	t := target.String()
	if t == "" || t == "1.6"{
		return fixTo16(cmd, args)
	} else if strings.HasPrefix(t, "2") {
		// upgrade to v2
		return fix1To2()
	}

	beeLogger.Log.Info("The target is compatible version, do nothing")
	return 0
}

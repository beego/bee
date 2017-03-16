package scripts

import (
	"os/exec"

	"os"

	"runtime"

	"strings"

	"github.com/beego/bee/cmd/commands"
	"github.com/beego/bee/cmd/commands/version"
	"github.com/beego/bee/config"
	"github.com/beego/bee/logger"
)

func init() {
	for commandName, command := range config.Conf.Scripts {
		CmdNew := &commands.Command{
			UsageLine: commandName,
			Short:     command,
			PreRun:    func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
			Run:       RunScript,
		}
		commands.AvailableCommands = append(commands.AvailableCommands, CmdNew)
	}
}

func RunScript(cmd *commands.Command, args []string) int {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		c = exec.Command("sh", "-c", cmd.Short+" "+strings.Join(args, " "))
	case "windows": //TODO
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		beeLogger.Log.Error(err.Error())
	}
	return 0
}

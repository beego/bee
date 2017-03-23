package commands

import (
	"flag"
	"io"
	"os"
	"strings"

	"github.com/beego/bee/logger/colors"
	"github.com/beego/bee/utils"
)

// Command is the unit of execution
type Command struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(cmd *Command, args []string) int

	// PreRun performs an operation before running the command
	PreRun func(cmd *Command, args []string)

	// UsageLine is the one-line Usage message.
	// The first word in the line is taken to be the command name.
	UsageLine string

	// Short is the short description shown in the 'go help' output.
	Short string

	// Long is the long message shown in the 'go help <this-command>' output.
	Long string

	// Flag is a set of flags specific to this command.
	Flag flag.FlagSet

	// CustomFlags indicates that the command will do its own
	// flag parsing.
	CustomFlags bool

	// output out writer if set in SetOutput(w)
	output *io.Writer
}

var AvailableCommands = []*Command{}
var cmdUsage = `Use {{printf "bee help %s" .Name | bold}} for more information.{{endline}}`

// Name returns the command's name: the first word in the Usage line.
func (c *Command) Name() string {
	name := c.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

// SetOutput sets the destination for Usage and error messages.
// If output is nil, os.Stderr is used.
func (c *Command) SetOutput(output io.Writer) {
	c.output = &output
}

// Out returns the out writer of the current command.
// If cmd.output is nil, os.Stderr is used.
func (c *Command) Out() io.Writer {
	if c.output != nil {
		return *c.output
	}
	return colors.NewColorWriter(os.Stderr)
}

// Usage puts out the Usage for the command.
func (c *Command) Usage() {
	utils.Tmpl(cmdUsage, c)
	os.Exit(2)
}

// Runnable reports whether the command can be run; otherwise
// it is a documentation pseudo-command such as import path.
func (c *Command) Runnable() bool {
	return c.Run != nil
}

func (c *Command) Options() map[string]string {
	options := make(map[string]string)
	c.Flag.VisitAll(func(f *flag.Flag) {
		defaultVal := f.DefValue
		if len(defaultVal) > 0 {
			options[f.Name+"="+defaultVal] = f.Usage
		} else {
			options[f.Name] = f.Usage
		}
	})
	return options
}

// Copyright 2013 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// Bee is a tool for developing applications based on beego framework.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/template"
)

const version = "1.6.2"

// Command is the unit of execution
type Command struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(cmd *Command, args []string) int

	// PreRun performs an operation before running the command
	PreRun func(cmd *Command, args []string)

	// UsageLine is the one-line usage message.
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

// Name returns the command's name: the first word in the usage line.
func (c *Command) Name() string {
	name := c.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

// SetOutput sets the destination for usage and error messages.
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
	return NewColorWriter(os.Stderr)
}

// Usage puts out the usage for the command.
func (c *Command) Usage() {
	tmpl(cmdUsage, c)
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
			if strings.Contains(defaultVal, ":") {
				// Truncate the flag's default value by appending '...' at the end
				options[f.Name+"="+strings.Split(defaultVal, ":")[0]+":..."] = f.Usage
			} else {
				options[f.Name+"="+defaultVal] = f.Usage
			}
		} else {
			options[f.Name] = f.Usage
		}
	})
	return options
}

var availableCommands = []*Command{
	cmdNew,
	cmdRun,
	cmdPack,
	cmdApiapp,
	cmdHproseapp,
	//cmdRouter,
	//cmdTest,
	cmdBale,
	cmdVersion,
	cmdGenerate,
	//cmdRundocs,
	cmdMigrate,
	cmdFix,
}

var logger = GetBeeLogger(os.Stdout)

func main() {
	currentpath, _ := os.Getwd()

	flag.Usage = usage
	flag.Parse()
	log.SetFlags(0)

	args := flag.Args()

	if len(args) < 1 {
		usage()
	}

	if args[0] == "help" {
		help(args[1:])
		return
	}

	for _, cmd := range availableCommands {
		if cmd.Name() == args[0] && cmd.Run != nil {
			cmd.Flag.Usage = func() { cmd.Usage() }
			if cmd.CustomFlags {
				args = args[1:]
			} else {
				cmd.Flag.Parse(args[1:])
				args = cmd.Flag.Args()
			}

			if cmd.PreRun != nil {
				cmd.PreRun(cmd, args)
			}

			// Check if current directory is inside the GOPATH,
			// if so parse the packages inside it.
			if strings.Contains(currentpath, GetGOPATHs()[0]+"/src") && isGenerateDocs(cmd.Name(), args) {
				parsePackagesFromDir(currentpath)
			}

			os.Exit(cmd.Run(cmd, args))
			return
		}
	}

	printErrorAndExit("Unknown subcommand")
}

func isGenerateDocs(name string, args []string) bool {
	if name != "generate" {
		return false
	}
	for _, a := range args {
		if a == "docs" {
			return true
		}
	}
	return false
}

var usageTemplate = `Bee is a Fast and Flexible tool for managing your Beego Web Application.

{{"USAGE" | headline}}
    {{"bee command [arguments]" | bold}}

{{"AVAILABLE COMMANDS" | headline}}
{{range .}}{{if .Runnable}}
    {{.Name | printf "%-11s" | bold}} {{.Short}}{{end}}{{end}}

Use {{"bee help [command]" | bold}} for more information about a command.

{{"ADDITIONAL HELP TOPICS" | headline}}
{{range .}}{{if not .Runnable}}
    {{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use {{"bee help [topic]" | bold}} for more information about that topic.
`

var helpTemplate = `{{"USAGE" | headline}}
  {{.UsageLine | printf "bee %s" | bold}}
{{if .Options}}{{endline}}{{"OPTIONS" | headline}}{{range $k,$v := .Options}}
  {{$k | printf "-%-12s" | bold}} {{$v}}{{end}}{{endline}}{{end}}
{{"DESCRIPTION" | headline}}
  {{tmpltostr .Long . | trim}}
`

var errorTemplate = `bee: %s.
Use {{"bee help" | bold}} for more information.
`

var cmdUsage = `Use {{printf "bee help %s" .Name | bold}} for more information.{{endline}}`

func usage() {
	tmpl(usageTemplate, availableCommands)
	os.Exit(2)
}

func tmpl(text string, data interface{}) {
	output := NewColorWriter(os.Stderr)

	t := template.New("usage").Funcs(BeeFuncMap())
	template.Must(t.Parse(text))

	err := t.Execute(output, data)
	MustCheck(err)
}

func help(args []string) {
	if len(args) == 0 {
		usage()
	}
	if len(args) != 1 {
		printErrorAndExit("Too many arguments")
	}

	arg := args[0]

	for _, cmd := range availableCommands {
		if cmd.Name() == arg {
			tmpl(helpTemplate, cmd)
			return
		}
	}
	printErrorAndExit("Unknown help topic")
}

func printErrorAndExit(message string) {
	tmpl(fmt.Sprintf(errorTemplate, message), nil)
	os.Exit(2)
}

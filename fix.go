package main

var cmdFix = &Command{
	UsageLine: "fix",
	Short:     "fix the beego application to compatibel with beego 1.6",
	Long: `
As from beego1.6, there's some incompatible code with the old version.

bee fix help to upgrade the application to beego 1.6
`,
}

func init() {
	cmdFix.Run = runFix
}

func runFix(cmd *Command, args []string) int {
	return 0
}

package main

import "os"

var cmdGenerate = &Command{
	UsageLine: "generate [Command]",
	Short:     "generate code based on application",
	Long: `
	bee g model [modelfile] [dbconfig]
        generate model base on struct
    bee g controller [modelfile]
        generate RESTFul controllers based on modelfile             
    bee g router [controllerfile]
	    generate router based on controllerfile
    bee g docs	
        generate swagger doc file
    bee g test [routerfile]
	    	generate testcase
`,
}

func generateCode(cmd *Command, args []string) {
	curpath, _ := os.Getwd()
	if len(args) < 1 {
		ColorLog("[ERRO] command is missing\n")
		os.Exit(2)
	}

	gopath := os.Getenv("GOPATH")
	Debugf("gopath:%s", gopath)
	if gopath == "" {
		ColorLog("[ERRO] $GOPATH not found\n")
		ColorLog("[HINT] Set $GOPATH in your environment vairables\n")
		os.Exit(2)
	}

	gcmd := args[0]
	switch gcmd {
	case "docs":
		generateDocs(curpath)
	default:
		ColorLog("[ERRO] command is missing\n")
	}
	ColorLog("[SUCC] generate successfully created!\n")
}

// Copyright 2016 bee authors
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

package dockerize

import (
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/beego/bee/v2/cmd/commands"
	"github.com/beego/bee/v2/cmd/commands/version"
	beeLogger "github.com/beego/bee/v2/logger"
	"github.com/beego/bee/v2/utils"
)

const dockerBuildTemplate = `FROM {{.BaseImage}}

WORKDIR {{.Appdir}}

COPY . .
RUN go get -v && go build -v -o /usr/local/bin/{{.Entrypoint}}

EXPOSE {{.Expose}}
CMD ["{{.Entrypoint}}"]
`

const composeBuildTemplate = `version: '3'
networks:
  {{.Appname}}_network_compose:
    driver: bridge 
services:
  {{.Appname}}:
    container_name: {{.Appname}}
    build: .
    restart: unless-stopped
    networks:
      {{.Appname}}_network_compose:
    ports:{{.Expose}}
`

// Dockerfile holds the information about the Docker container.
type Dockerfile struct {
	BaseImage  string
	Appdir     string
	Entrypoint string
	Expose     string
}

// docker-compose.yaml
type Composefile struct {
	Appname   string
	Expose    string
}

var CmdDockerize = &commands.Command{
	CustomFlags: true,
	UsageLine:   "dockerize",
	Short:       "Generates a Dockerfile and docker-compose.yaml for your Beego application",
	Long: `Dockerize generates a Dockerfile and docker-compose.yaml for your Beego Web Application.
  The Dockerfile will compile and run the application.
  The docker-compose.yaml can be used to build and deploy the generated Dockerfile.
  {{"Example:"|bold}}
    $ bee dockerize -expose="3000,80,25"
  `,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    dockerizeApp,
}

var (
	expose    string
	baseImage string
)

func init() {
	fs := flag.NewFlagSet("dockerize", flag.ContinueOnError)
	fs.StringVar(&baseImage, "baseimage", "golang:1.20.2", "Set the base image of the Docker container.")
	fs.StringVar(&expose, "expose", "8080", "Port(s) to expose for the Docker container.")
	CmdDockerize.Flag = *fs
	commands.AvailableCommands = append(commands.AvailableCommands, CmdDockerize)
}

func dockerizeApp(cmd *commands.Command, args []string) int {
	if err := cmd.Flag.Parse(args); err != nil {
		beeLogger.Log.Fatalf("Error parsing flags: %v", err.Error())
	}

	beeLogger.Log.Info("Generating Dockerfile and docker-compose.yaml...")

	gopath := os.Getenv("GOPATH")
	dir, err := filepath.Abs(".")
	if err != nil {
		beeLogger.Log.Error(err.Error())
	}

	appdir := strings.Replace(dir, gopath, "", 1)

	// In case of multiple ports to expose inside the container,
	// replace all the commas with whitespaces.
	// See the verb EXPOSE in the Docker documentation.
	exposedockerfile := strings.Replace(expose, ",", " ", -1)
	
	// Multiple ports expose for docker-compose.yaml
        ports := strings.Fields(strings.Replace(expose, ",", " ", -1))
        exposecompose := ""
        for _, port := range ports {
            composeport := ("\n    - " + "\"" + port + ":" + port + "\"")
            exposecompose += composeport
        }
	
	
	_, entrypoint := path.Split(appdir)
	dockerfile := Dockerfile{
		BaseImage:  baseImage,
		Appdir:     appdir,
		Entrypoint: entrypoint,
		Expose:     exposedockerfile,
	}
	composefile := Composefile{
		Appname:    entrypoint,
		Expose:     exposecompose,
	}

	generateDockerfile(dockerfile)
	generatecomposefile(composefile)
	return 0
}

func generateDockerfile(df Dockerfile) {
	t := template.Must(template.New("dockerBuildTemplate").Parse(dockerBuildTemplate)).Funcs(utils.BeeFuncMap())

	f, err := os.Create("Dockerfile")
	if err != nil {
		beeLogger.Log.Fatalf("Error writing Dockerfile: %v", err.Error())
	}
	defer utils.CloseFile(f)

	t.Execute(f, df)

	beeLogger.Log.Success("Dockerfile generated.")
}

func generatecomposefile(df Composefile) {
	t := template.Must(template.New("composeBuildTemplate").Parse(composeBuildTemplate)).Funcs(utils.BeeFuncMap())

	f, err := os.Create("docker-compose.yaml")
	if err != nil {
		beeLogger.Log.Fatalf("Error writing docker-compose.yaml: %v", err.Error())
	}
	defer utils.CloseFile(f)

	t.Execute(f, df)

	beeLogger.Log.Success("docker-compose.yaml generated.")
}

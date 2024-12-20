
bee
===

Bee is a command-line tool facilitating development of Beego-based application.

[![Build Status](https://img.shields.io/travis/beego/bee.svg?branch=master&label=master)](https://travis-ci.org/beego/bee)
[![Build Status](https://img.shields.io/travis/beego/bee.svg?branch=develop&label=develop)](https://travis-ci.org/beego/bee)

## Requirements

- Go version >= 1.13

## Installation

To install or update `bee` use the `go install` command:

```bash
go install github.com/beego/bee/v2@latest
```

## Then you can add `bee` binary to PATH environment variable in your `~/.bashrc` or `~/.bash_profile` file:

```bash
export PATH=$PATH:<your_main_gopath>/bin
```

## Installing and updating bee prior Go version 1.17

To install `bee` use the `go get` command:

```bash
go get github.com/beego/bee/v2
```

> If you already have `bee` installed, updating `bee` is simple:

```bash
go get -u github.com/beego/bee/v2
```

## Basic commands

Bee provides a variety of commands which can be helpful at various stages of development. The top level commands include:

```
    version     Prints the current Bee version
    migrate     Runs database migrations
    api         Creates a Beego API application
    bale        Transforms non-Go files to Go source files
    fix         Fixes your application by making it compatible with newer versions of Beego
    pro         Source code generator
    dlv         Start a debugging session using Delve
    dockerize   Generates a Dockerfile and docker-compose.yaml for your Beego application
    generate    Source code generator
    hprose      Creates an RPC application based on Hprose and Beego frameworks
    new         Creates a Beego application
    pack        Compresses a Beego application into a single file
    rs          Run customized scripts
    run         Run the application by starting a local development server
    server      serving static content over HTTP on port
    update      Update Bee
```

### bee version

To display the current version of `bee`, `beego` and `go` installed on your machine:

```bash
$ bee version
______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v2.0.4

├── Beego     : 2.0.4
├── GoVersion : go1.14.1
├── GOOS      : darwin
├── GOARCH    : amd64
├── NumCPU    : 4
├── GOPATH    : /home/beeuser/.go
├── GOROOT    : /usr/local/Cellar/go/1.14.1/libexec
├── Compiler  : gc
└── Published : 2020-09-13
```

You can also change the output format using `-o` flag:

```bash
$ bee version -o json
{
    "GoVersion": "go1.14.1",
    "GOOS": "darwin",
    "GOARCH": "amd64",
    "NumCPU": 4,
    "GOPATH": "/home/beeuser/.go",
    "GOROOT": "/usr/local/Cellar/go/1.14.1/libexec",
    "Compiler": "gc",
    "BeeVersion": "2.0.4",
    "BeegoVersion": "2.0.4",
    "Published": "2020-09-13"
}
```

For more information on the usage, run `bee help version`.

### bee new

To create a new Beego web application:

```bash
$ bee new my-web-app
2020/09/14 22:28:51 INFO     ▶ 0001 generate new project support go modules.
2020/09/14 22:28:51 INFO     ▶ 0002 Creating application...
	create	 /Users/beeuser/learn/my-web-app/go.mod
	create	 /Users/beeuser/learn/my-web-app/
	create	 /Users/beeuser/learn/my-web-app/conf/
	create	 /Users/beeuser/learn/my-web-app/controllers/
	create	 /Users/beeuser/learn/my-web-app/models/
	create	 /Users/beeuser/learn/my-web-app/routers/
	create	 /Users/beeuser/learn/my-web-app/tests/
	create	 /Users/beeuser/learn/my-web-app/static/
	create	 /Users/beeuser/learn/my-web-app/static/js/
	create	 /Users/beeuser/learn/my-web-app/static/css/
	create	 /Users/beeuser/learn/my-web-app/static/img/
	create	 /Users/beeuser/learn/my-web-app/views/
	create	 /Users/beeuser/learn/my-web-app/conf/app.conf
	create	 /Users/beeuser/learn/my-web-app/controllers/default.go
	create	 /Users/beeuser/learn/my-web-app/views/index.tpl
	create	 /Users/beeuser/learn/my-web-app/routers/router.go
	create	 /Users/beeuser/learn/my-web-app/tests/default_test.go
	create	 /Users/beeuser/learn/my-web-app/main.go
2020/09/14 22:28:51 SUCCESS  ▶ 0003 New application successfully created!
```

For more information on the usage, run `bee help new`.

### bee run

To run the application we just created, you can navigate to the application folder and execute:

```bash
$ cd my-web-app && bee run
```

For more information on the usage, run `bee help run`.

### bee pack

To compress a Beego application into a single deployable file:

```bash
$ bee pack
______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v2.0.0
2016/12/26 22:29:29 INFO     ▶ 0001 Packaging application on '/home/beeuser/.go/src/github.com/user/my-web-app'...
2016/12/26 22:29:29 INFO     ▶ 0002 Building application...
2016/12/26 22:29:29 INFO     ▶ 0003 Using: GOOS=linux GOARCH=amd64
2016/12/26 22:29:31 SUCCESS  ▶ 0004 Build Successful!
2016/12/26 22:29:31 INFO     ▶ 0005 Writing to output: /home/beeuser/.go/src/github.com/user/my-web-app/my-web-app.tar.gz
2016/12/26 22:29:31 INFO     ▶ 0006 Excluding relpath prefix: .
2016/12/26 22:29:31 INFO     ▶ 0007 Excluding relpath suffix: .go:.DS_Store:.tmp
2016/12/26 22:29:32 SUCCESS  ▶ 0008 Application packed!
```

For more information on the usage, run `bee help pack`.

### bee rs 
Inspired by makefile / npm scripts.
  Run script allows you to run arbitrary commands using Bee.
  Custom commands are provided from the "scripts" object inside bee.json or Beefile.

  To run a custom command, use: $ bee rs mycmd ARGS

```bash
$ bee help rs

USAGE
  bee rs

DESCRIPTION
  Run script allows you to run arbitrary commands using Bee.
  Custom commands are provided from the "scripts" object inside bee.json or Beefile.

  To run a custom command, use: $ bee rs mycmd ARGS
  
AVAILABLE SCRIPTS
  gtest
      APP_ENV=test APP_CONF_PATH=$(pwd)/conf go test -v -cover
  gtestall
      APP_ENV=test APP_CONF_PATH=$(pwd)/conf go test -v -cover $(go list ./... | grep -v /vendor/)

```

*Run your scripts with:*
```$ bee rs gtest tests/*.go```
```$ bee rs gtestall```


### bee api

To create a Beego API application:

```bash
$ bee api my-api
______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v2.0.0
2020/09/14 22:35:11 INFO     ▶ 0001 generate api project support go modules.
2020/09/14 22:35:11 INFO     ▶ 0002 Creating API...
	create	 /Users/beeuser/code/learn/my-api/go.mod
	create	 /Users/beeuser/code/learn/my-api
	create	 /Users/beeuser/code/learn/my-api/conf
	create	 /Users/beeuser/code/learn/my-api/controllers
	create	 /Users/beeuser/code/learn/my-api/tests
	create	 /Users/beeuser/code/learn/my-api/conf/app.conf
	create	 /Users/beeuser/code/learn/my-api/models
	create	 /Users/beeuser/code/learn/my-api/routers/
	create	 /Users/beeuser/code/learn/my-api/controllers/object.go
	create	 /Users/beeuser/code/learn/my-api/controllers/user.go
	create	 /Users/beeuser/code/learn/my-api/tests/default_test.go
	create	 /Users/beeuser/code/learn/my-api/routers/router.go
	create	 /Users/beeuser/code/learn/my-api/models/object.go
	create	 /Users/beeuser/code/learn/my-api/models/user.go
	create	 /Users/beeuser/code/learn/my-api/main.go
2020/09/14 22:35:11 SUCCESS  ▶ 0003 New API successfully created!
```

For more information on the usage, run `bee help api`.

### bee hprose

To create an Hprose RPC application based on Beego:

```bash
$ bee hprose my-rpc-app
______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v2.0.0
2020/09/14 22:36:39 INFO     ▶ 0001 generate api project support go modules.
2020/09/14 22:36:39 INFO     ▶ 0002 Creating Hprose application...
	create	 /Users/beeuser/code/learn/my-rpc-app/go.mod
	create	 /Users/beeuser/code/learn/my-rpc-app
	create	 /Users/beeuser/code/learn/my-rpc-app/conf
	create	 /Users/beeuser/code/learn/my-rpc-app/conf/app.conf
	create	 /Users/beeuser/code/learn/my-rpc-app/models
	create	 /Users/beeuser/code/learn/my-rpc-app/models/object.go
	create	 /Users/beeuser/code/learn/my-rpc-app/models/user.go
	create	 /Users/beeuser/code/learn/my-rpc-app/main.go
2020/09/14 22:36:39 SUCCESS  ▶ 0003 New Hprose application successfully created!
```

For more information on the usage, run `bee help hprose`.

### bee bale

To pack all the static files into Go source files:

```bash
$ bee bale
______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v2.0.0
2020/09/14 22:37:56 SUCCESS  ▶ 0001 Baled resources successfully!
```

For more information on the usage, run `bee help bale`.

### bee migrate

For database migrations, use `bee migrate`.

For more information on the usage, run `bee help migrate`.

### bee generate

Bee also comes with a source code generator which speeds up the development.

For example, to generate a new controller named `hello`:

```bash
$ bee generate controller hello
______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v2.0.0
2020/09/14 22:38:44 INFO     ▶ 0001 Using 'Hello' as controller name
2020/09/14 22:38:44 INFO     ▶ 0002 Using 'controllers' as package name
	create	 /Users/beeuser/code/learn/my-api/controllers/hello.go
2020/09/14 22:38:44 SUCCESS  ▶ 0003 Controller successfully generated!
```

For more information on the usage, run `bee help generate`.

### bee dockerize

Bee also helps you dockerize your Beego application by generating a Dockerfile and a docker-compose.yaml file.

For example, to generate a Dockerfile with `golang:1.20.1` baseimage and exposing port `9000`:

```bash
$ bee dockerize -baseimage=golang:1.20.1 -expose=9000
______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v2.0.4
2023/05/02 21:03:05 INFO     ▶ 0001 Generating Dockerfile and docker-compose.yaml...
2023/05/02 21:03:05 SUCCESS  ▶ 0002 Dockerfile generated.
2023/05/02 21:03:05 SUCCESS  ▶ 0003 docker-compose.yaml generated.
```

For more information on the usage, run `bee help dockerize`.

### bee dlv

Bee can also help with debugging your application. To start a debugging session:

```bash
______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v2.0.0
2020/09/14 22:40:12 INFO     ▶ 0001 Starting Delve Debugger...
Type 'help' for list of commands.
(dlv) break main.main
Breakpoint 1 set at 0x40100f for main.main() ./main.go:8

(dlv) continue
> main.main() ./main.go:8 (hits goroutine(1):1 total:1) (PC: 0x40100f)
     3:	import (
     4:		_ "github.com/user/myapp/routers"
     5:		beego "github.com/beego/beego/v2/server/web"
     6:	)
     7:	
=>   8:	func main() {
     9:		beego.Run()
    10:	}
    11:
```

For more information on the usage, run `bee help dlv`.

### bee pro 

#### bee pro toml

To create a beegopro.toml file

```bash
$ bee pro toml
2020/09/14 22:51:18 SUCCESS  ▶ 0001 Successfully created file beegopro.toml
2020/09/14 22:51:18 SUCCESS  ▶ 0002 Toml successfully generated!
```

#### bee pro gen

Source code generator by beegopro.toml

```bash
$ bee pro gen
2020/09/14 23:01:13 INFO     ▶ 0001 Create /Users/beeuser/.beego/beego-pro Success!
2020/09/14 23:01:13 INFO     ▶ 0002 git pull /Users/beeuser/.beego/beego-pro
2020/09/14 23:01:15 INFO     ▶ 0003 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0004 Using 'example' as package name from controllers
2020/09/14 23:01:15 INFO     ▶ 0005 create file '/Users/beeuser/code/learn/my-web-app/controllers/bee_default_controller.go' from controllers
2020/09/14 23:01:15 INFO     ▶ 0006 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0007 Using 'example' as package name from controllers
2020/09/14 23:01:15 INFO     ▶ 0008 create file '/Users/beeuser/code/learn/my-web-app/controllers/example.go' from controllers
2020/09/14 23:01:15 INFO     ▶ 0009 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0010 Using 'example' as package name from models
2020/09/14 23:01:15 INFO     ▶ 0011 create file '/Users/beeuser/code/learn/my-web-app/models/bee_default_model.go' from models
2020/09/14 23:01:15 INFO     ▶ 0012 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0013 Using 'example' as package name from models
2020/09/14 23:01:15 INFO     ▶ 0014 create file '/Users/beeuser/code/learn/my-web-app/models/example.go' from models
2020/09/14 23:01:15 INFO     ▶ 0015 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0016 Using 'example' as package name from routers
2020/09/14 23:01:15 INFO     ▶ 0017 create file '/Users/beeuser/code/learn/my-web-app/routers/example.go' from routers
2020/09/14 23:01:15 INFO     ▶ 0018 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0019 Using 'example' as package name from example
2020/09/14 23:01:15 INFO     ▶ 0020 create file '/Users/beeuser/code/learn/my-web-app/ant/src/pages/example/list.tsx' from example
2020/09/14 23:01:15 INFO     ▶ 0021 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0022 Using 'example' as package name from example
2020/09/14 23:01:15 INFO     ▶ 0023 create file '/Users/beeuser/code/learn/my-web-app/ant/src/pages/example/formconfig.tsx' from example
2020/09/14 23:01:15 INFO     ▶ 0024 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0025 Using 'example' as package name from example
2020/09/14 23:01:15 INFO     ▶ 0026 create file '/Users/beeuser/code/learn/my-web-app/ant/src/pages/example/create.tsx' from example
2020/09/14 23:01:15 INFO     ▶ 0027 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0028 Using 'example' as package name from example
2020/09/14 23:01:15 INFO     ▶ 0029 create file '/Users/beeuser/code/learn/my-web-app/ant/src/pages/example/update.tsx' from example
2020/09/14 23:01:15 INFO     ▶ 0030 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0031 Using 'example' as package name from example
2020/09/14 23:01:15 INFO     ▶ 0032 create file '/Users/beeuser/code/learn/my-web-app/ant/src/pages/example/info.tsx' from example
2020/09/14 23:01:15 INFO     ▶ 0033 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0034 Using 'example' as package name from sql
2020/09/14 23:01:15 INFO     ▶ 0035 create file '/Users/beeuser/code/learn/my-web-app/sql/example_up.sql' from sql
2020/09/14 23:01:15 INFO     ▶ 0036 2020/09/14 23:01:15 INFO     ▶ 0001 db exec info ./sql/example_up.sql
2020/09/14 23:01:15 SUCCESS  ▶ 0002 Migration successfully generated!
2020/09/14 23:01:15 INFO     ▶ 0037 Using 'example' as name
2020/09/14 23:01:15 INFO     ▶ 0038 Using 'example' as package name from sql
2020/09/14 23:01:15 INFO     ▶ 0039 create file '/Users/beeuser/code/learn/my-web-app/sql/example_down.sql' from sql
2020/09/14 23:01:15 SUCCESS  ▶ 0040 Gen successfully generated!
```

#### 
## Shortcuts

Because you'll likely type these generator commands over and over, it makes sense to create aliases:

```bash
# Generator Stuff
alias g:a="bee generate appcode"
alias g:m="bee generate model"
alias g:c="bee generate controller"
alias g:v="bee generate view"
alias g:mi="bee generate migration"
```

These can be stored , for example, in your `~/.bash_profile` or `~/.bashrc` files.

## Help

To print more information on the usage of a particular command, use `bee help <command>`.

For instance, to get more information about the `run` command:

```bash
$ bee help run
USAGE
  bee run [appname] [watchall] [-main=*.go] [-downdoc=true]  [-gendoc=true] [-vendor=true] [-e=folderToExclude]  [-tags=goBuildTags] [-runmode=BEEGO_RUNMODE]

OPTIONS
  -downdoc
      Enable auto-download of the swagger file if it does not exist.

  -e=[]
      List of paths to exclude.

  -gendoc
      Enable auto-generate the docs.

  -main=[]
      Specify main go files.

  -runmode
      Set the Beego run mode.

  -tags
      Set the build tags. See: https://golang.org/pkg/go/build/

  -vendor=false
      Enable watch vendor folder.

DESCRIPTION
  Run command will supervise the filesystem of the application for any changes, and recompile/restart it.
```

## Contributing
Bug reports, feature requests and pull requests are always welcome.

We work on two branches: `master` for stable, released code and `develop`, a development branch.
It might be important to distinguish them when you are reading the commit history searching for a feature or a bugfix,
or when you are unsure of where to base your work from when contributing.

### Found a bug?

Please [submit an issue][new-issue] on GitHub and we will follow up.
Even better, we would appreciate a [Pull Request][new-pr] with a fix for it!

- If the bug was found in a release, it is best to base your work on `master` and submit your PR against it.
- If the bug was found on `develop` (the development branch), base your work on `develop` and submit your PR against it.

Please follow the [Pull Request Guidelines][new-pr].

### Want a feature?

Feel free to request a feature by [submitting an issue][new-issue] on GitHub and open the discussion.

If you'd like to implement a new feature, please consider opening an issue first to talk about it.
It may be that somebody is already working on it, or that there are particular issues that you should be aware of
before implementing the change. If you are about to open a Pull Request, please make sure to follow the [submissions guidelines][new-pr].

## Submission Guidelines

### Submitting an issue

Before you submit an issue, search the archive, maybe you will find that a similar one already exists.

If you are submitting an issue for a bug, please include the following:

- An overview of the issue
- Your use case (why is this a bug for you?)
- The version of `bee` you are running (include the output of `bee version`)
- Steps to reproduce the issue
- Eventually, logs from your application.
- Ideally, a suggested fix

The more information you give us, the more able to help we will be!

### Submitting a Pull Request

- First of all, make sure to base your work on the `develop` branch (the development branch):

```
  # a bugfix branch for develop would be prefixed by fix/
  # a bugfix branch for master would be prefixed by hotfix/
  $ git checkout -b feature/my-feature develop
```

- Please create commits containing **related changes**. For example, two different bugfixes should produce two separate commits.
A feature should be made of commits splitted by **logical chunks** (no half-done changes). Use your best judgement as to
how many commits your changes require.

- Write insightful and descriptive commit messages. It lets us and future contributors quickly understand your changes
without having to read your changes. Please provide a summary in the first line (50-72 characters) and eventually,
go to greater lengths in your message's body. A good example can be found in [Angular commit message format](https://github.com/angular/angular.js/blob/master/DEVELOPERS.md#commit-message-format).

- Please **include the appropriate test cases** for your patch.

- Make sure all tests pass before submitting your changes.

- Rebase your commits. It may be that new commits have been introduced on `develop`.
Rebasing will update your branch with the most recent code and make your changes easier to review:

  ```
  $ git fetch
  $ git rebase origin/develop
  ```

- Push your changes:

  ```
  $ git push origin -u feature/my-feature
  ```

- Open a pull request against the `develop` branch.

- If we suggest changes:
  - Please make the required updates (after discussion if any)
  - Only create new commits if it makes sense. Generally, you will want to amend your latest commit or rebase your branch after the new changes:

    ```
    $ git rebase -i develop
    # choose which commits to edit and perform the updates
    ```

  - Re-run the tests
  - Force push to your branch:

    ```
    $ git push origin feature/my-feature -f
    ```

[new-issue]: #submitting-an-issue
[new-pr]: #submitting-a-pull-request

## Licence

```text
Copyright 2020 bee authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

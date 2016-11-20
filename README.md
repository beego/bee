bee
===

Bee is a command-line tool facilitating development of Beego-based application.

[![Build Status](https://drone.io/github.com/beego/bee/status.png)](https://drone.io/github.com/beego/bee/latest)

## Requirements

- Go version >= 1.3.

## Installation

To install `bee` use the `go get` command:

```bash
go get github.com/beego/bee
```

Then you can add `bee` binary to PATH environment variable in your `~/.bashrc` or `~/.bash_profile` file:

```bash
export PATH=$PATH:<your_main_gopath>/bin
```

> If you already have `bee` installed, updating `bee` is simple:

```bash
go get -u github.com/beego/bee
```

## Basic commands

Bee provides a variety of commands which can be helpful at various stages of development. The top level commands include: 
```
    new         Create a Beego application
    run         Run the app and start a Web server for development
    pack        Compress a beego project into a single file
    api         Create an API beego application
    hprose      Create an rpc application use hprose base on beego framework
    bale        Packs non-Go files to Go source files
    version     Prints the current Bee version
    generate    Source code generator
    migrate     Run database migrations
    fix         Fix the Beego application to make it compatible with Beego 1.6
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
\____/  \___| \___| v1.5.0

├── Beego     : 1.7.0
├── GoVersion : go1.6.2
├── GOOS      : windows
├── GOARCH    : amd64
├── NumCPU    : 4
├── GOPATH    : C:\Users\beeuser\go
├── GOROOT    : C:\go
├── Compiler  : gc
└── Date      : Monday, 22 Aug 2016
``` 

### bee new

To create a new Beego web application:

```bash
$ bee new my-web-app
______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v1.5.0
2016/08/22 14:53:45 [INFO] Creating application...
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\conf\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\controllers\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\models\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\routers\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\tests\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\static\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\static\js\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\static\css\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\static\img\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\views\
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\conf\app.conf
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\controllers\default.go
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\views\index.tpl
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\routers\router.go
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\tests\default_test.go
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app\main.go
2016/08/22 14:53:45 [SUCC] New application successfully created!
```

For more information on the usage, run `bee help new`.

### bee run

To run the application we just created, you can navigate to the application folder and execute:

```bash
$ cd my-web-app && bee run
```

Or from anywhere in your machine:

```
$ bee run github.com/user/my-web-app
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
\____/  \___| \___| v1.5.0
2016/08/22 15:11:01 Packaging application: C:\Users\beeuser\go\src\github.com\user\my-web-app
2016/08/22 15:11:01 Building application...
2016/08/22 15:11:01 Env: GOOS=windows GOARCH=amd64
2016/08/22 15:11:08 Build successful
2016/08/22 15:11:08 Excluding relpath prefix: .
2016/08/22 15:11:08 Excluding relpath suffix: .go:.DS_Store:.tmp
2016/08/22 15:11:10 Writing to output: `C:\Users\beeuser\go\src\github.com\user\my-web-app\my-web-app.tar.gz`
```

For more information on the usage, run `bee help pack`.

### bee api

To create a Beego API application:

```bash
$ bee api my-api
______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v1.5.0
2016/08/22 15:14:10 [INFO] Creating API...
        create   C:\Users\beeuser\go\src\github.com\user\my-api
        create   C:\Users\beeuser\go\src\github.com\user\my-api\conf
        create   C:\Users\beeuser\go\src\github.com\user\my-api\controllers
        create   C:\Users\beeuser\go\src\github.com\user\my-api\tests
        create   C:\Users\beeuser\go\src\github.com\user\my-api\conf\app.conf
        create   C:\Users\beeuser\go\src\github.com\user\my-api\models
        create   C:\Users\beeuser\go\src\github.com\user\my-api\routers\
        create   C:\Users\beeuser\go\src\github.com\user\my-api\controllers\object.go
        create   C:\Users\beeuser\go\src\github.com\user\my-api\controllers\user.go
        create   C:\Users\beeuser\go\src\github.com\user\my-api\tests\default_test.go
        create   C:\Users\beeuser\go\src\github.com\user\my-api\routers\router.go
        create   C:\Users\beeuser\go\src\github.com\user\my-api\models\object.go
        create   C:\Users\beeuser\go\src\github.com\user\my-api\models\user.go
        create   C:\Users\beeuser\go\src\github.com\user\my-api\main.go
2016/08/22 15:14:10 [SUCC] New API successfully created!
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
\____/  \___| \___| v1.5.0
2016/08/22 16:09:13 [INFO] Creating Hprose application...
        create   C:\Users\beeuser\go\src\github.com\user\my-rpc-app
        create   C:\Users\beeuser\go\src\github.com\user\my-rpc-app\conf
        create   C:\Users\beeuser\go\src\github.com\user\my-rpc-app\conf\app.conf
        create   C:\Users\beeuser\go\src\github.com\user\my-rpc-app\models
        create   C:\Users\beeuser\go\src\github.com\user\my-rpc-app\models\object.go
        create   C:\Users\beeuser\go\src\github.com\user\my-rpc-app\models\user.go
        create   C:\Users\beeuser\go\src\github.com\user\my-rpc-app\main.go
2016/08/22 16:09:13 [SUCC] New Hprose application successfully created!
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
\____/  \___| \___| v1.5.0
2016/08/22 16:37:24 [INFO] Detected bee.json
2016/08/22 16:37:24 [INFO] Packaging directory(static/js)
2016/08/22 16:37:24 [INFO] Packaging directory(static/css)
2016/08/22 16:37:24 [SUCC] Baled resources successfully!
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
\____/  \___| \___| v1.5.0
2016/08/22 16:55:30 [INFO] Using 'Hello' as controller name
2016/08/22 16:55:30 [INFO] Using 'controllers' as package name
        create   C:\Users\beeuser\go\src\github.com\user\my-web-app/controllers/hello.go
2016/08/22 16:55:30 [SUCC] Controller successfully generated!                                  
```

For more information on the usage, run `bee help generate`.

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
usage: bee run [appname] [watchall] [-main=*.go] [-downdoc=true]  [-gendoc=true] [-vendor=true] [-e=folderToExclude]  [-tags=goBuildTags]

Run command will supervise the file system of the beego project using inotify,
it will recompile and restart the app after any modifications.
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
go to greater lengths in your message's body. A good example can be found in [Angular commit message format](https://github.com/angular/angular.js/blob/master/CONTRIBUTING.md#commit-message-format).

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
Copyright 2016 bee authors

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
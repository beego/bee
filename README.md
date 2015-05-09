bee
===

[![Build Status](https://drone.io/github.com/beego/bee/status.png)](https://drone.io/github.com/beego/bee/latest)

Bee is a command line tool facilitating development with beego framework.

## Requirements

- Go version >= 1.1.


## Installation

Begin by installing `bee` using `go get` command.

```bash
go get github.com/beego/bee
```

Then you can add `bee` binary to PATH environment variable in your `~/.bashrc` or `~/.bash_profile` file:

```bash
export PATH=$PATH:<your_main_gopath>/bin/bee
```

> If you already have `bee` installed, updating `bee` is simple:

```bash
go get -u github.com/beego/bee
```

## Basic commands

Bee provides a variety of commands which can be helpful at various stage of development. The top level commands include: 
```base
  new         create an application base on beego framework
  run         run the app which can hot compile
  pack        compress an beego project
  api         create an api application base on beego framework
  bale        packs non-Go files to Go source files
  version     show the bee & beego version
  generate    source code generator
  migrate     run database migrations
  hprose      create an rpc application use hprose base on beego framework
```
## bee version

The first command is the easiest: displaying which version of `bee`, `beego` and `go` is installed on your machine:

```bash
$ bee version
bee   :1.2.2
beego :1.4.0
Go    :go version go1.2.1 linux/amd64
``` 

## bee new

Creating a new beego web application is no big deal, too.

```bash
$ bee new myapp
[INFO] Creating application...
/home/zheng/gopath/src/myapp/
/home/zheng/gopath/src/myapp/conf/
/home/zheng/gopath/src/myapp/controllers/
/home/zheng/gopath/src/myapp/models/
/home/zheng/gopath/src/myapp/routers/
/home/zheng/gopath/src/myapp/tests/
/home/zheng/gopath/src/myapp/static/
/home/zheng/gopath/src/myapp/static/js/
/home/zheng/gopath/src/myapp/static/css/
/home/zheng/gopath/src/myapp/static/img/
/home/zheng/gopath/src/myapp/views/
/home/zheng/gopath/src/myapp/conf/app.conf
/home/zheng/gopath/src/myapp/controllers/default.go
/home/zheng/gopath/src/myapp/views/index.tpl
/home/zheng/gopath/src/myapp/routers/router.go
/home/zheng/gopath/src/myapp/tests/default_test.go
/home/zheng/gopath/src/myapp/main.go
2014/08/29 15:45:47 [SUCC] New application successfully created!
```

## bee run

To run the application we just created, navigate to the application folder and execute `bee run`.

```bash
$ cd myapp
$ bee run
```

## bee pack

```bash
usage: bee pack

compress an beego project

-p            app path. default is current path
-b            build specify platform app. default true
-ba           additional args of go build
-be=[]        additional ENV Variables of go build. eg: GOARCH=arm
-o            compressed file output dir. default use current path
-f=""         format. [ tar.gz / zip ]. default tar.gz
-exp=""       relpath exclude prefix. default: .
-exs=""       relpath exclude suffix. default: .go:.DS_Store:.tmp
              all path use : as separator
-exr=[]       file/directory name exclude by Regexp. default: ^.
-fs=false     follow symlink. default false
-ss=false     skip symlink. default false
              default embed symlink into compressed file
-v=false      verbose
```

## bee api

```bash
usage: bee api [appname]

create an api application base on beego framework

bee api [appname] [-tables=""] [-driver="mysql"] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    -tables: a list of table names separated by ',', default is empty, indicating all tables
    -driver: [mysql | postgres | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is ''
             e.g. for mysql:    root:@tcp(127.0.0.1:3306)/test
             e.g. for postgres: postgres://postgres:postgres@127.0.0.1:5432/postgres
  
if conn is empty will create a example api application. otherwise generate api application based on an existing database.

In the current path, will create a folder named [appname]

In the appname folder has the follow struct:

  ├── conf
  │   └── app.conf
  ├── controllers
  │   └── object.go
  │   └── user.go
  ├── routers
  │   └── router.go
  ├── tests
  │   └── default_test.go
  ├── main.go
  └── models
      └── object.go
      └── user.go
```

## bee hprose

```bash
usage: bee hprose [appname]

create an rpc application use hprose base on beego framework

bee hprose [appname] [-tables=""] [-driver=mysql] [-conn=root:@tcp(127.0.0.1:3306)/test]
    -tables: a list of table names separated by ',', default is empty, indicating all tables
    -driver: [mysql | postgres | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is ''
             e.g. for mysql:    root:@tcp(127.0.0.1:3306)/test
             e.g. for postgres: postgres://postgres:postgres@127.0.0.1:5432/postgres
  
if conn is empty will create a example rpc application. otherwise generate rpc application use hprose based on an existing database.

In the current path, will create a folder named [appname]

In the appname folder has the follow struct:

  ├── conf
  │   └── app.conf
  ├── main.go
  └── models
      └── object.go
      └── user.go

```

## bee bale

```bash
usage: bee bale

bale packs non-Go files to Go source files and

auto-generate unpack function to main package then run it

during the runtime.

This is mainly used for zealots who are requiring 100% Go code.
```

## bee migrate

```bash
usage: bee migrate [Command]

bee migrate [-driver="mysql"] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    run all outstanding migrations
    -driver: [mysql | postgresql | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test

bee migrate rollback [-driver="mysql"] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    rollback the last migration operation
    -driver: [mysql | postgresql | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test

bee migrate reset [-driver="mysql"] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    rollback all migrations
    -driver: [mysql | postgresql | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test

bee migrate refresh [-driver="mysql"] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    rollback all migrations and run them all again
    -driver: [mysql | postgresql | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test

```

## bee generate

Bee also comes with a souce code generator which speeds up the development.

```bash
usage: bee generate [Command]

bee generate scaffold [scaffoldname] [-fields=""] [-driver="mysql"] [-conn="root:@tcp(127.0.0.1:3306)/test"]
    The generate scaffold command will do a number of things for you.
    -fields: a list of table fields. Format: field:type, ...
    -driver: [mysql | postgres | sqlite], the default is mysql
    -conn:   the connection string used by the driver, the default is root:@tcp(127.0.0.1:3306)/test
    example: bee generate scaffold post -fields="title:string,body:text"

bee generate model [modelname] [-fields=""]
    generate RESTFul model based on fields
    -fields: a list of table fields. Format: field:type, ...

bee generate controller [controllerfile]
    generate RESTFul controllers             

bee generate view [viewpath]
    generate CRUD view in viewpath

bee generate migration [migrationfile] [-fields=""]
    generate migration file for making database schema update
    -fields: a list of table fields. Format: field:type, ...
  
bee generate docs
    generate swagger doc file

bee generate test [routerfile]
    generate testcase

bee generate appcode [-tables=""] [-driver="mysql"] [-conn="root:@tcp(127.0.0.1:3306)/test"] [-level=3]
    generate appcode based on an existing database
    -tables: a list of table names separated by ',', default is empty, indicating all tables
    -driver: [mysql | postgres | sqlite], the default is mysql
    -conn:   the connection string used by the driver.
             default for mysql:    root:@tcp(127.0.0.1:3306)/test
             default for postgres: postgres://postgres:postgres@127.0.0.1:5432/postgres
    -level:  [1 | 2 | 3], 1 = models; 2 = models,controllers; 3 = models,controllers,router

```


## Shortcuts

Because you'll likely type these generator commands over and over, it makes sense to create aliases.

```bash
# Generator Stuff
alias g:a="bee generate appcode"
alias g:m="bee generate model"
alias g:c="bee generate controller"
alias g:v="bee generate view"
alias g:mi="bee generate migration"
```

These can be stored in, for example, your `~/.bash_profile` or `~/.bashrc` files.

## Help

If you happend to forget the usage of a command, you can always find the usage information by `bee help <command>`.

For instance, to get more information about the `run` command:

```bash
$ bee help run
usage: bee run [appname] [watchall] [-main=*.go] [-downdoc=true]  [-gendoc=true]

start the appname throw exec.Command

then start a inotify watch for current dir
                    
when the file has changed bee will auto go build and restart the app

  file changed
       |
  check if it's go file
       |
     yes     no
      |       |
 go build    do nothing
     |
 restart app

```

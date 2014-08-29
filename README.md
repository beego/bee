bee
===

[![Build Status](https://drone.io/github.com/beego/bee/status.png)](https://drone.io/github.com/beego/bee/latest)

Bee is a command line tool facilitating development with beego framework.

## Requirements

- Go version >= 1.1.

## Installation

Begin by installing `bee` using `go get` command.

	go get github.com/beego/bee

Then you can add `bee` binary to PATH:

	export PATH=$PATH:<your_main_gopath>/bin/bee

> If you already have `bee` installed, updating `bee` is simple:

	go get -u github.com/beego/bee

## Basic commands

Bee provides a variety of commands which can be helpful at various stage of development. The top level commands include: 

	new         create an application base on beego framework
	run         run the app which can hot compile
	pack        compress an beego project
	api         create an api application base on beego framework
	bale        packs non-Go files to Go source files
	version     show the bee & beego version
	generate    source code generator
	migrate     run database migrations

### Shortcuts

Because you'll likely type these commands over and over, it makes sense to create aliases.

```bash
# Generator Stuff
alias g:a="bee generate appcode"
alias g:m="bee generate model"
alias g:c="bee generate controller"
alias g:v="bee generate view"
alias g:mig="bee generate migration"
```

These can be stored in, for example, your `~/.bash_profile` or `~/.bashrc` files.

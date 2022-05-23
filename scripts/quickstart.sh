#!/bin/sh

export GO111MODULE=on
go get github.com/beego/bee/v2@latest
bee new hello
cd hello
bee run

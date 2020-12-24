#!/bin/sh

export GO111MODULE=on
go get github.com/beego/bee/v2@v2.0.2
bee new hello
cd hello
bee run

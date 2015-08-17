#!/bin/bash

export GOPATH=/go
export PATH=$PATH:/go/bin
go get github.com/Sirupsen/logrus
go run main.go -nme=$1 -nmeContainerId=$2


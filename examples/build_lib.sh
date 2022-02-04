#!/bin/bash


# go build -o libtest.so -buildmode=c-archive *.go
go build -o atest.a -buildmode=c-archive *.go

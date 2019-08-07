@echo off

set GOOS=windows
set GOARCH=amd64
go build

set GOOS=linux
go build -o watercooler-chat-linux

set GOOS=darwin
go build -o watercooler-chat-mac
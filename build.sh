#!/usr/bin/env bash

env GOOS=darwin GOARCH=amd64 go build -o watercooler-chat-mac
env GOOS=linux GOARCH=amd64 go build -o watercooler-chat-linux
env GOOS=windows GOARCH=amd64 go build
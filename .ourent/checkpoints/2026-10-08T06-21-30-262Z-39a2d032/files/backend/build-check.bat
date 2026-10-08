@echo off
cd /d %~dp0
set GOFLAGS=-mod=mod
go build ./... 1>> build-out.txt 2>&1
echo EXITCODE=%ERRORLEVEL%>> build-out.txt

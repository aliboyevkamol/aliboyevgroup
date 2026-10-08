@echo off
cd /d %~dp0
go build ./... > build-all.txt 2>&1
echo EXITCODE=%ERRORLEVEL%>> build-all.txt
echo DONE>> build-all.txt

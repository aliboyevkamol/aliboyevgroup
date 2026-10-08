@echo off
cd /d %~dp0
go build ./config/ > cfg.txt 2>&1
echo EXITCODE=%ERRORLEVEL% >> cfg.txt

@echo off
cd /d "%~dp0"
docker compose build backend > "%~dp0build-bg.log" 2>&1
echo DONE_%ERRORLEVEL% >> "%~dp0build-bg.log"

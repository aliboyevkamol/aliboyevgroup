@echo off
cd /d "%~dp0"
docker compose build backend > "%~dp0build-bg.log" 2>&1
echo EXIT_%ERRORLEVEL%>> "%~dp0build-bg.log"

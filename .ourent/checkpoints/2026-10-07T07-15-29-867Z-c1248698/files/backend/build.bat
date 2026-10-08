@echo off
cd /d "%~dp0"
docker compose up -d --build > build.log 2>&1
echo DONE_EXIT_%ERRORLEVEL% >> build.log

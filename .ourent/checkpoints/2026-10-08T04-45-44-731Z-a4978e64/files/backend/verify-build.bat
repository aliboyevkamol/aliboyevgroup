@echo off
cd /d "%~dp0"
docker compose -f docker-compose.verify.yml up -d --build > verify-build.log 2>&1
echo DONE_EXIT_%ERRORLEVEL% >> verify-build.log

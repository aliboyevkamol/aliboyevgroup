@echo off
cd /d "%~dp0"
call npm install playwright-core --no-save --no-audit --no-fund > npm-install.log 2>&1
echo DONE_EXIT_%ERRORLEVEL% >> npm-install.log

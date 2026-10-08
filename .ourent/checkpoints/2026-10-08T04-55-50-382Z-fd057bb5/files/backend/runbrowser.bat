@echo off
cd /d "%~dp0"
start "browsertest" /b cmd /c "node browser-test.cjs > browser-test-4.log 2>&1"

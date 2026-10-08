@echo off
cd /d %~dp0
node browser-test.cjs > browser-test-3.log 2>&1

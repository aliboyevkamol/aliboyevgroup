@echo off
cd /d %~dp0
node e2e-test.mjs > e2e-run2.log 2>&1

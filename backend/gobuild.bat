@echo off
cd /d "%~dp0"
docker rm -f kp-gobuild >nul 2>&1
docker run --rm -d --name kp-gobuild -v "%CD%:/src" -w /src -e CGO_ENABLED=0 -e GOCACHE=/tmp/gocache golang:1.24-alpine sh -c "go build -o /src/server.check ./cmd/server > /src/gobuild.log 2>&1; echo EXIT_$? >> /src/gobuild.log"
echo LAUNCH_ERR_%ERRORLEVEL%

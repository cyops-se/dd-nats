echo off
for /f "delims=" %%a in ('git rev-list --abbrev-commit -1 HEAD') do @set GIT_COMMIT=%%a
for /f "delims=" %%a in ('git describe --tags --dirty') do @set GIT_VERSION=%%a
set goarch=386
set cgo_enabled=1
go build -ldflags "-X main.GitCommit=%GIT_COMMIT% -X main.GitVersion=%GIT_VERSION%"
@echo off
setlocal
set CWD=%CD%
cd /d %~dp0..
set GO111MODULE=on
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -o aiw.exe main.go
if exist aiw.exe (
@REM    aiw.exe
    del /q  aiw.exe
)

cd /d %CWD%
endlocal

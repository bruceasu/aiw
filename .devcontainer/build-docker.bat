@echo off
setlocal
SET CWD=%CD%
set "SCRIPT_DIR=%~dp0"
cd /d "%SCRIPT_DIR%\.devcontainer"
dir
docker build -t victor/java25 .

cd /d "%CWD%"
endlocal

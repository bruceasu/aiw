@echo off
setlocal
set CWD=%CD%
set SCRIPT_DIR=%~dp0
cd /d %SCRIPT_DIR%
set SCRIPT_DIR=%CD%
set COMPOSE_FILE=docker-compose.yaml
docker compose -f "%COMPOSE_FILE%" run --rm app %*
cd /d %CWD%
endlocal

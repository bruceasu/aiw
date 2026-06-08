@echo off

set CWD=%CD%
cd /d "%~dp0"
setlocal
SET INSTALL_DIR=c:\green\aiw
if not exist "%INSTALL_DIR%" (
    mkdir "%INSTALL_DIR%"
)
if not exist "%INSTALL_DIR%\plugins" (
    mkdir "%INSTALL_DIR%\plugins"
)

if not exist "%INSTALL_DIR%\docs\usage" (
    mkdir "%INSTALL_DIR%\docs\usage"
)

@if "%~1"=="" (
    #echo Usage: %~nx0 [plugin-name] [, plugin-name2, ...]
    call cp-mirror.bat plugins  "%INSTALL_DIR%\plugins" >nul 2>&1
    call cp-mirror.bat docs\usage  "%INSTALL_DIR%\docs\usage" >nul 2>&1
    exit /b 0
)


for %%a in (%*) do (
    call cp-mirror.bat "plugins\%%a" "%INSTALL_DIR%\plugins\%%a" >nul 2>&1
    call cp-mirror.bat "docs\usage\%%a" "%INSTALL_DIR%\docs\usage\%%a" >nul 2>&1
)

endlocal
cd /d "%CWD%"
echo Done.
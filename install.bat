@echo off

set CWD=%CD%
cd /d "%~dp0"
setlocal
if not exist "bin" (
    echo Build directory not found. Please run build.ps1 first.
    exit /b 1
)
if not exist "c:\green\aiw" (
    mkdir c:\green\aiw
)

xcopy /D /Y bin\aiw-linux-amd64 c:\green\aiw\aiw >nul
xcopy /D /Y bin\aiw-windows-amd64.exe c:\green\aiw\aiw.exe >nul

call cp-mirror.bat plugins  c:\green\aiw\plugins
call cp-mirror.bat docs\usage  c:\green\aiw\docs\usage
endlocal
cd /d "%CWD%"
echo Installation complete. aiw is now available in c:\green\aiw.
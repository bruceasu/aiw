@echo off

set CWD=%CD%
cd /d "%~dp0"
setlocal
set INSTALL_DIR=c:\green\aiw

if not exist "bin" (
    echo Build directory not found. Please run build.ps1 first.
    exit /b 1
)

if not exist "%INSTALL_DIR%" (
    mkdir "%INSTALL_DIR%"
)


gbuild windows

xcopy /D /Y bin\aiw-windows-amd64.exe %INSTALL_DIR%\aiw.exe >nul
if "%~1"=="linux" (
    gbuild linux
    xcopy /D /Y bin\aiw-linux-amd64 %INSTALL_DIR%\aiw >nul
) else if "%~1"=="plugins" (
   call cp-mirror.bat plugins  %INSTALL_DIR%\plugins
) else if "%~1"=="docs" (
    call cp-mirror.bat docs\usage  %INSTALL_DIR%\docs\usage
)

echo Installation complete. aiw is now available in %INSTALL_DIR%.

endlocal
cd /d "%CWD%"

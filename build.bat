@echo off

set CWD=%CD%
cd /d "%~dp0"
setlocal
set INSTALL_DIR=c:\green\aiw

if not exist "bin" (
    md bin
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
    call cp-mirror.bat plugins  %INSTALL_DIR%\plugins || exit /b 1
    call cp-mirror.bat skills  %INSTALL_DIR%\skills || exit /b 1
) else if "%~1"=="docs" (
    call cp-mirror.bat docs\usage  %INSTALL_DIR%\docs\usage
) else if "%~1"=="skills" (
    call cp-mirror.bat skills  %INSTALL_DIR%\skills || exit /b 1
)

echo Installation complete. aiw is now available in %INSTALL_DIR%.

endlocal
cd /d "%CWD%"

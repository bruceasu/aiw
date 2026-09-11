@echo off

setlocal EnableExtensions
set "CWD=%CD%"
cd /d "%~dp0"
set INSTALL_DIR=c:\green\aiw

if /i "%~1"=="-h" goto :help
if /i "%~1"=="--help" goto :help
if /i "%~1"=="help" goto :help
if /i "%~1"=="/h" goto :help
if /i "%~1"=="/?" goto :help

if not exist "bin" (
    md bin
)

if not exist "%INSTALL_DIR%" (
    mkdir "%INSTALL_DIR%"
)

if "%~1"=="" (
    call :build_windows
) else (
    for %%A in (%*) do (
        call :run_action %%~A
        if errorlevel 1 (
            set "RESULT=1"
            goto :finish
        )
    )
)
if errorlevel 1 (
    set "RESULT=1"
) else (
    set "RESULT=0"
)
:finish
endlocal & cd /d "%CWD%" & exit /b %RESULT%

 :run_action
if /i "%~1"=="windows" goto :build_windows
if /i "%~1"=="linux" goto :build_linux
if /i "%~1"=="plugins" goto :build_plugins
if /i "%~1"=="docs" goto :build_docs
if /i "%~1"=="skills" goto :build_skills
if /i "%~1"=="all" call :build_windows & call :build_linux & call :build_plugins & call :build_docs & call :build_skills & exit /b 0
echo Error: Unknown build action: %~1
echo Run build.bat --help for available actions.
exit /b 1

 :build_windows
del /s/q bin\aiw-windows-amd64.exe 2>nul
gbuild windows
if not exist "bin\aiw-windows-amd64.exe" (
    echo Error: Windows build failed.
    exit /b 1
)
xcopy /D /Y bin\aiw-windows-amd64.exe %INSTALL_DIR%\aiw.exe >nul
echo Installation complete. aiw is now available in %INSTALL_DIR%.
exit /b 0

 :build_linux
del /s/q bin\aiw-linux-amd64 2>nul
gbuild linux
if not exist "bin\aiw-linux-amd64" (
    echo Error: Linux build failed.
    exit /b 1
)
xcopy /D /Y bin\aiw-linux-amd64 %INSTALL_DIR%\aiw >nul
echo Installation complete. aiw is now available in %INSTALL_DIR%.
exit /b 0

 :build_plugins
call cp-mirror.bat plugins %INSTALL_DIR%\plugins || exit /b 1
call cp-mirror.bat skills %INSTALL_DIR%\skills || exit /b 1
exit /b 0

 :build_docs
call cp-mirror.bat docs\usage %INSTALL_DIR%\docs\usage || exit /b 1
exit /b 0

 :build_skills
call cp-mirror.bat skills %INSTALL_DIR%\skills || exit /b 1
exit /b 0

 :help
echo Usage: build.bat [windows] [linux] [plugins] [docs] [skills]
echo.
echo Actions can be combined and run in the specified order.
echo With no arguments, only the Windows build is performed.
echo.
echo   windows  Build and install the Windows executable.
echo   linux    Build and install the Linux executable.
echo   plugins  Copy plugins and skills to the install directory.
echo   docs     Copy usage documentation to the install directory.
echo   skills   Copy skills to the install directory.
echo.
echo Help: -h, --help, help, /h, /?
exit /b 0

@echo off

setlocal EnableExtensions
set "CWD=%CD%"
cd /d "%~dp0"
set INSTALL_DIR=c:\green\aiw
if not defined AIW_VERSION set "AIW_VERSION=dev"
set "AIW_GOFLAGS=-ldflags=-X=aiw/internal/version.Version=%AIW_VERSION%"

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
if /i "%~1"=="wf" goto :build_workflow
if /i "%~1"=="req" goto :build_req
if /i "%~1"=="cz" goto :build_cz
if /i "%~1"=="plugins" goto :build_plugins
if /i "%~1"=="docs" goto :build_docs
if /i "%~1"=="skills" goto :build_skills
if /i "%~1"=="bin" (
    call :build_windows || exit /b 1
    call :build_linux || exit /b 1
    exit /b 0
)
if /i "%~1"=="all" (
    call :build_windows || exit /b 1
    call :build_linux || exit /b 1
    call :build_plugins || exit /b 1
    call :build_docs || exit /b 1
    call :build_skills || exit /b 1
    exit /b 0
)
echo Error: Unknown build action: %~1
echo Run build.bat --help for available actions.
exit /b 1

 :build_windows
del /s/q bin\aiw-windows-amd64.exe 2>nul
set "GOFLAGS=%AIW_GOFLAGS%"
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
set "GOFLAGS=%AIW_GOFLAGS%"
gbuild linux
if not exist "bin\aiw-linux-amd64" (
    echo Error: Linux build failed.
    exit /b 1
)
xcopy /D /Y bin\aiw-linux-amd64 %INSTALL_DIR%\aiw >nul
echo Installation complete. aiw is now available in %INSTALL_DIR%.
exit /b 0

 :build_workflow
if not exist "plugins\aiw-wf" md "plugins\aiw-wf"
del /s/q plugins\aiw-wf\aiw-wf.exe 2>nul
del /s/q plugins\aiw-wf\aiw-wf 2>nul
set "GOFLAGS="

set "GOOS=windows"
set "GOARCH=amd64"
go build -trimpath  -ldflags="-s -w -X aiw/internal/version.Version=%AIW_VERSION%" -o "plugins\aiw-wf\aiw-wf.exe" ./cmd/aiw-wf
if errorlevel 1 (
    set "GOOS="
    set "GOARCH="
    echo Error: Windows workflow build failed.
    exit /b 1
)

set "GOOS=linux"
set "GOARCH=amd64"
go build -trimpath  -ldflags="-s -w -X aiw/internal/version.Version=%AIW_VERSION%" -o "plugins\aiw-wf\aiw-wf" ./cmd/aiw-wf
if errorlevel 1 (
    set "GOOS="
    set "GOARCH="
    echo Error: Linux workflow build failed.
    exit /b 1
)

set "GOOS="
set "GOARCH="
echo Workflow plugin binaries built in plugins\aiw-wf.
exit /b 0

 :build_req
if not exist "plugins\aiw-req" md "plugins\aiw-req"
del /s/q plugins\aiw-req\aiw-req.exe 2>nul
del /s/q plugins\aiw-req\aiw-req 2>nul
set "GOFLAGS="

set "GOOS=windows"
set "GOARCH=amd64"
go build -trimpath -ldflags="-s -w -X aiw/internal/version.Version=%AIW_VERSION%" -o "plugins\aiw-req\aiw-req.exe" ./cmd/aiw-req
if errorlevel 1 (
    set "GOOS="
    set "GOARCH="
    echo Error: Windows req build failed.
    exit /b 1
)

set "GOOS=linux"
set "GOARCH=amd64"
go build -trimpath -ldflags="-s -w -X aiw/internal/version.Version=%AIW_VERSION%" -o "plugins\aiw-req\aiw-req" ./cmd/aiw-req
if errorlevel 1 (
    set "GOOS="
    set "GOARCH="
    echo Error: Linux req build failed.
    exit /b 1
)

set "GOOS="
set "GOARCH="
echo Req plugin binaries built in plugins\aiw-req.
exit /b 0

 :build_cz
pushd plugins\aiw-cz
call npm install || (popd & exit /b 1)
call npm run build || (popd & exit /b 1)
popd
echo TypeScript cz plugin built in plugins\aiw-cz\dist.
exit /b 0

 :build_plugins
call :build_workflow || exit /b 1
call :build_req || exit /b 1
call :build_cz || exit /b 1
call :install_entries plugins "%INSTALL_DIR%\plugins" || exit /b 1
exit /b 0

 :build_docs
call cp-mirror.bat docs\usage %INSTALL_DIR%\docs\usage || exit /b 1
exit /b 0

 :build_skills
call :build_workflow || exit /b 1
call :install_entries skills "%INSTALL_DIR%\skills" || exit /b 1
exit /b 0

 :install_entries
if not exist "%~2\" mkdir "%~2" || exit /b 1
for %%E in ("%~1\*") do (
    if exist "%%~fE\" (
        xcopy /E /I /Y /H "%%~fE" "%~2\%%~nxE\" >nul || exit /b 1
    ) else (
        copy /Y "%%~fE" "%~2\" >nul || exit /b 1
    )
)
exit /b 0

 :help
echo Usage: build.bat [windows] [linux] [wf] [req] [cz] [plugins] [docs] [skills]
echo.
echo Actions can be combined and run in the specified order.
echo With no arguments, only the Windows build is performed.
echo.
echo   windows  Build and install the Windows executable.
echo   linux    Build and install the Linux executable.
echo   wf       Build Windows and Linux workflow plugin binaries.
echo   req      Build Windows and Linux req plugin binaries.
echo   cz       Install dependencies and compile the TypeScript cz plugin.
echo   plugins  Build plugin binaries and install plugins individually.
echo   docs     Copy usage documentation to the install directory.
echo   skills   Build workflow binaries and install skills individually.
echo.
echo Help: -h, --help, help, /h, /?
exit /b 0

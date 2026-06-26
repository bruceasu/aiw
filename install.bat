@echo off
setlocal
if not exist "bin" (
    echo Build directory not found. Please run build.ps1 first.
    exit /b 1
)
if not exist "c:\green\aiw" (
    mkdir c:\green\aiw
)
call cpFile.bat bin\aiw-linux-amd64 c:\green\aiw\aiw
call cpFile.bat bin\aiw-windows-amd64.exe c:\green\aiw\aiw.exe

endlocal

@echo off
setlocal EnableExtensions
aiw gw stop %*
exit /b %errorlevel%

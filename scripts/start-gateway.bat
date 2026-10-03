@echo off
setlocal EnableExtensions
aiw gw start %*
exit /b %errorlevel%

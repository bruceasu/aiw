@echo off
icacls . /grant "%COMPUTERNAME%\CodexSandboxUsers:(OI)(CI)(RX)" /T

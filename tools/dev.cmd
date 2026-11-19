@echo off
REM Windows dev launcher: bypass execution policy for tools\dev.ps1.
REM Usage:
REM   tools\dev.cmd
REM   tools\dev.cmd start -Docker
REM   tools\dev.cmd stop
REM   tools\dev.cmd status
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0dev.ps1" %*
exit /b %ERRORLEVEL%

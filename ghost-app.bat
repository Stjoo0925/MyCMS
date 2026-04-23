@echo off
setlocal

title mycms ghost app
echo Ghost app is running. Press Ctrl+C to stop.

:idle
timeout /t 60 /nobreak >nul
goto idle

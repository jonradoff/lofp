@echo off
setlocal enabledelayedexpansion

echo Starting Legends of Future Past...

:: Get the directory where this .bat file lives
set ROOT_DIR=%~dp0
:: Remove trailing backslash
if "%ROOT_DIR:~-1%"=="\" set ROOT_DIR=%ROOT_DIR:~0,-1%

:: Load .env file if present
if exist "%ROOT_DIR%\.env" (
    for /f "usebackq tokens=1,* delims==" %%A in ("%ROOT_DIR%\.env") do (
        set "%%A=%%B"
    )
)

:: Kill anything already on ports 4993 and 4992
for /f "tokens=5" %%P in ('netstat -aon ^| findstr ":4993 " ^| findstr "LISTENING"') do (
    taskkill /PID %%P /F >nul 2>&1
)
for /f "tokens=5" %%P in ('netstat -aon ^| findstr ":4992 " ^| findstr "LISTENING"') do (
    taskkill /PID %%P /F >nul 2>&1
)

timeout /t 1 /nobreak >nul

:: Start backend in a new window
start "LoFP Backend" /D "%ROOT_DIR%\engine" cmd /c "go run cmd/lofp/main.go"

:: Start frontend in a new window
start "LoFP Frontend" /D "%ROOT_DIR%\frontend" cmd /c "npx vite --port 4992 --host"

echo Backend starting on port 4993
echo Frontend starting on port 4992
echo.
echo Open http://localhost:4992 to play!
echo.
echo Press Ctrl+C or close this window to stop.
echo Note: You will need to close the Backend and Frontend windows separately.
echo.
pause
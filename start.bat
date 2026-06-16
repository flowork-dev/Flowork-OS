@echo off
REM ============================================================================
REM Flowork - one-click start (Windows). Launches the whole stack:
REM   Router (:2402)  +  Agent (:1987)
REM Each opens in its own window and builds on first run (needs Go 1.25+).
REM The agent boots its schedule + trigger engine automatically.
REM Open the panel:  http://127.0.0.1:1987     Stop: stop.bat
REM ============================================================================
cd /d "%~dp0"
echo Flowork - starting the full stack...
echo.

REM --- Auto-update: pull latest if this is a git clone (opt out: set FLOWORK_NO_UPDATE=1). ---
REM --ff-only is safe: it never merges/clobbers; it just fast-forwards or stops.
if not "%FLOWORK_NO_UPDATE%"=="1" if exist ".git" (
    where git >nul 2>&1 && (
        echo - checking for updates...
        git pull --ff-only
    )
)
echo.

if exist "%~dp0router\start.bat" (
    echo - Router  ^(:2402^)  in its own window...
    start "Flowork Router" /d "%~dp0router" cmd /k start.bat
) else (
    echo   router\start.bat not found - skipping router
)

if exist "%~dp0agent\start.bat" (
    echo - Agent   ^(:1987^)  in its own window...
    start "Flowork Agent" /d "%~dp0agent" cmd /k start.bat
) else (
    echo   agent\start.bat not found - cannot start Flowork
    pause
    exit /b 1
)

REM --- Semantic RAG index: auto-build/resume on launch (skip if already built). ---
REM Pipeline is bash + Go tooling; on Windows it runs via bash (WSL / Git Bash).
REM The launcher self-guards: if the index is already built it just exits fast.
REM A native-Windows build path is a tracked follow-up. Opt-out: set FLOWORK_NO_RAG=1.
if not "%FLOWORK_NO_RAG%"=="1" (
    where bash >nul 2>&1 && (
        echo - Semantic RAG index: auto-build/resume in background ^(via bash^)...
        start "Flowork RAG" /min bash -c "nohup bash router/scripts/rag-autostart.sh >> router/brain/_rag/rag-pipeline.log 2>&1 &"
    ) || echo - Semantic RAG: bash not found - skipped ^(needs WSL/Git Bash; native-Windows build pending^).
)

echo.
echo Flowork is starting:
echo    Control panel ^>  http://127.0.0.1:1987
echo    LLM router    ^>  http://127.0.0.1:2402/v1
echo Schedules ^& triggers run automatically inside the agent.
echo (First run compiles the binaries - give each window a minute.)
echo Stop everything: stop.bat
pause

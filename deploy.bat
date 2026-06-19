@echo off
echo Building vidc...
go build -o bin\vidc.exe .\cmd\vidc
if errorlevel 1 (
    echo Build failed.
    exit /b 1
)

echo.

if not exist "C:\Tools" (
    mkdir "C:\Tools"
    echo Created C:\Tools.
)

copy /Y "bin\vidc.exe" "C:\Tools\vidc.exe"
if errorlevel 1 (
    echo Deploy failed.
    exit /b 1
)

echo Deployed vidc.exe to C:\Tools.

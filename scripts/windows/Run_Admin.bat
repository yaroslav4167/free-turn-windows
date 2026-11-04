@echo off
set "FREETURN_EXE=%~dp0freeturn-desktop.exe"
powershell.exe -NoLogo -NoProfile -Command "Start-Process -FilePath $env:FREETURN_EXE -Verb RunAs"

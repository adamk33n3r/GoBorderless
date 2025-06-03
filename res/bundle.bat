@echo off
cd %~dp0
fyne bundle -pkg res -prefix Res -o bundled.go icon.png

@echo off
cd %~dp0
fyne bundle -pkg res -prefix Res -o bundled.go icon.png
fyne bundle -a -pkg res -prefix Res -o bundled.go icon.ico

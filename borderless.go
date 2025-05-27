package main

import (
	"fmt"

	"github.com/lxn/win"
)

func makeBorderless(window Window, appSetting AppSetting) {
	// fmt.Println("Making window borderless:", window.title, window.exePath)
	style := getWindowStyle(window.hwnd)
	// Remove the border and title bar
	setWindowStyle(window.hwnd, style & ^win.WS_CAPTION & ^win.WS_THICKFRAME)
	monitor := monitors[appSetting.Monitor-1]
	setWindowPos(window.hwnd, appSetting.OffsetX+monitor.left, appSetting.OffsetY+monitor.top, appSetting.Width, appSetting.Height)
}

func restoreWindow(window Window, appSetting AppSetting) {
	fmt.Println("Restoring window:", window.title, window.exePath)
	style := getWindowStyle(window.hwnd)
	// Restore the border and title bar
	setWindowStyle(window.hwnd, style|win.WS_OVERLAPPEDWINDOW)
	setWindowPos(window.hwnd, 0, 0, appSetting.PreWidth, appSetting.PreHeight)
}

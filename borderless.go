package main

import (
	"fmt"

	"github.com/lxn/win"
)

func isBorderless(window Window) bool {
	style := getWindowStyle(window.hwnd)
	return !(style&win.WS_CAPTION > 0 &&
		((style&win.WS_BORDER) > 0 || (style&win.WS_THICKFRAME) > 0))
}

func makeBorderless(window Window, appSetting AppSetting) {
	// fmt.Println("Making window borderless:", window.title, window.exePath)
	style := getWindowStyle(window.hwnd)
	// Remove the border and title bar
	setWindowStyle(window.hwnd, style & ^win.WS_CAPTION & ^win.WS_THICKFRAME)
	monitor := monitors[appSetting.Monitor-1]
	setWindowPos(window.hwnd, appSetting.OffsetX+monitor.left, appSetting.OffsetY+monitor.top, appSetting.Width, appSetting.Height)
}

/**
 * Only restores the window if it's borderless
 */
func restoreWindow(window Window, appSetting AppSetting) {
	if !isBorderless(window) {
		return
	}
	fmt.Println("Restoring window:", window.title, window.exePath)
	style := getWindowStyle(window.hwnd)
	// Restore the border and title bar
	setWindowStyle(window.hwnd, style|win.WS_OVERLAPPEDWINDOW)
	setWindowPos(window.hwnd, appSetting.PreOffsetX, appSetting.PreOffsetY, appSetting.PreWidth, appSetting.PreHeight)
}

package main

import (
	"fmt"

	"github.com/lxn/win"
)

/**
 * A window counts as bordered when it has a caption plus either a border or a
 * resizable frame. Split out from isBorderless so the window-select filter can
 * share the exact same rule (and so it can be unit tested without an HWND).
 */
func isBorderlessStyle(style int32) bool {
	return !(style&win.WS_CAPTION > 0 &&
		((style&win.WS_BORDER) > 0 || (style&win.WS_THICKFRAME) > 0))
}

func isBorderless(window Window) bool {
	return isBorderlessStyle(getWindowStyle(window.hwnd))
}

// Remove the border and title bar
func borderlessStyle(style int32) int32 {
	return style & ^win.WS_CAPTION & ^win.WS_THICKFRAME
}

// Restore the border and title bar
func restoredStyle(style int32) int32 {
	return style | win.WS_OVERLAPPEDWINDOW
}

/**
 * Where a config wants its window: offsets are relative to the chosen monitor's
 * top left corner, not to the desktop origin.
 */
func borderlessRect(appSetting AppSetting, monitor Monitor) (x, y, width, height int32) {
	return appSetting.OffsetX + monitor.left, appSetting.OffsetY + monitor.top, appSetting.Width, appSetting.Height
}

func makeBorderless(window Window, appSetting AppSetting) {
	// fmt.Println("Making window borderless:", window.title, window.exePath)
	setWindowStyle(window.hwnd, borderlessStyle(getWindowStyle(window.hwnd)))
	x, y, width, height := borderlessRect(appSetting, monitors[appSetting.Monitor-1])
	setWindowPos(window.hwnd, x, y, width, height)

	if appSetting.BlackOverlay {
		createOverlay(window.hwnd, appSetting)
	}
}

/**
 * Only restores the window if it's borderless
 */
func restoreWindow(window Window, appSetting AppSetting) {
	if !isBorderless(window) {
		return
	}
	fmt.Println("Restoring window:", window.title, window.exePath)
	setWindowStyle(window.hwnd, restoredStyle(getWindowStyle(window.hwnd)))
	setWindowPos(window.hwnd, appSetting.PreOffsetX, appSetting.PreOffsetY, appSetting.PreWidth, appSetting.PreHeight)

	destroyOverlay(window.hwnd)
}

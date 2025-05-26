package main

import (
	"fmt"

	"github.com/lxn/win"
)

func makeBorderless(window Window, x, y, width, height int32) {
	fmt.Println("Making window borderless:", window.title, window.exePath)
	// originalRect := getWindowRect(window.hwnd) // save this
	style := getWindowStyle(window.hwnd)
	// Remove the border and title bar
	setWindowStyle(window.hwnd, style & ^win.WS_CAPTION & ^win.WS_THICKFRAME)
	setWindowPos(window.hwnd, x, y, width, height)
}

func restoreWindow(window Window) {
	style := getWindowStyle(window.hwnd)
	// Restore the border and title bar
	setWindowStyle(window.hwnd, style|win.WS_OVERLAPPEDWINDOW)
	setWindowPos(window.hwnd, 0, 0, 1920, 1080)
}

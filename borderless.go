package main

import (
	"fmt"
	"sync"

	"github.com/lxn/win"
)

var (
	taskbarMu            sync.Mutex
	taskbarOriginalState uint32
	taskbarStateSaved    bool
	taskbarHiddenByApp   bool
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
func borderlessRect(appSetting AppConfig, monitor Monitor) (x, y, width, height int32) {
	return appSetting.OffsetX + monitor.left, appSetting.OffsetY + monitor.top, appSetting.Width, appSetting.Height
}

func makeBorderless(window Window, appSetting AppConfig) {
	// fmt.Println("Making window borderless:", window.title, window.exePath)
	setWindowStyle(window.hwnd, borderlessStyle(getWindowStyle(window.hwnd)))
	// Monitor is 1-based; 0 means unset and out-of-range would panic on index.
	idx := appSetting.Monitor - 1
	if idx < 0 || idx >= len(monitors) {
		return
	}
	x, y, width, height := borderlessRect(appSetting, monitors[idx])
	setWindowPos(window.hwnd, x, y, width, height)

	if appSetting.BlackOverlay {
		createOverlay(window.hwnd, appSetting)
	}
}

/**
 * Only restores the window if it's borderless
 */
func restoreWindow(window Window, appSetting AppConfig) {
	if !isBorderless(window) {
		return
	}
	fmt.Println("Restoring window:", window.title, window.exePath)
	setWindowStyle(window.hwnd, restoredStyle(getWindowStyle(window.hwnd)))
	setWindowPos(window.hwnd, appSetting.PreOffsetX, appSetting.PreOffsetY, appSetting.PreWidth, appSetting.PreHeight)

	destroyOverlay(window.hwnd)
	if appSetting.HideTaskbar {
		restoreTaskbar()
	}
}

// saveTaskbarState records the user's original taskbar auto-hide preference
// so we can restore it later. Only saves once until explicitly restored.
func saveTaskbarState() {
	taskbarMu.Lock()
	defer taskbarMu.Unlock()
	if !taskbarStateSaved {
		state, err := getTaskbarAutoHide()
		if err != nil {
			fmt.Println("saveTaskbarState:", err)
			return
		}
		taskbarOriginalState = state
		taskbarStateSaved = true
	}
}

func hideTaskbar() {
	taskbarMu.Lock()
	defer taskbarMu.Unlock()
	if !taskbarHiddenByApp {
		if !taskbarStateSaved {
			state, err := getTaskbarAutoHide()
			if err != nil {
				fmt.Println("hideTaskbar:", err)
				return
			}
			taskbarOriginalState = state
			taskbarStateSaved = true
		}
		if err := setTaskbarAutoHide(ABS_AUTOHIDE); err != nil {
			fmt.Println("hideTaskbar:", err)
			return
		}
		taskbarHiddenByApp = true
	}
}

func restoreTaskbar() {
	taskbarMu.Lock()
	defer taskbarMu.Unlock()
	if taskbarHiddenByApp && taskbarStateSaved {
		if err := setTaskbarAutoHide(taskbarOriginalState); err != nil {
			fmt.Println("restoreTaskbar:", err)
		}
		taskbarHiddenByApp = false
	}
}

// restoreTaskbarOnExit forces taskbar restoration regardless of tracking state.
// Called when the application is shutting down.
func restoreTaskbarOnExit() {
	taskbarMu.Lock()
	defer taskbarMu.Unlock()
	if taskbarStateSaved {
		if err := setTaskbarAutoHide(taskbarOriginalState); err != nil {
			fmt.Println("restoreTaskbarOnExit:", err)
		}
		taskbarHiddenByApp = false
	}
}

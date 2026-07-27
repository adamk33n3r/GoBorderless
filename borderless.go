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
	style := getWindowStyle(window.hwnd)
	// Restore the border and title bar
	setWindowStyle(window.hwnd, style|win.WS_OVERLAPPEDWINDOW)
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

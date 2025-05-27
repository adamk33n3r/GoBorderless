package main

import (
	"fmt"
	"time"
	"unsafe"
)

// Load required Windows DLLs
var (
// modUser32   = windows.NewLazySystemDLL("user32.dll")
// modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
// modPsapi    = windows.NewLazySystemDLL("psapi.dll")

// Procedure pointers for Windows API functions
// procEnumWindows              = modUser32.NewProc("EnumWindows")
// procGetWindowTextW           = modUser32.NewProc("GetWindowTextW")
// procGetWindowTextLengthW     = modUser32.NewProc("GetWindowTextLengthW")
// procIsWindowVisible          = modUser32.NewProc("IsWindowVisible")
// procGetWindowThreadProcessId = modUser32.NewProc("GetWindowThreadProcessId")
// procGetModuleFileNameExW     = modPsapi.NewProc("GetModuleFileNameExW")
// procOpenProcess              = modKernel32.NewProc("OpenProcess")
// procCloseHandle              = modKernel32.NewProc("CloseHandle")
)

type Window struct {
	hwnd    uintptr
	title   string
	exePath string
}

func (w Window) String() string {
	return fmt.Sprintf("%s - %s", w.title, w.exePath)
}

var monitors []Monitor

var chWindowList = make(chan []Window, 1) // Channel to send window list updates
func enumWindowsCallback(hwnd uintptr, lparam uintptr) uintptr {
	// Check if the window is visible; skip if not
	if !isVisible(hwnd) {
		return 1 // continue enumeration
	}

	title := getWindowTitle(hwnd)
	if title == "" {
		return 1 // continue enumeration
	}

	exePath, err := getProcessExecutable(getProcessID(hwnd))
	if err != nil {
		exePath = ""
	}

	restoredPtr := (*[]Window)(unsafe.Pointer(lparam))

	*restoredPtr = append(*restoredPtr, Window{hwnd: hwnd, title: title, exePath: exePath})

	// Print window handle, title, and executable path
	// fmt.Printf("HWND: 0x%x\nTitle: %s\nExe Path: %s\n\n", hwnd, title, exePath)

	return 1 // continue enumeration
}

func matchWindow(win Window, appSetting AppSetting) bool {
	switch appSetting.MatchType {
	case MatchWindowTitle:
		return win.title == appSetting.WindowName
	case MatchExePath:
		return win.exePath == appSetting.ExePath
	case MatchBoth:
		return win.title == appSetting.WindowName && win.exePath == appSetting.ExePath
	case MatchEither:
		return win.title == appSetting.WindowName || win.exePath == appSetting.ExePath
	default:
		return false
	}
}

func scanWindows(settings *Settings) {
	for {
		tempList := make([]Window, 0, 32) // Temporary list to store window titles
		// Callback function for EnumWindows, called for each top-level window handle (hwnd)
		enumWindows(enumWindowsCallback, unsafe.Pointer(&tempList))

		// fmt.Println("sending channel")
		chWindowList <- tempList // Update global window list
		// fmt.Println("done sending channel")
		for appSettingIdx := range settings.Apps {
			appSetting := &settings.Apps[appSettingIdx]
			// fmt.Println("Checking app setting:", appSetting)
			for _, win := range tempList {
				if matchWindow(win, *appSetting) {
					makeBorderless(win, *appSetting)
					break
				}
			}
		}

		time.Sleep(time.Second * 1) // Sleep for 1 second before next scan
	}
}

func main() {
	settings, err := loadSettings()
	if err != nil {
		fmt.Println("Error loading settings:", err)
		backUpSettingsFile()
	}
	settings.Save()
	fmt.Println(settings)
	monitors = getMonitors()
	for _, mon := range monitors {
		fmt.Printf("Monitor %d\n", mon.number)
		fmt.Printf("  Resolution: %dx%d\n", mon.width, mon.height)
		fmt.Printf("  Position: (%d, %d)\n", mon.left, mon.top)
	}
	go scanWindows(settings)
	buildApp(settings)
}

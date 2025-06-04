package main

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"slices"

	"github.com/lxn/win"
)

const (
	APP_NAME = "GoBorderless"
)

type Window struct {
	hwnd    uintptr
	title   string
	exePath string
}

func (w Window) String() string {
	return fmt.Sprintf("%s | %s", w.title, w.exePath)
}

var monitors []Monitor
var chWindowList = make(chan []Window) // Channel to send window list updates

var ALWAYS_HIDDEN_PROCESSESS = []string{
	// Skip self
	strings.ToLower(APP_NAME),
	// Skip other borderless apps
	"borderlessgaming",
	"nomoreborder",

	// Inspired by BorderlessGaming
	// Skip Windows core system processes
	"csrss",
	"smss",
	"lsass",
	"wininit",
	"svchost",
	"services",
	"winlogon",
	"dwm",
	"explorer",
	"taskmgr",
	"mmc",
	"rundll32",
	"vcredist_x86",
	"vcredist_x64",
	"msiexec",
	// Skip common video streaming software
	"xsplit",
	"obs64",
	// Skip common web browsers
	"iexplore",
	"firefox",
	"chrome",
	"safari",
	"msedge",
	// Skip launchers/misc.
	"iw4 console",
	"steam",
	"steamwebhelper",
	"origin",
	"uplay",
}

func enumWindowsCallback(hwnd uintptr, lparam uintptr) uintptr {
	// Check if the window is visible; skip if not
	if !isVisible(hwnd) {
		return 1 // continue enumeration
	}

	title := getWindowTitle(hwnd)
	if title == "" {
		return 1 // continue enumeration
	}

	// Filter out windows that don't have normal borders cause they're probably not "real" windows
	// TODO put this only in the select dropdown since we need these kinds here to be able to restore windows
	// style := getWindowStyle(hwnd)
	// if !(style&win.WS_CAPTION > 0 &&
	// 	((style&win.WS_BORDER) > 0 || (style&win.WS_THICKFRAME) > 0)) {
	// 	return 1
	// }

	// Filter out windows with no size
	rect := getWindowRect(hwnd)
	if rect.Left == rect.Right && rect.Top == rect.Bottom {
		return 1
	}

	// exePath, err := getProcessExecutable(getProcessID(hwnd))
	// if err != nil {
	// 	return 1
	// }

	var pid uint32
	win.GetWindowThreadProcessId(win.HWND(hwnd), &pid)
	pname, err := getProcessName(pid)
	if err != nil {
		return 1
	}

	// Filter out some stuff you probably never want to see
	if slices.Contains(ALWAYS_HIDDEN_PROCESSESS, strings.ToLower(pname)[:len(pname)-4]) {
		return 1
	}
	if slices.Contains(ALWAYS_HIDDEN_PROCESSESS, strings.ToLower(title)) {
		return 1
	}

	restoredPtr := (*[]Window)(unsafe.Pointer(lparam)) //nolint:govet

	// fmt.Println("adding window to list:", title)
	*restoredPtr = append(*restoredPtr, Window{hwnd: hwnd, title: title, exePath: pname})
	// fmt.Printf("address of restoredPtr: 0x%x\n", lparam)
	// fmt.Println(len(*restoredPtr))

	// Print window handle, title, and executable path
	// fmt.Printf("HWND: 0x%x\nTitle: %s\nExe Path: %s\n\n", hwnd, title, pname)

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
		tempList := make([]Window, 0) // Temporary list to store window titles
		tempListPtr := unsafe.Pointer(&tempList)
		// fmt.Println("address of tempList:", tempListPtr)

		// stupid needs to be done for some reason. if not, the first time we call enumWinodws len(tempList) is 0
		// even though you can see it adding things in enumWindowsCallback. why does "using" the pointer cause it to work?
		var _ = fmt.Sprint(tempListPtr)
		// Callback function for EnumWindows, called for each top-level window handle (hwnd)
		enumWindows(enumWindowsCallback, tempListPtr)
		// time.Sleep(1 * time.Second)

		// fmt.Println("address of tempList after:", unsafe.Pointer(&tempList))
		// fmt.Println("sending channel:", len(tempList))
		chWindowList <- tempList // Update global window list
		// fmt.Println("done sending channel")
		for appSettingIdx, appSetting := range settings.Apps {
			// appSetting := &settings.Apps[appSettingIdx]
			if !appSetting.AutoApply {
				continue
			}
			// fmt.Println("Checking app setting:", appSetting)
			for _, win := range tempList {
				if matchWindow(win, appSetting) {
					if !isBorderless(win) {
						originalRect := getWindowRect(win.hwnd)
						appSetting.PreWidth = int32(originalRect.Right - originalRect.Left)
						appSetting.PreHeight = int32(originalRect.Bottom - originalRect.Top)
						appSetting.PreOffsetX = int32(originalRect.Left)
						appSetting.PreOffsetY = int32(originalRect.Top)
						settings.Apps[appSettingIdx] = appSetting
						settings.Save()
					}
					makeBorderless(win, appSetting)
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
	fmt.Println("Building app")
	fyneApp := buildApp(settings)
	fyneApp.Run()
}

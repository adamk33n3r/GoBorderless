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
	hwnd    win.HWND
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

func getWindowData(hwnds []win.HWND) []Window {
	windows := make([]Window, 0)
	for _, hwnd := range hwnds {
		// Check if the window is visible; skip if not
		if !isVisible(uintptr(hwnd)) {
			continue
		}

		title := getWindowTitle(uintptr(hwnd))
		if title == "" {
			continue
		}

		// Filter out windows with no size
		rect := getWindowRect(hwnd)
		if rect.Left == rect.Right && rect.Top == rect.Bottom {
			continue
		}

		// exePath, err := getProcessExecutable(getProcessID(hwnd))
		// if err != nil {
		// 	return 1
		// }

		var pid uint32
		win.GetWindowThreadProcessId(win.HWND(hwnd), &pid)
		pname, err := getProcessName(pid)
		if err != nil {
			continue
		}

		// Filter out some stuff you probably never want to see
		if slices.Contains(ALWAYS_HIDDEN_PROCESSESS, strings.ToLower(pname)[:len(pname)-4]) {
			continue
		}
		if slices.Contains(ALWAYS_HIDDEN_PROCESSESS, strings.ToLower(title)) {
			continue
		}

		windows = append(windows, Window{hwnd: hwnd, title: title, exePath: pname})
	}
	return windows
}

func enumWindowsCallback(hwnd uintptr, lparam uintptr) uintptr {
	restoredPtr := (*[]win.HWND)(unsafe.Pointer(lparam)) //nolint:govet
	*restoredPtr = append(*restoredPtr, win.HWND(hwnd))
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

func getPrimaryMonitor() Monitor {
	for _, monitor := range monitors {
		if monitor.isPrimary {
			return monitor
		}
	}
	return monitors[0]
}

type EnumWindowsWrapper struct {
	windows []win.HWND
}

func EnumWindows() []win.HWND {
	enumWindowsWrapper := &EnumWindowsWrapper{
		windows: make([]win.HWND, 0),
	}
	enumWindows(enumWindowsCallback, unsafe.Pointer(enumWindowsWrapper))
	return enumWindowsWrapper.windows
}

func scanWindows(settings *Settings) {
	for {
		allWindows := EnumWindows()
		windowData := getWindowData(allWindows)

		chWindowList <- windowData // Update global window list
		for appSettingIdx, appSetting := range settings.Apps {
			if !appSetting.AutoApply {
				continue
			}
			for _, win := range windowData {
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

	monitors = getMonitors()
	primaryMonitor := getPrimaryMonitor()
	if settings.Defaults.Monitor == 0 {
		settings.Defaults.Monitor = primaryMonitor.number
	}
	if settings.Defaults.Width == 0 {
		settings.Defaults.Width = primaryMonitor.width
	}
	if settings.Defaults.Height == 0 {
		settings.Defaults.Height = primaryMonitor.height
	}

	settings.Save()
	fmt.Println(settings)
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

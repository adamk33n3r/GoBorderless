package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/lxn/win"

	"golang.org/x/sys/windows"
)

const (
	maxPath          = 260    // Maximum path length for Windows file paths
	processQueryInfo = 0x0400 // Process access right for querying information
	processVMRead    = 0x0010 // Process access right for reading memory
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	psapi    = windows.NewLazySystemDLL("psapi.dll")

	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetModuleFileNameExW     = psapi.NewProc("GetModuleFileNameExW")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procEnumDisplayMonitors      = user32.NewProc("EnumDisplayMonitors")
	procGetKnownFolderPath       = shell32.NewProc("SHGetKnownFolderPath")
)

func enumWindows(callback func(hwnd uintptr, lparam uintptr) uintptr, extra unsafe.Pointer) {
	// Callback function for EnumWindows, called for each top-level window handle (hwnd)
	procEnumWindows.Call(windows.NewCallback(callback), uintptr(extra))
}

func isVisible(hwnd uintptr) bool {
	isVisible, _, _ := procIsWindowVisible.Call(hwnd)
	return isVisible != 0
}

func getWindowTitle(hwnd uintptr) string {
	textLen, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if textLen == 0 {
		return ""
	}
	// fmt.Println("textLen", textLen)

	textBuf := make([]uint16, textLen+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&textBuf[0])), uintptr(len(textBuf)))
	return windows.UTF16ToString(textBuf)
}

func getProcessID(hwnd uintptr) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}

func getProcessExecutable(pid uint32) (string, error) {
	hProc, _, err := procOpenProcess.Call(processQueryInfo|processVMRead, 0, uintptr(pid))
	if hProc == 0 {
		return "", err
	}
	defer procCloseHandle.Call(hProc) // Ensure handle is closed after use
	exeBuf := make([]uint16, maxPath)
	procGetModuleFileNameExW.Call(hProc, 0, uintptr(unsafe.Pointer(&exeBuf[0])), uintptr(maxPath))
	exePath := windows.UTF16ToString(exeBuf)
	return exePath, nil
}

func moveWindow(hwnd uintptr, x, y, width, height int32) {
	win.MoveWindow(win.HWND(hwnd), x, y, width, height, true)
}

func setWindowPos(hwnd uintptr, x, y, width, height int32) {
	win.SetWindowPos(win.HWND(hwnd), 0, x, y, width, height, win.SWP_NOZORDER)
}

func getWindowRect(hwnd uintptr) win.RECT {
	rect := win.RECT{}
	win.GetWindowRect(win.HWND(hwnd), &rect)
	return rect
}

func getWindowStyle(hwnd uintptr) int32 {
	return win.GetWindowLong(win.HWND(hwnd), win.GWL_STYLE)
	// index := GWL_STYLE
	// style, _, err := procGetWindowLong.Call(hwnd, uintptr(index))
	// if style == 0 {
	// 	return 0, err
	// }
	// return uint32(style), nil
}

func setWindowStyle(hwnd uintptr, style int32) {
	win.SetWindowLong(win.HWND(hwnd), win.GWL_STYLE, style)
	win.SetWindowPos(win.HWND(hwnd), 0, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_FRAMECHANGED)
}

type Monitor struct {
	number    int
	isPrimary bool
	width     int32
	height    int32
	left      int32
	top       int32
}

func (m Monitor) String() string {
	str := fmt.Sprintf("Display %d", m.number)
	if m.isPrimary {
		str += " (Primary)"
	}
	return str
}

func getMonitors() []Monitor {
	var monitors []Monitor
	index := 0
	cb := syscall.NewCallback(func(hMonitor win.HMONITOR, hdcMonitor win.HDC, lprcMonitor *win.RECT, dwData uintptr) uintptr {
		var info win.MONITORINFO
		info.CbSize = uint32(unsafe.Sizeof(info))
		if win.GetMonitorInfo(hMonitor, &info) {
			index++
			// info.RcMonitor.Left = info.RcMonitor.Left / win.GetDeviceCaps(hdcMonitor, win.LOGPIXELSX) * win.GetDeviceCaps(hdcMonitor, win.LOGPIXELSX)
			fmt.Println("Monitor ", index, "\n", info.DwFlags&win.MONITORINFOF_PRIMARY != 0, "\n", info.RcMonitor, "\n", info.RcWork)
			monitors = append(monitors, Monitor{
				number:    index,
				isPrimary: info.DwFlags&win.MONITORINFOF_PRIMARY != 0,
				width:     info.RcMonitor.Right - info.RcMonitor.Left,
				height:    info.RcMonitor.Bottom - info.RcMonitor.Top,
				left:      info.RcMonitor.Left,
				top:       info.RcMonitor.Top,
			})
		}
		return 1
	})
	procEnumDisplayMonitors.Call(0, 0, cb, 0)
	return monitors
}

func getDocumentsFolder() string {
	var buf uintptr
	hr, _, _ := procGetKnownFolderPath.Call(uintptr(unsafe.Pointer(windows.FOLDERID_Documents)), 0, 0, uintptr(unsafe.Pointer(&buf)))
	if hr != 0 {
		return ""
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(buf))
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(buf)))
}

package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/lxn/win"

	"golang.org/x/sys/windows"
)

const (
	maxPath = 260 // Maximum path length for Windows file paths
)

var (
	user32  = windows.NewLazySystemDLL("user32.dll")
	shell32 = windows.NewLazySystemDLL("shell32.dll")

	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procEnumDisplayMonitors  = user32.NewProc("EnumDisplayMonitors")
	procGetKnownFolderPath   = shell32.NewProc("SHGetKnownFolderPath")
)

func enumWindows(callback func(hwnd uintptr, lparam uintptr) uintptr, extra unsafe.Pointer) {
	windows.EnumWindows(windows.NewCallback(callback), extra)
}

func isVisible(hwnd uintptr) bool {
	return win.IsWindowVisible(win.HWND(hwnd))
}

func getWindowTitle(hwnd uintptr) string {
	textLen, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if textLen == 0 {
		return ""
	}

	textBuf := make([]uint16, textLen+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&textBuf[0])), uintptr(len(textBuf)))
	return windows.UTF16ToString(textBuf)
}

func getProcessName(pid uint32) (string, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle) // Ensure handle is closed after use
	processNameBuf := make([]uint16, maxPath)
	err = windows.GetModuleBaseName(handle, 0, &processNameBuf[0], maxPath)
	if err != nil {
		return "", err
	}
	processName := windows.UTF16ToString(processNameBuf)
	return processName, nil
}

func getProcessExecutable(pid uint32) (string, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if handle == 0 {
		return "", err
	}
	defer windows.CloseHandle(handle) // Ensure handle is closed after use
	exeBuf := make([]uint16, maxPath)
	windows.GetModuleFileNameEx(handle, 0, &exeBuf[0], maxPath)
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
	defer windows.CoTaskMemFree(unsafe.Pointer(buf))                //nolint:govet
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(buf))) //nolint:govet
}

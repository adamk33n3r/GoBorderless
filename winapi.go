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

const (
	ABM_GETSTATE = 0x00000004
	ABM_SETSTATE = 0x0000000A
	ABS_AUTOHIDE = 0x01
	ABS_ALWAYSONTOP = 0x02
)

type APPBARDATA struct {
	cbSize           uint32
	hWnd             uintptr
	uCallbackMessage uint32
	uEdge            uint32
	rc               win.RECT
	lParam           uintptr
}

var (
	user32  = windows.NewLazySystemDLL("user32.dll")
	shell32 = windows.NewLazySystemDLL("shell32.dll")

	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procEnumDisplayMonitors  = user32.NewProc("EnumDisplayMonitors")
	procGetForegroundWindow  = user32.NewProc("GetForegroundWindow")
	procFindWindowW          = user32.NewProc("FindWindowW")
	procGetKnownFolderPath   = shell32.NewProc("SHGetKnownFolderPath")
	procSHAppBarMessage      = shell32.NewProc("SHAppBarMessage")
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

func getProcessPath(pid uint32) (string, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle) // Ensure handle is closed after use
	processPathBuf := make([]uint16, maxPath)
	processPathBufLen := uint32(maxPath)
	err = windows.QueryFullProcessImageName(handle, 0, &processPathBuf[0], &processPathBufLen)
	if err != nil {
		return "", err
	}
	processPath := windows.UTF16ToString(processPathBuf)
	return processPath, nil
}

func moveWindow(hwnd win.HWND, x, y, width, height int32) {
	win.MoveWindow(hwnd, x, y, width, height, true)
}

func setWindowPos(hwnd win.HWND, x, y, width, height int32) {
	win.SetWindowPos(hwnd, 0, x, y, width, height, win.SWP_NOZORDER)
}

func getWindowRect(hwnd win.HWND) win.RECT {
	rect := win.RECT{}
	win.GetWindowRect(hwnd, &rect)
	return rect
}

func getWindowStyle(hwnd win.HWND) int32 {
	return win.GetWindowLong(hwnd, win.GWL_STYLE)
}

func setWindowStyle(hwnd win.HWND, style int32) {
	win.SetWindowLong(hwnd, win.GWL_STYLE, style)
	win.SetWindowPos(hwnd, 0, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_FRAMECHANGED)
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

func createMutex(name string) (windows.Handle, error) {
	if namePtr, err := windows.UTF16PtrFromString(name); err == nil {
		return windows.CreateMutex(nil, false, namePtr)
	}
	return windows.Handle(0), fmt.Errorf("failed to create mutex")
}

func openMutex(name string) (windows.Handle, error) {
	if namePtr, err := windows.UTF16PtrFromString(name); err == nil {
		return windows.OpenMutex(windows.SYNCHRONIZE, false, namePtr)
	}
	return windows.Handle(0), fmt.Errorf("failed to open mutex")
}

func closeMutex(mutex windows.Handle) error {
	return windows.CloseHandle(mutex)
}

// func waitForSingleObject(mutex *windows.Mutex) error {
// 	return windows.WaitForSingleObject(windows.Handle(mutex), windows.INFINITE)
// }

func getForegroundWindow() win.HWND {
	ret, _, _ := procGetForegroundWindow.Call()
	return win.HWND(ret)
}

func findWindowByClass(className string) (uintptr, error) {
	classPtr, err := windows.UTF16PtrFromString(className)
	if err != nil {
		return 0, fmt.Errorf("findWindowByClass: %w", err)
	}
	ret, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(classPtr)), 0)
	if ret == 0 {
		return 0, fmt.Errorf("findWindowByClass: window %q not found", className)
	}
	return ret, nil
}

func getTaskbarAutoHide() (uint32, error) {
	taskbarHwnd, err := findWindowByClass("Shell_TrayWnd")
	if err != nil {
		return 0, fmt.Errorf("getTaskbarAutoHide: %w", err)
	}
	abd := APPBARDATA{
		cbSize: uint32(unsafe.Sizeof(APPBARDATA{})),
		hWnd:   taskbarHwnd,
	}
	ret, _, _ := procSHAppBarMessage.Call(ABM_GETSTATE, uintptr(unsafe.Pointer(&abd)))
	return uint32(ret), nil
}

func setTaskbarAutoHide(state uint32) error {
	taskbarHwnd, err := findWindowByClass("Shell_TrayWnd")
	if err != nil {
		return fmt.Errorf("setTaskbarAutoHide: %w", err)
	}
	abd := APPBARDATA{
		cbSize: uint32(unsafe.Sizeof(APPBARDATA{})),
		hWnd:   taskbarHwnd,
		lParam: uintptr(state),
	}
	procSHAppBarMessage.Call(ABM_SETSTATE, uintptr(unsafe.Pointer(&abd)))
	return nil
}

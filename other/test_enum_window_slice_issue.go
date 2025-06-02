package main

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const maxPath = 260

var (
	user32 = windows.NewLazySystemDLL("user32.dll")

	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
)

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

func enumWindowsCallback(hwnd uintptr, lparam uintptr) uintptr {
	var pid uint32
	win.GetWindowThreadProcessId(win.HWND(hwnd), &pid)
	// COMMENT THIS FUNCTION CALL OUT TO "FIX" THE ISSUE
	_, _ = getProcessName(pid)

	textLen, _, _ := procGetWindowTextLengthW.Call(hwnd)
	textBuf := make([]uint16, textLen+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&textBuf[0])), uintptr(len(textBuf)))
	winTitle := windows.UTF16ToString(textBuf)
	if winTitle == "" {
		return 1
	}

	windowList := (*[]string)(unsafe.Pointer(lparam))
	*windowList = append(*windowList, winTitle)
	// fmt.Println("append:", len(*windowList)) // UNCOMMENT THIS TO SEE THAT IT IS INDEED APPENDING DURING FIRST CALL
	return 1
}

func main() {
	done := make(chan bool)
	go func() {
		for range 2 {
			windowList := make([]string, 0)
			// var _ = fmt.Sprint(unsafe.Pointer(&windowList)) // UNCOMMENT THIS TO "FIX" THE ISSUE
			windows.EnumWindows(windows.NewCallback(enumWindowsCallback), unsafe.Pointer(&windowList))
			fmt.Println("length of windowList after:", len(windowList))
			time.Sleep(1 * time.Second)
		}
		done <- true
	}()

	for done := range done {
		if done {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

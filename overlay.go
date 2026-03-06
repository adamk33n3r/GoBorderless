package main

import (
	"log"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

const overlayClassName = "GoBorderlessOverlay"

type Overlay struct {
	hwnd    win.HWND
	appHwnd win.HWND
	monitor Monitor
}

var (
	activeOverlays   = map[win.HWND]*Overlay{} // keyed by app HWND
	overlayMu        sync.Mutex
	overlayClassOnce sync.Once
	overlayWndProcCb uintptr
)

func registerOverlayClass() {
	overlayClassOnce.Do(func() {
		overlayWndProcCb = syscall.NewCallback(overlayWndProc)
		classNamePtr, _ := syscall.UTF16PtrFromString(overlayClassName)
		wc := win.WNDCLASSEX{
			CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
			Style:         win.CS_HREDRAW | win.CS_VREDRAW,
			LpfnWndProc:   overlayWndProcCb,
			HInstance:     win.GetModuleHandle(nil),
			HbrBackground: win.HBRUSH(win.GetStockObject(win.BLACK_BRUSH)),
			LpszClassName: classNamePtr,
		}
		win.RegisterClassEx(&wc)
	})
}

func overlayWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_LBUTTONDOWN, win.WM_RBUTTONDOWN:
		log.Printf("Overlay window received click message: %d\n", msg)
		// Focus the associated app window
		overlayMu.Lock()
		var targetHwnd win.HWND
		for appHwnd, overlay := range activeOverlays {
			if overlay.hwnd == hwnd {
				targetHwnd = appHwnd
				break
			}
		}
		overlayMu.Unlock()
		if targetHwnd != 0 {
			log.Printf("Overlay clicked, setting focus to app HWND: %d\n", targetHwnd)
			win.SetForegroundWindow(targetHwnd)
		} else {
			log.Printf("Overlay clicked but no associated app found for HWND: %d\n", hwnd)
		}
		return 0
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

// createOverlay creates a black overlay window covering the monitor around the app.
// It is safe to call concurrently; duplicate calls for the same appHwnd are no-ops.
func createOverlay(appHwnd win.HWND, appSetting AppSetting) {
	registerOverlayClass()

	overlayMu.Lock()
	if _, exists := activeOverlays[appHwnd]; exists {
		overlayMu.Unlock()
		return
	}
	// Add placeholder to prevent duplicate creation during goroutine startup
	activeOverlays[appHwnd] = &Overlay{appHwnd: appHwnd}
	overlayMu.Unlock()

	monitor := monitors[appSetting.Monitor-1]

	go func() {
		// Win32 message loops must run on the same OS thread as the window
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		classNamePtr, _ := syscall.UTF16PtrFromString(overlayClassName)
		titlePtr, _ := syscall.UTF16PtrFromString("")

		hwnd := win.CreateWindowEx(
			win.WS_EX_TOOLWINDOW|win.WS_EX_NOACTIVATE,
			classNamePtr,
			titlePtr,
			win.WS_POPUP,
			monitor.left, monitor.top, monitor.width, monitor.height,
			0, 0, win.GetModuleHandle(nil), nil,
		)
		if hwnd == 0 {
			overlayMu.Lock()
			delete(activeOverlays, appHwnd)
			overlayMu.Unlock()
			return
		}

		overlayMu.Lock()
		activeOverlays[appHwnd].hwnd = hwnd
		activeOverlays[appHwnd].monitor = monitor
		overlayMu.Unlock()

		applyOverlayRegion(hwnd, appSetting, monitor)

		// Place overlay just below the app in z-order, then show without activating
		win.SetWindowPos(hwnd, appHwnd, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOACTIVATE)
		win.ShowWindow(hwnd, win.SW_SHOWNOACTIVATE)

		// Run the message loop for this overlay window
		var msg win.MSG
		for win.GetMessage(&msg, 0, 0, 0) > 0 {
			win.TranslateMessage(&msg)
			win.DispatchMessage(&msg)
		}
	}()
}

// applyOverlayRegion sets a region on the overlay that covers the full monitor
// except for a transparent "hole" where the app sits.
func applyOverlayRegion(overlayHwnd win.HWND, appSetting AppSetting, monitor Monitor) {
	// Full monitor area in window-local coordinates (overlay is positioned at monitor origin)
	fullRgn := win.CreateRectRgn(0, 0, monitor.width, monitor.height)

	// App rect is relative to monitor origin (OffsetX/Y are relative to monitor in makeBorderless)
	appX := appSetting.OffsetX
	appY := appSetting.OffsetY
	holeRgn := win.CreateRectRgn(appX, appY, appX+appSetting.Width, appY+appSetting.Height)

	// Subtract the hole from the full region
	win.CombineRgn(fullRgn, fullRgn, holeRgn, win.RGN_DIFF)
	win.DeleteObject(win.HGDIOBJ(holeRgn))

	// SetWindowRgn takes ownership of fullRgn on success; do not DeleteObject it
	setWindowRgn(overlayHwnd, fullRgn, true)
}

// destroyOverlay closes the overlay window for the given app HWND.
func destroyOverlay(appHwnd win.HWND) {
	overlayMu.Lock()
	overlay, ok := activeOverlays[appHwnd]
	if ok {
		delete(activeOverlays, appHwnd)
	}
	overlayMu.Unlock()

	if ok && overlay.hwnd != 0 {
		win.PostMessage(overlay.hwnd, win.WM_CLOSE, 0, 0)
	}
}

// hideOverlay hides the overlay window for the given app HWND (e.g. app minimized).
func hideOverlay(appHwnd win.HWND) {
	overlayMu.Lock()
	overlay, ok := activeOverlays[appHwnd]
	overlayMu.Unlock()
	if ok && overlay.hwnd != 0 {
		win.ShowWindow(overlay.hwnd, win.SW_HIDE)
	}
}

// showOverlay makes the overlay visible again (e.g. app restored from minimize).
func showOverlay(appHwnd win.HWND) {
	overlayMu.Lock()
	overlay, ok := activeOverlays[appHwnd]
	overlayMu.Unlock()
	if ok && overlay.hwnd != 0 {
		win.ShowWindow(overlay.hwnd, win.SW_SHOWNOACTIVATE)
	}
}

// syncOverlayZOrder places the overlay just behind the app in z-order.
func syncOverlayZOrder(appHwnd win.HWND) {
	overlayMu.Lock()
	overlay, ok := activeOverlays[appHwnd]
	overlayMu.Unlock()
	if ok && overlay.hwnd != 0 {
		win.SetWindowPos(overlay.hwnd, appHwnd, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOACTIVATE)
	}
}

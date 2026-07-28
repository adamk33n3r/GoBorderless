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

// registerOverlayClass registers the Win32 window class used by all overlay windows.
// It is safe to call multiple times — the actual registration only runs once via sync.Once.
// The class uses a solid black background brush and the overlayWndProc message handler.
func registerOverlayClass() {
	overlayClassOnce.Do(func() {
		// Convert the WndProc func to a C-callable uintptr; must be stored globally
		// to prevent the garbage collector from reclaiming it while callbacks are active.
		overlayWndProcCb = syscall.NewCallback(overlayWndProc)

		classNamePtr, _ := syscall.UTF16PtrFromString(overlayClassName)
		wc := win.WNDCLASSEX{
			CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
			Style:         win.CS_HREDRAW | win.CS_VREDRAW, // Redraw on resize
			LpfnWndProc:   overlayWndProcCb,
			HInstance:     win.GetModuleHandle(nil),
			HCursor:       win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_ARROW)),
			HbrBackground: win.HBRUSH(win.GetStockObject(win.BLACK_BRUSH)), // Solid black fill
			LpszClassName: classNamePtr,
		}
		win.RegisterClassEx(&wc)
	})
}

// overlayWndProc is the Win32 window procedure (message handler) for overlay windows.
// It handles mouse clicks by focusing the associated app window, and cleans up the
// message loop on window destruction. All other messages are forwarded to the default
// Win32 handler.
//
// Parameters:
//   - hwnd:   Handle to the overlay window receiving the message.
//   - msg:    The Windows message identifier (e.g. WM_LBUTTONDOWN).
//   - wParam: Message-specific parameter (unused here).
//   - lParam: Message-specific parameter (unused here).
//
// Returns the result of message processing (0 for handled messages).
func overlayWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_LBUTTONDOWN, win.WM_RBUTTONDOWN:
		// Reverse-lookup: find the app HWND whose overlay matches this window.
		// The overlay map is keyed by app HWND, so we iterate to find a match.
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
			// Bring the associated app to the foreground so clicking the black
			// border area behaves as if the user clicked on the app itself.
			log.Printf("Overlay clicked, setting focus to app HWND: %d\n", targetHwnd)
			win.SetForegroundWindow(targetHwnd)
		}
		return 0

	case win.WM_DESTROY:
		// Signal the GetMessage loop in the goroutine to exit.
		win.PostQuitMessage(0)
		return 0
	}

	// Pass all unhandled messages to the default Win32 handler.
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

// createOverlay creates a black overlay window covering the full monitor around the given app.
// The overlay is a borderless WS_POPUP window with a transparent hole cut out where the app sits,
// so only the surrounding area is filled with black. The window runs its own Win32 message loop
// in a dedicated goroutine pinned to a single OS thread.
//
// It is safe to call concurrently — a placeholder entry is inserted into activeOverlays before
// the goroutine starts, so duplicate calls for the same appHwnd are no-ops.
//
// Parameters:
//   - appHwnd:  Handle to the target application window being made borderless.
//   - payload:  Apply geometry/flags used to determine monitor, size, and position.
func createOverlay(appHwnd win.HWND, payload ApplyPayload) {
	// Ensure the Win32 window class is registered before creating any overlay windows.
	registerOverlayClass()

	// Guard against duplicate overlays for the same app. Insert a placeholder immediately
	// so concurrent calls returning before the goroutine creates the real window.
	overlayMu.Lock()
	if _, exists := activeOverlays[appHwnd]; exists {
		overlayMu.Unlock()
		return
	}
	activeOverlays[appHwnd] = &Overlay{appHwnd: appHwnd}
	overlayMu.Unlock()

	monitor := monitors[payload.Monitor-1]

	go func() {
		// Win32 requires that GetMessage, window creation, and message dispatch all run
		// on the same OS thread. LockOSThread prevents the Go scheduler from migrating
		// this goroutine to a different thread.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		classNamePtr, _ := syscall.UTF16PtrFromString(overlayClassName)
		titlePtr, _ := syscall.UTF16PtrFromString("")

		// Create a full-monitor, borderless popup window.
		// WS_EX_TOOLWINDOW hides it from the taskbar/alt-tab switcher.
		// WS_EX_NOACTIVATE prevents it from stealing keyboard focus on click.
		hwnd := win.CreateWindowEx(
			win.WS_EX_TOOLWINDOW|win.WS_EX_NOACTIVATE,
			classNamePtr,
			titlePtr,
			win.WS_POPUP,
			monitor.left, monitor.top, monitor.width, monitor.height,
			0, 0, win.GetModuleHandle(nil), nil,
		)
		if hwnd == 0 {
			// Window creation failed; remove the placeholder so a retry is possible.
			overlayMu.Lock()
			delete(activeOverlays, appHwnd)
			overlayMu.Unlock()
			return
		}

		// Update the placeholder entry with the real window handle and monitor info.
		overlayMu.Lock()
		activeOverlays[appHwnd].hwnd = hwnd
		activeOverlays[appHwnd].monitor = monitor
		overlayMu.Unlock()

		// Cut a transparent hole in the overlay region so the app window shows through.
		applyOverlayRegion(hwnd, payload, monitor)

		// Position the overlay directly behind the app in z-order so it moves with the
		// app without ever appearing above it. SWP_NOMOVE|SWP_NOSIZE keeps position/size.
		win.SetWindowPos(hwnd, appHwnd, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOACTIVATE)

		// Show the overlay without activating it (preserves the app's foreground status).
		win.ShowWindow(hwnd, win.SW_SHOWNOACTIVATE)

		// Pump Win32 messages until WM_QUIT is posted (via PostQuitMessage in WM_DESTROY).
		var msg win.MSG
		for win.GetMessage(&msg, 0, 0, 0) > 0 {
			win.TranslateMessage(&msg)
			win.DispatchMessage(&msg)
		}
	}()
}

// applyOverlayRegion configures the visible region of the overlay window so that
// the full monitor area is filled with black except for a transparent hole where
// the app window sits. Clicks on the black region are received by the overlay;
// clicks inside the hole fall through to the app window naturally.
//
// The overlay window is positioned at the monitor's top-left corner, so all
// coordinates here are monitor-local (i.e. relative to the monitor origin).
// ApplyPayload.OffsetX/Y are also monitor-relative (set by makeBorderless).
//
// Parameters:
//   - overlayHwnd: Handle to the overlay window whose region will be updated.
//   - payload:     Apply payload providing the hole's position (OffsetX/Y) and size (Width/Height).
//   - monitor:     The monitor the overlay covers, used for the outer boundary dimensions.
func applyOverlayRegion(overlayHwnd win.HWND, payload ApplyPayload, monitor Monitor) {
	// Start with a region that covers the entire monitor surface.
	fullRgn := win.CreateRectRgn(0, 0, monitor.width, monitor.height)

	// Create a rectangular region representing the app's position within the monitor.
	// OffsetX/Y are monitor-relative, matching how makeBorderless positions the window.
	appX := payload.OffsetX
	appY := payload.OffsetY
	holeRgn := win.CreateRectRgn(appX, appY, appX+payload.Width, appY+payload.Height)

	// Subtract the hole region from the full region, leaving a "donut" shape.
	// RGN_DIFF computes: fullRgn = fullRgn - holeRgn.
	win.CombineRgn(fullRgn, fullRgn, holeRgn, win.RGN_DIFF)

	// The hole region is no longer needed after the combine operation.
	win.DeleteObject(win.HGDIOBJ(holeRgn))

	// Apply the region to the overlay window. On success, the OS takes ownership of
	// fullRgn and will free it — do NOT call DeleteObject on it after this point.
	setWindowRgn(overlayHwnd, fullRgn, true)
}

// destroyOverlay closes and removes the overlay window associated with the given app HWND.
// The overlay entry is removed from activeOverlays before the window is destroyed so that
// no further lifecycle operations attempt to act on it. The overlay's message loop goroutine
// exits naturally when it receives WM_CLOSE → WM_DESTROY → PostQuitMessage.
//
// Parameters:
//   - appHwnd: Handle to the app window whose overlay should be destroyed.
func destroyOverlay(appHwnd win.HWND) {
	// Remove from the active map first so scanWindows won't try to act on it again.
	overlayMu.Lock()
	overlay, ok := activeOverlays[appHwnd]
	if ok {
		delete(activeOverlays, appHwnd)
	}
	overlayMu.Unlock()

	if ok && overlay.hwnd != 0 {
		// PostMessage is asynchronous and safe to call from any goroutine.
		// WM_CLOSE triggers the default handler which calls DestroyWindow → WM_DESTROY.
		win.PostMessage(overlay.hwnd, win.WM_CLOSE, 0, 0)
	}
}

// hideOverlay hides the overlay window without destroying it.
// Called when the associated app is minimized so the overlay doesn't linger
// on screen while the app is not visible.
//
// Parameters:
//   - appHwnd: Handle to the app window whose overlay should be hidden.
func hideOverlay(appHwnd win.HWND) {
	overlayMu.Lock()
	overlay, ok := activeOverlays[appHwnd]
	overlayMu.Unlock()
	if ok && overlay.hwnd != 0 {
		win.ShowWindow(overlay.hwnd, win.SW_HIDE)
	}
}

// showOverlay makes the overlay visible again without stealing focus.
// Called when the associated app is restored from a minimized state.
//
// Parameters:
//   - appHwnd: Handle to the app window whose overlay should be shown.
func showOverlay(appHwnd win.HWND) {
	overlayMu.Lock()
	overlay, ok := activeOverlays[appHwnd]
	overlayMu.Unlock()
	if ok && overlay.hwnd != 0 {
		// SW_SHOWNOACTIVATE displays the window without changing the active/focused window.
		win.ShowWindow(overlay.hwnd, win.SW_SHOWNOACTIVATE)
	}
}

// syncOverlayZOrder places the overlay directly behind the app in the window z-order.
// Called each scan tick so the overlay stays paired with the app as other windows
// are raised and lowered. Passing appHwnd as the insert-after handle causes Windows
// to position the overlay immediately below the app.
//
// Parameters:
//   - appHwnd: Handle to the app window; the overlay will be placed just below it.
func syncOverlayZOrder(appHwnd win.HWND) {
	overlayMu.Lock()
	overlay, ok := activeOverlays[appHwnd]
	overlayMu.Unlock()
	if ok && overlay.hwnd != 0 {
		// SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE: only change z-order, leave everything else alone.
		win.SetWindowPos(overlay.hwnd, appHwnd, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOACTIVATE)
	}
}

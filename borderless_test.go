package main

import (
	"testing"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

// styleBits converts a Win32 style constant to the signed value GetWindowLong
// hands back. WS_POPUP has the top bit set, so it cannot be written as an
// int32 constant directly.
func styleBits(style uint32) int32 { return int32(style) }

func TestIsBorderlessStyle(t *testing.T) {
	tests := []struct {
		name  string
		style int32
		want  bool
	}{
		{"no style at all", 0, true},
		{"standard window", win.WS_OVERLAPPEDWINDOW, false},
		{"caption and border", win.WS_CAPTION | win.WS_BORDER, false},
		{"caption and thick frame", win.WS_CAPTION | win.WS_THICKFRAME, false},
		{"caption only", win.WS_DLGFRAME, true},
		{"thick frame without caption", win.WS_THICKFRAME, true},
		{"after makeBorderless strips the frame", win.WS_OVERLAPPEDWINDOW & ^win.WS_CAPTION & ^win.WS_THICKFRAME, true},
		{"visible fullscreen popup", styleBits(win.WS_POPUP | win.WS_VISIBLE), true},
		{"unrelated styles only", styleBits(win.WS_VISIBLE | win.WS_CLIPCHILDREN), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBorderlessStyle(tt.style); got != tt.want {
				t.Errorf("isBorderlessStyle(%#x) = %v, want %v", uint32(tt.style), got, tt.want)
			}
		})
	}
}

// WS_BORDER is a subset of WS_CAPTION, so a window carrying only WS_BORDER
// still trips the caption check. Documents that overlap explicitly.
func TestIsBorderlessStyleBorderImpliesCaptionBit(t *testing.T) {
	if isBorderlessStyle(win.WS_BORDER) {
		t.Error("WS_BORDER alone should count as bordered (it overlaps WS_CAPTION)")
	}
}

func TestBorderlessStyle(t *testing.T) {
	stripped := borderlessStyle(win.WS_OVERLAPPEDWINDOW)

	if stripped&win.WS_CAPTION != 0 {
		t.Error("WS_CAPTION survived")
	}
	if stripped&win.WS_THICKFRAME != 0 {
		t.Error("WS_THICKFRAME survived")
	}
	// Everything else about the window must be left alone.
	for name, bit := range map[string]int32{
		"WS_VISIBLE":       win.WS_VISIBLE,
		"WS_MINIMIZEBOX":   win.WS_MINIMIZEBOX,
		"WS_MAXIMIZEBOX":   win.WS_MAXIMIZEBOX,
		"WS_SYSMENU":       win.WS_SYSMENU,
		"WS_CLIPCHILDREN":  win.WS_CLIPCHILDREN,
		"WS_CLIPSIBLINGS":  win.WS_CLIPSIBLINGS,
		"WS_HSCROLL":       win.WS_HSCROLL,
		"WS_VSCROLL":       win.WS_VSCROLL,
		"WS_TABSTOP":       win.WS_TABSTOP,
		"WS_MINIMIZE":      win.WS_MINIMIZE,
		"WS_MAXIMIZE":      win.WS_MAXIMIZE,
		"WS_DISABLED":      win.WS_DISABLED,
		"WS_GROUP":         win.WS_GROUP,
		"WS_CHILD":         win.WS_CHILD,
		"WS_OVERLAPPED":    win.WS_OVERLAPPED,
		"WS_EX_TOPMOST(0)": 0,
	} {
		if bit == 0 {
			continue
		}
		if got := borderlessStyle(bit); got != bit {
			t.Errorf("borderlessStyle stripped %s: %#x -> %#x", name, uint32(bit), uint32(got))
		}
	}

	if got := borderlessStyle(stripped); got != stripped {
		t.Error("borderlessStyle is not idempotent")
	}
	if !isBorderlessStyle(stripped) {
		t.Error("a stripped style should read as borderless")
	}
}

func TestRestoredStyle(t *testing.T) {
	original := int32(win.WS_OVERLAPPEDWINDOW | win.WS_VISIBLE)

	restored := restoredStyle(borderlessStyle(original))

	if restored != original {
		t.Errorf("round trip gave %#x, want %#x", uint32(restored), uint32(original))
	}
	if isBorderlessStyle(restored) {
		t.Error("a restored style should read as bordered")
	}
	if got := restoredStyle(restored); got != restored {
		t.Error("restoredStyle is not idempotent")
	}
}

// A window that never had a caption (a popup) still gets the full overlapped
// frame back, which is how Restore rescues a window that was already borderless
// when the config was created.
func TestRestoredStyleAddsFrameToPopup(t *testing.T) {
	restored := restoredStyle(styleBits(win.WS_POPUP))

	if restored&win.WS_CAPTION == 0 || restored&win.WS_THICKFRAME == 0 {
		t.Errorf("restoredStyle(%#x) = %#x, want a full frame", uint32(win.WS_POPUP), uint32(restored))
	}
}

func TestBorderlessRect(t *testing.T) {
	tests := []struct {
		name                  string
		setting               AppConfig
		monitor               Monitor
		wantX, wantY          int32
		wantWidth, wantHeight int32
	}{
		{
			name:    "primary monitor at the origin",
			setting: AppConfig{OffsetX: 0, OffsetY: 0, Width: 1920, Height: 1080},
			monitor: Monitor{number: 1, left: 0, top: 0},
			wantX:   0, wantY: 0, wantWidth: 1920, wantHeight: 1080,
		},
		{
			name:    "offsets are added to the monitor origin",
			setting: AppConfig{OffsetX: 10, OffsetY: 20, Width: 800, Height: 600},
			monitor: Monitor{number: 1, left: 0, top: 0},
			wantX:   10, wantY: 20, wantWidth: 800, wantHeight: 600,
		},
		{
			name:    "secondary monitor to the right",
			setting: AppConfig{OffsetX: 5, OffsetY: 15, Width: 640, Height: 480},
			monitor: Monitor{number: 2, left: 1920, top: 0},
			wantX:   1925, wantY: 15, wantWidth: 640, wantHeight: 480,
		},
		{
			name:    "monitor with a negative origin",
			setting: AppConfig{OffsetX: 100, OffsetY: 50, Width: 1280, Height: 720},
			monitor: Monitor{number: 2, left: -1920, top: -120},
			wantX:   -1820, wantY: -70, wantWidth: 1280, wantHeight: 720,
		},
		{
			name:    "negative offsets to hide chrome off screen",
			setting: AppConfig{OffsetX: -8, OffsetY: -31, Width: 1936, Height: 1111},
			monitor: Monitor{number: 1, left: 0, top: 0},
			wantX:   -8, wantY: -31, wantWidth: 1936, wantHeight: 1111,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, width, height := borderlessRect(tt.setting, tt.monitor)
			if x != tt.wantX || y != tt.wantY || width != tt.wantWidth || height != tt.wantHeight {
				t.Errorf("borderlessRect() = (%d,%d) %dx%d, want (%d,%d) %dx%d",
					x, y, width, height, tt.wantX, tt.wantY, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

// Wine implements enough of user32 to link and run the suite, but SetWindowPos
// on a real window never returns there, so the integration tests below are
// skipped rather than left to hang.
func runningUnderWine() bool {
	return windows.NewLazySystemDLL("ntdll.dll").NewProc("wine_get_version").Find() == nil
}

// newTestWindow creates a real off-screen window so the Win32 side of
// makeBorderless/restoreWindow is exercised end to end. STATIC is a predefined
// class, so no class registration is needed.
func newTestWindow(t *testing.T, style uint32) Window {
	t.Helper()

	if runningUnderWine() {
		t.Skip("window manipulation is not reliable under wine")
	}

	className, err := windows.UTF16PtrFromString("STATIC")
	if err != nil {
		t.Fatalf("class name: %v", err)
	}
	title, err := windows.UTF16PtrFromString("GoBorderless Test Window")
	if err != nil {
		t.Fatalf("window title: %v", err)
	}

	hwnd := win.CreateWindowEx(0, className, title, style, 100, 100, 400, 300, 0, 0, 0, nil)
	if hwnd == 0 {
		t.Skip("cannot create windows in this environment")
	}
	t.Cleanup(func() { win.DestroyWindow(hwnd) })

	return Window{hwnd: hwnd, title: "GoBorderless Test Window", exePath: `C:\test.exe`}
}

func rectOf(t *testing.T, w Window) (x, y, width, height int32) {
	t.Helper()
	r := getWindowRect(w.hwnd)
	return r.Left, r.Top, r.Right - r.Left, r.Bottom - r.Top
}

func TestMakeBorderlessStripsFrameAndPositions(t *testing.T) {
	useMonitors(t, []Monitor{{number: 1, isPrimary: true, width: 1920, height: 1080}})
	window := newTestWindow(t, win.WS_OVERLAPPEDWINDOW)

	if isBorderless(window) {
		t.Fatal("test window did not start out bordered")
	}

	makeBorderless(window, AppConfig{Monitor: 1, OffsetX: 10, OffsetY: 20, Width: 800, Height: 600})

	if !isBorderless(window) {
		t.Error("window is still bordered after makeBorderless")
	}
	style := getWindowStyle(window.hwnd)
	if style&win.WS_CAPTION != 0 {
		t.Error("WS_CAPTION was not removed")
	}
	if style&win.WS_THICKFRAME != 0 {
		t.Error("WS_THICKFRAME was not removed")
	}

	x, y, w, h := rectOf(t, window)
	if x != 10 || y != 20 || w != 800 || h != 600 {
		t.Errorf("window at (%d,%d) %dx%d, want (10,20) 800x600", x, y, w, h)
	}
}

// Offsets are relative to the chosen monitor, so a config on a secondary
// display must land at that display's origin plus the offset.
func TestMakeBorderlessOffsetsFromChosenMonitor(t *testing.T) {
	useMonitors(t, []Monitor{
		{number: 1, isPrimary: true, width: 1920, height: 1080},
		{number: 2, width: 1920, height: 1080, left: 1920, top: -120},
	})
	window := newTestWindow(t, win.WS_OVERLAPPEDWINDOW)

	makeBorderless(window, AppConfig{Monitor: 2, OffsetX: 5, OffsetY: 15, Width: 640, Height: 480})

	x, y, w, h := rectOf(t, window)
	if x != 1925 || y != -105 || w != 640 || h != 480 {
		t.Errorf("window at (%d,%d) %dx%d, want (1925,-105) 640x480", x, y, w, h)
	}
}

func TestRestoreWindowRestoresFrameAndGeometry(t *testing.T) {
	useMonitors(t, []Monitor{{number: 1, isPrimary: true, width: 1920, height: 1080}})
	window := newTestWindow(t, win.WS_OVERLAPPEDWINDOW)

	origX, origY, origW, origH := rectOf(t, window)
	setting := AppConfig{AppMatcher: AppMatcher{PreOffsetX: origX, PreOffsetY: origY, PreWidth: origW, PreHeight: origH}, Monitor: 1, OffsetX: 0, OffsetY: 0, Width: 1920, Height: 1080}

	makeBorderless(window, setting)
	restoreWindow(window, setting)

	if isBorderless(window) {
		t.Error("window is still borderless after restoreWindow")
	}
	style := getWindowStyle(window.hwnd)
	if style&win.WS_CAPTION == 0 || style&win.WS_THICKFRAME == 0 {
		t.Errorf("frame styles not restored: %#x", uint32(style))
	}

	x, y, w, h := rectOf(t, window)
	if x != origX || y != origY || w != origW || h != origH {
		t.Errorf("window at (%d,%d) %dx%d, want the original (%d,%d) %dx%d", x, y, w, h, origX, origY, origW, origH)
	}
}

// Restore is a no-op on a window that still has its frame, so a stray click on
// Restore cannot teleport a window to stale saved geometry.
func TestRestoreWindowIgnoresBorderedWindow(t *testing.T) {
	window := newTestWindow(t, win.WS_OVERLAPPEDWINDOW)

	beforeX, beforeY, beforeW, beforeH := rectOf(t, window)
	beforeStyle := getWindowStyle(window.hwnd)

	restoreWindow(window, AppConfig{AppMatcher: AppMatcher{PreOffsetX: 999, PreOffsetY: 888, PreWidth: 111, PreHeight: 222}})

	x, y, w, h := rectOf(t, window)
	if x != beforeX || y != beforeY || w != beforeW || h != beforeH {
		t.Errorf("window moved to (%d,%d) %dx%d, want it left at (%d,%d) %dx%d", x, y, w, h, beforeX, beforeY, beforeW, beforeH)
	}
	if got := getWindowStyle(window.hwnd); got != beforeStyle {
		t.Errorf("style changed from %#x to %#x", uint32(beforeStyle), uint32(got))
	}
}

// Auto-apply calls makeBorderless once a second on an already-borderless
// window; repeating it must be stable.
func TestMakeBorderlessIsIdempotent(t *testing.T) {
	useMonitors(t, []Monitor{{number: 1, isPrimary: true, width: 1920, height: 1080}})
	window := newTestWindow(t, win.WS_OVERLAPPEDWINDOW)
	setting := AppConfig{Monitor: 1, OffsetX: 30, OffsetY: 40, Width: 1024, Height: 768}

	makeBorderless(window, setting)
	firstStyle := getWindowStyle(window.hwnd)
	x1, y1, w1, h1 := rectOf(t, window)

	makeBorderless(window, setting)

	if got := getWindowStyle(window.hwnd); got != firstStyle {
		t.Errorf("style drifted from %#x to %#x on the second apply", uint32(firstStyle), uint32(got))
	}
	x2, y2, w2, h2 := rectOf(t, window)
	if x1 != x2 || y1 != y2 || w1 != w2 || h1 != h2 {
		t.Errorf("geometry drifted from (%d,%d) %dx%d to (%d,%d) %dx%d", x1, y1, w1, h1, x2, y2, w2, h2)
	}
}

func TestIsBorderlessOnRealWindows(t *testing.T) {
	bordered := newTestWindow(t, win.WS_OVERLAPPEDWINDOW)
	if isBorderless(bordered) {
		t.Error("a WS_OVERLAPPEDWINDOW window should not be borderless")
	}

	popup := newTestWindow(t, win.WS_POPUP)
	if !isBorderless(popup) {
		t.Error("a WS_POPUP window should be borderless")
	}
}

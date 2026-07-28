package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// useMonitors swaps the package-level monitor list for the duration of a test.
func useMonitors(t *testing.T, mons []Monitor) {
	t.Helper()

	original := monitors
	monitors = mons
	t.Cleanup(func() { monitors = original })
}

func TestWindowString(t *testing.T) {
	w := Window{title: "Some Game", exePath: `C:\Games\game.exe`}

	if got, want := w.String(), `Some Game | C:\Games\game.exe`; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestMatchWindow(t *testing.T) {
	const (
		title = "Some Game"
		exe   = `C:\Games\game.exe`
	)
	window := Window{title: title, exePath: exe}

	tests := []struct {
		name      string
		matchType MatchType
		matcher   AppMatcher
		want      bool
	}{
		{"title matches", MatchWindowTitle, AppMatcher{WindowName: title, ExePath: `C:\other.exe`}, true},
		{"title differs", MatchWindowTitle, AppMatcher{WindowName: "Other", ExePath: exe}, false},
		{"exe matches", MatchExePath, AppMatcher{WindowName: "Other", ExePath: exe}, true},
		{"exe differs", MatchExePath, AppMatcher{WindowName: title, ExePath: `C:\other.exe`}, false},
		{"both match", MatchBoth, AppMatcher{WindowName: title, ExePath: exe}, true},
		{"both: only title", MatchBoth, AppMatcher{WindowName: title, ExePath: `C:\other.exe`}, false},
		{"both: only exe", MatchBoth, AppMatcher{WindowName: "Other", ExePath: exe}, false},
		{"either: only title", MatchEither, AppMatcher{WindowName: title, ExePath: `C:\other.exe`}, true},
		{"either: only exe", MatchEither, AppMatcher{WindowName: "Other", ExePath: exe}, true},
		{"either: neither", MatchEither, AppMatcher{WindowName: "Other", ExePath: `C:\other.exe`}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := tt.matcher
			matcher.MatchType = tt.matchType
			if got := matchWindow(window, matcher); got != tt.want {
				t.Errorf("matchWindow() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Matching is exact: a config saved for one window must not grab a similarly
// named one.
func TestMatchWindowIsExact(t *testing.T) {
	window := Window{title: "Game", exePath: `C:\Games\game.exe`}

	for _, matcher := range []AppMatcher{
		{WindowName: "game", MatchType: MatchWindowTitle},
		{WindowName: "Game ", MatchType: MatchWindowTitle},
		{WindowName: "Gam", MatchType: MatchWindowTitle},
		{ExePath: `c:\games\game.exe`, MatchType: MatchExePath},
		{ExePath: `C:\Games\`, MatchType: MatchExePath},
	} {
		if matchWindow(window, matcher) {
			t.Errorf("matchWindow() matched %+v, want no match", matcher)
		}
	}
}

func TestMatchWindowUnknownMatchType(t *testing.T) {
	window := Window{title: "Game", exePath: `C:\game.exe`}
	matcher := AppMatcher{WindowName: "Game", ExePath: `C:\game.exe`, MatchType: MatchType(99)}

	if matchWindow(window, matcher) {
		t.Error("an unrecognised match type must never match")
	}
}

// An empty matcher would otherwise match every untitled window, which is how a
// half-filled dialog could hijack unrelated windows.
func TestMatchWindowEmptySettingDoesNotMatchRealWindow(t *testing.T) {
	window := Window{title: "Game", exePath: `C:\game.exe`}

	for _, matchType := range []MatchType{MatchWindowTitle, MatchExePath, MatchBoth, MatchEither} {
		if matchWindow(window, AppMatcher{MatchType: matchType}) {
			t.Errorf("empty matcher with %v matched a real window", matchType)
		}
	}
}

func TestShouldHideWindow(t *testing.T) {
	tests := []struct {
		name        string
		processPath string
		title       string
		want        bool
	}{
		{"self", `C:\Tools\GoBorderless.exe`, "GoBorderless", true},
		{"other borderless tool", `C:\Tools\BorderlessGaming.exe`, "Borderless Gaming", true},
		{"system process", `C:\Windows\System32\svchost.exe`, "", true},
		{"explorer", `C:\Windows\explorer.exe`, "File Explorer", true},
		{"browser", `C:\Program Files\Google\Chrome\chrome.exe`, "Some Page", true},
		{"launcher", `C:\Program Files (x86)\Steam\steam.exe`, "Steam", true},
		{"streaming software", `D:\obs\bin\obs64.exe`, "OBS", true},
		{"uppercase extension", `C:\Windows\EXPLORER.EXE`, "", true},
		{"mixed case name", `C:\Windows\SvcHost.exe`, "", true},
		{"no extension", `C:\Windows\dwm`, "", true},
		{"blacklisted by title", `C:\Games\cod.exe`, "IW4 Console", true},
		{"blacklisted title, different case", `C:\Games\cod.exe`, "iw4 CONSOLE", true},
		{"ordinary game", `C:\Games\game.exe`, "Some Game", false},
		{"name merely contains a blacklisted word", `C:\Games\chrome-dino.exe`, "Dino", false},
		{"blacklisted word in a directory only", `C:\Program Files\Steam\steamapps\game.exe`, "Some Game", false},
		{"empty path and title", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldHideWindow(tt.processPath, tt.title); got != tt.want {
				t.Errorf("shouldHideWindow(%q, %q) = %v, want %v", tt.processPath, tt.title, got, tt.want)
			}
		})
	}
}

// The blacklist is compared against extension-stripped, lowercased names, so an
// entry that carries an extension or capitals can never match anything.
func TestBlacklistEntriesAreNormalised(t *testing.T) {
	for _, entry := range ALWAYS_HIDDEN_PROCESSESS {
		if entry != strings.ToLower(entry) {
			t.Errorf("blacklist entry %q is not lowercase; it can never match", entry)
		}
		if filepath.Ext(entry) != "" {
			t.Errorf("blacklist entry %q carries an extension; names are compared without one", entry)
		}
	}
}

func TestGetPrimaryMonitor(t *testing.T) {
	useMonitors(t, []Monitor{
		{number: 1, width: 1920, height: 1080},
		{number: 2, isPrimary: true, width: 2560, height: 1440},
	})

	if got := getPrimaryMonitor(); got.number != 2 {
		t.Errorf("getPrimaryMonitor() = %+v, want the monitor flagged primary", got)
	}
}

// Windows should always flag one display primary, but fall back rather than
// return a zero monitor (a zero width/height ends up saved as the default size).
func TestGetPrimaryMonitorFallsBackToFirst(t *testing.T) {
	useMonitors(t, []Monitor{
		{number: 1, width: 1920, height: 1080},
		{number: 2, width: 2560, height: 1440},
	})

	if got := getPrimaryMonitor(); got.number != 1 {
		t.Errorf("getPrimaryMonitor() = %+v, want the first monitor", got)
	}
}

func TestMonitorString(t *testing.T) {
	if got, want := (Monitor{number: 2}).String(), "Display 2"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got, want := (Monitor{number: 1, isPrimary: true}).String(), "Display 1 (Primary)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// Monitor numbers are 1-based because AppConfig.Monitor indexes monitors as
// Monitor-1; a 0-based number would silently shift every saved config.
func TestGetMonitorsNumbersAreOneBased(t *testing.T) {
	mons := getMonitors()
	if len(mons) == 0 {
		t.Skip("no monitors reported in this environment")
	}

	for i, mon := range mons {
		if mon.number != i+1 {
			t.Errorf("monitor at index %d has number %d, want %d", i, mon.number, i+1)
		}
		if mon.width <= 0 || mon.height <= 0 {
			t.Errorf("monitor %d has a zero dimension: %+v", mon.number, mon)
		}
	}
}

// Smoke test over the real enumeration path: whatever it returns must already
// satisfy the filters getWindowData applies, so the UI never shows junk.
func TestGetWindowDataAppliesItsFilters(t *testing.T) {
	windowData := getWindowData(EnumWindows())
	if len(windowData) == 0 {
		t.Skip("no visible windows in this environment")
	}

	for _, w := range windowData {
		if w.title == "" {
			t.Error("returned a window with no title")
		}
		if w.exePath == "" {
			t.Errorf("window %q has no executable path", w.title)
		}
		if !isVisible(uintptr(w.hwnd)) {
			t.Errorf("window %q is not visible", w.title)
		}
		if shouldHideWindow(w.exePath, w.title) {
			t.Errorf("window %q (%s) should have been filtered out", w.title, w.exePath)
		}
		rect := getWindowRect(w.hwnd)
		if rect.Left == rect.Right && rect.Top == rect.Bottom {
			t.Errorf("window %q has no size", w.title)
		}
	}
}

func TestEnumWindowsReturnsStableCounts(t *testing.T) {
	first := EnumWindows()
	second := EnumWindows()

	if len(first) == 0 && len(second) == 0 {
		t.Skip("no windows reported in this environment")
	}
	// Guards the append-through-unsafe.Pointer callback that used to drop every
	// window on the first pass (see other/test_enum_window_slice_issue.go).
	if len(first) == 0 || len(second) == 0 {
		t.Errorf("EnumWindows returned %d then %d windows; enumeration is dropping results", len(first), len(second))
	}
}

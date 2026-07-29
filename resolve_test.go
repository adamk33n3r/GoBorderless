package main

import "testing"

func TestResolveApplyAppConfigBeatsLayout(t *testing.T) {
	const (
		title = "Some Game"
		exe   = `C:\Games\game.exe`
	)
	win := Window{title: title, exePath: exe}
	settings := &Settings{
		Apps: []AppConfig{{
			AppMatcher:   AppMatcher{WindowName: title, ExePath: exe, MatchType: MatchBoth},
			Monitor:      1,
			OffsetX:      10,
			OffsetY:      20,
			Width:        800,
			Height:       600,
			BlackOverlay: true,
			HideTaskbar:  true,
		}},
		Layouts: []LayoutConfig{{
			Name:         "Layout",
			Monitor:      2,
			OffsetX:      1,
			OffsetY:      2,
			Width:        100,
			Height:       100,
			BlackOverlay: false,
			HideTaskbar:  false,
			Matchers: []AppMatcher{{
				WindowName: title,
				ExePath:    exe,
				MatchType:  MatchBoth,
			}},
		}},
	}

	got, ok := resolveApply(settings, win)
	if !ok {
		t.Fatal("resolveApply() returned no match")
	}
	want := ApplyPayload{
		Monitor: 1, OffsetX: 10, OffsetY: 20, Width: 800, Height: 600,
		BlackOverlay: true, HideTaskbar: true,
	}
	if got != want {
		t.Errorf("resolveApply() = %+v, want App Config payload %+v", got, want)
	}
}

func TestResolveApplyLayoutListOrder(t *testing.T) {
	const (
		title = "Some Game"
		exe   = `C:\Games\game.exe`
	)
	win := Window{title: title, exePath: exe}
	matcher := AppMatcher{WindowName: title, ExePath: exe, MatchType: MatchBoth}
	settings := &Settings{
		Layouts: []LayoutConfig{
			{Name: "First", Monitor: 1, OffsetX: 10, Width: 800, Height: 600, Matchers: []AppMatcher{matcher}},
			{Name: "Second", Monitor: 2, OffsetX: 99, Width: 100, Height: 100, Matchers: []AppMatcher{matcher}},
		},
	}

	got, ok := resolveApply(settings, win)
	if !ok {
		t.Fatal("resolveApply() returned no match")
	}
	if got.Monitor != 1 || got.OffsetX != 10 {
		t.Errorf("resolveApply() = monitor=%d offsetX=%d, want first layout (1, 10)", got.Monitor, got.OffsetX)
	}
}

func TestResolveApplyFirstMatcherAfterTitleSort(t *testing.T) {
	const exe = `C:\Games\game.exe`
	win := Window{title: "Live", exePath: exe}
	settings := &Settings{
		Layouts: []LayoutConfig{{
			Name:    "Layout",
			Monitor: 1,
			OffsetX: 0,
			Width:   1920,
			Height:  1080,
			// Unsorted on purpose — all three match by exe; Alpha wins after title sort.
			Matchers: []AppMatcher{
				{WindowName: "Zulu", ExePath: exe, MatchType: MatchExePath, PreOffsetX: 9},
				{WindowName: "Bravo", ExePath: exe, MatchType: MatchExePath, PreOffsetX: 2},
				{WindowName: "Alpha", ExePath: exe, MatchType: MatchExePath, PreOffsetX: 1},
			},
		}},
	}

	got, ok := resolveApply(settings, win)
	if !ok {
		t.Fatal("resolveApply() returned no match")
	}
	if got.PreOffsetX != 1 {
		t.Errorf("resolveApply() PreOffsetX = %d, want Alpha matcher (1)", got.PreOffsetX)
	}
}

func TestResolveApplyNoMatch(t *testing.T) {
	win := Window{title: "Other", exePath: `C:\other.exe`}
	settings := &Settings{
		Apps: []AppConfig{{
			AppMatcher: AppMatcher{WindowName: "Game", MatchType: MatchWindowTitle},
			Monitor:    1,
		}},
		Layouts: []LayoutConfig{{
			Name:     "Layout",
			Matchers: []AppMatcher{{WindowName: "Game", MatchType: MatchWindowTitle}},
		}},
	}

	if _, ok := resolveApply(settings, win); ok {
		t.Error("resolveApply() matched an unrelated window")
	}
}

func TestResolveWinnerReturnsLayoutMatcher(t *testing.T) {
	const (
		title = "Some Game"
		exe   = `C:\Games\game.exe`
	)
	win := Window{title: title, exePath: exe}
	matcher := AppMatcher{
		WindowName: title,
		ExePath:    exe,
		MatchType:  MatchBoth,
		AutoApply:  true,
		PreOffsetX: 5,
	}
	settings := &Settings{
		Layouts: []LayoutConfig{{
			Name:         "Ultrawide",
			Monitor:      2,
			OffsetX:      10,
			OffsetY:      20,
			Width:        3440,
			Height:       1440,
			BlackOverlay: true,
			HideTaskbar:  true,
			Matchers:     []AppMatcher{matcher},
		}},
	}

	winner, ok := resolveWinner(settings, win)
	if !ok {
		t.Fatal("resolveWinner() returned no match")
	}
	if winner.appIdx >= 0 {
		t.Fatalf("resolveWinner() appIdx = %d, want layout winner", winner.appIdx)
	}
	if winner.layoutIdx != 0 {
		t.Errorf("layoutIdx = %d, want 0", winner.layoutIdx)
	}
	if winner.matcher != matcher {
		t.Errorf("matcher = %+v, want %+v", winner.matcher, matcher)
	}
	want := ApplyPayload{
		Monitor: 2, OffsetX: 10, OffsetY: 20, Width: 3440, Height: 1440,
		BlackOverlay: true, HideTaskbar: true,
		PreOffsetX: 5,
	}
	if winner.payload != want {
		t.Errorf("payload = %+v, want %+v", winner.payload, want)
	}
}

func TestResolveLayoutAutoApply(t *testing.T) {
	const (
		gameTitle = "Some Game"
		gameExe   = `C:\Games\game.exe`
		calcTitle = "Calculator"
		calcExe   = `C:\Windows\System32\calc.exe`
	)
	gameWin := Window{title: gameTitle, exePath: gameExe}
	calcWin := Window{title: calcTitle, exePath: calcExe}
	calcMatcher := AppMatcher{WindowName: calcTitle, ExePath: calcExe, MatchType: MatchBoth}

	tests := []struct {
		name           string
		settings       *Settings
		win            Window
		wantOK         bool
		wantLayoutIdx  int
		wantMatcherIdx int
	}{
		{
			name: "skips manual layout higher in list",
			settings: &Settings{
				Layouts: []LayoutConfig{
					{Name: "Manual", Monitor: 1, Matchers: []AppMatcher{calcMatcher}},
					{Name: "Auto", Monitor: 2, Matchers: []AppMatcher{
						{WindowName: calcTitle, ExePath: calcExe, MatchType: MatchBoth, AutoApply: true},
					}},
				},
			},
			win: calcWin, wantOK: true, wantLayoutIdx: 1, wantMatcherIdx: 0,
		},
		{
			name: "skips manual matcher in same layout",
			settings: &Settings{
				Layouts: []LayoutConfig{{
					Name: "Layout", Monitor: 1,
					Matchers: []AppMatcher{
						{WindowName: "Alpha", ExePath: gameExe, MatchType: MatchExePath},
						{WindowName: "Bravo", ExePath: gameExe, MatchType: MatchExePath, AutoApply: true},
					},
				}},
			},
			win: gameWin, wantOK: true, wantLayoutIdx: 0, wantMatcherIdx: 1,
		},
		{
			name: "blocked by auto-applying app config",
			settings: &Settings{
				Apps: []AppConfig{{
					AppMatcher: AppMatcher{WindowName: gameTitle, ExePath: gameExe, MatchType: MatchBoth, AutoApply: true},
					Monitor:    1,
				}},
				Layouts: []LayoutConfig{{
					Name: "Layout", Monitor: 2,
					Matchers: []AppMatcher{{WindowName: gameTitle, ExePath: gameExe, MatchType: MatchBoth, AutoApply: true}},
				}},
			},
			win: gameWin, wantOK: false,
		},
		{
			name: "first auto layout in list order wins",
			settings: &Settings{
				Layouts: []LayoutConfig{
					{Name: "First", Monitor: 1, Matchers: []AppMatcher{
						{WindowName: gameTitle, ExePath: gameExe, MatchType: MatchBoth, AutoApply: true},
					}},
					{Name: "Second", Monitor: 2, Matchers: []AppMatcher{
						{WindowName: gameTitle, ExePath: gameExe, MatchType: MatchBoth, AutoApply: true},
					}},
				},
			},
			win: gameWin, wantOK: true, wantLayoutIdx: 0, wantMatcherIdx: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layoutIdx, matcherIdx, ok := resolveLayoutAutoApply(tt.settings, tt.win)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if layoutIdx != tt.wantLayoutIdx || matcherIdx != tt.wantMatcherIdx {
				t.Errorf("got (%d, %d), want (%d, %d)", layoutIdx, matcherIdx, tt.wantLayoutIdx, tt.wantMatcherIdx)
			}
		})
	}
}

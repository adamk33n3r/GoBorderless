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

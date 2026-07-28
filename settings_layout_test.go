package main

import "testing"

func TestAddLayoutPreservesInsertionOrder(t *testing.T) {
	settings := newSettings()

	for _, name := range []string{"Mid", "Zed", "Alpha"} {
		settings.AddLayout(LayoutConfig{Name: name, Matchers: []AppMatcher{}})
	}

	want := []string{"Mid", "Zed", "Alpha"}
	for i, name := range want {
		if settings.Layouts[i].Name != name {
			t.Errorf("layout %d = %q, want %q (insertion order, not alpha)", i, settings.Layouts[i].Name, name)
		}
	}
}

func TestRemoveLayout(t *testing.T) {
	settings := newSettings()
	for _, name := range []string{"First", "Second", "Third"} {
		settings.AddLayout(LayoutConfig{Name: name, Matchers: []AppMatcher{}})
	}

	settings.RemoveLayout(1)

	if len(settings.Layouts) != 2 {
		t.Fatalf("have %d layouts, want 2", len(settings.Layouts))
	}
	if settings.Layouts[0].Name != "First" || settings.Layouts[1].Name != "Third" {
		t.Errorf("after removing index 1 got %q and %q", settings.Layouts[0].Name, settings.Layouts[1].Name)
	}
}

func TestMoveLayoutUpDown(t *testing.T) {
	settings := newSettings()
	for _, name := range []string{"A", "B", "C"} {
		settings.AddLayout(LayoutConfig{Name: name, Matchers: []AppMatcher{}})
	}

	settings.MoveLayout(2, -1) // C up -> A, C, B
	if got := []string{settings.Layouts[0].Name, settings.Layouts[1].Name, settings.Layouts[2].Name}; got[0] != "A" || got[1] != "C" || got[2] != "B" {
		t.Errorf("after MoveLayout(2,-1) got %v, want [A C B]", got)
	}

	settings.MoveLayout(1, -1) // C up -> C, A, B
	if got := []string{settings.Layouts[0].Name, settings.Layouts[1].Name, settings.Layouts[2].Name}; got[0] != "C" || got[1] != "A" || got[2] != "B" {
		t.Errorf("after MoveLayout(1,-1) got %v, want [C A B]", got)
	}

	settings.MoveLayout(0, -1) // no-op at top
	if settings.Layouts[0].Name != "C" {
		t.Errorf("MoveLayout(0,-1) changed top; got %q", settings.Layouts[0].Name)
	}

	settings.MoveLayout(0, 1) // C down -> A, C, B
	if got := []string{settings.Layouts[0].Name, settings.Layouts[1].Name, settings.Layouts[2].Name}; got[0] != "A" || got[1] != "C" || got[2] != "B" {
		t.Errorf("after MoveLayout(0,1) got %v, want [A C B]", got)
	}

	settings.MoveLayout(2, 1) // no-op at bottom
	if settings.Layouts[2].Name != "B" {
		t.Errorf("MoveLayout(2,1) changed bottom; got %q", settings.Layouts[2].Name)
	}
}

func TestMoveLayoutPersistsOrderAcrossSave(t *testing.T) {
	useTempSettings(t)

	settings := newSettings()
	for _, name := range []string{"A", "B", "C"} {
		settings.AddLayout(LayoutConfig{Name: name, Matchers: []AppMatcher{}})
	}
	settings.MoveLayout(0, 1) // B, A, C
	if err := settings.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := loadSettings()
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	want := []string{"B", "A", "C"}
	for i, name := range want {
		if loaded.Layouts[i].Name != name {
			t.Errorf("layout %d = %q, want %q", i, loaded.Layouts[i].Name, name)
		}
	}
}

func TestNewLayoutFromDefaults(t *testing.T) {
	useMonitors(t, []Monitor{{number: 1, isPrimary: true, width: 1920, height: 1080}})
	settings := newSettings()
	settings.Defaults = AppConfigDefaults{
		Monitor: 2, OffsetX: 5, OffsetY: 6, Width: 100, Height: 200,
		BlackOverlay: true, HideTaskbar: true, MatchType: MatchExePath,
	}

	got := newLayoutFromDefaults(settings)
	if got.Name != "" {
		t.Errorf("Name = %q, want empty", got.Name)
	}
	if got.Monitor != 2 || got.OffsetX != 5 || got.OffsetY != 6 || got.Width != 100 || got.Height != 200 {
		t.Errorf("geometry = %+v", got)
	}
	if !got.BlackOverlay || !got.HideTaskbar {
		t.Errorf("flags BlackOverlay=%v HideTaskbar=%v", got.BlackOverlay, got.HideTaskbar)
	}
	if got.Matchers == nil || len(got.Matchers) != 0 {
		t.Errorf("Matchers = %+v, want empty non-nil", got.Matchers)
	}
}

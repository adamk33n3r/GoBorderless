package main

import "testing"

func TestFirstInSliceFindsMatch(t *testing.T) {
	windows := []Window{
		{title: "Alpha", exePath: `C:\a.exe`},
		{title: "Beta", exePath: `C:\b.exe`},
	}

	got := firstInSlice(windows, func(w Window) bool { return w.title == "Beta" })

	if got == nil {
		t.Fatal("firstInSlice() = nil, want the Beta window")
	}
	if got.title != "Beta" {
		t.Errorf("found %q, want Beta", got.title)
	}
}

// The apply/restore buttons use the returned pointer to read live window state,
// so it must point into the caller's slice rather than at a copy.
func TestFirstInSliceReturnsPointerIntoSlice(t *testing.T) {
	windows := []Window{{title: "Alpha"}, {title: "Beta"}}

	got := firstInSlice(windows, func(w Window) bool { return w.title == "Beta" })
	if got != &windows[1] {
		t.Fatalf("returned %p, want &slice[1] (%p)", got, &windows[1])
	}

	got.title = "Changed"
	if windows[1].title != "Changed" {
		t.Error("writing through the pointer did not affect the slice")
	}
}

func TestFirstInSliceReturnsFirstOfSeveralMatches(t *testing.T) {
	windows := []Window{
		{title: "Same", exePath: `C:\first.exe`},
		{title: "Same", exePath: `C:\second.exe`},
	}

	got := firstInSlice(windows, func(w Window) bool { return w.title == "Same" })

	if got == nil || got.exePath != `C:\first.exe` {
		t.Errorf("got %+v, want the first match", got)
	}
}

func TestFirstInSliceNoMatch(t *testing.T) {
	windows := []Window{{title: "Alpha"}}

	if got := firstInSlice(windows, func(w Window) bool { return w.title == "Missing" }); got != nil {
		t.Errorf("firstInSlice() = %+v, want nil", got)
	}
}

func TestFirstInSliceEmptyAndNil(t *testing.T) {
	always := func(Window) bool { return true }

	if got := firstInSlice([]Window{}, always); got != nil {
		t.Errorf("empty slice returned %+v, want nil", got)
	}
	if got := firstInSlice([]Window(nil), always); got != nil {
		t.Errorf("nil slice returned %+v, want nil", got)
	}
}

func TestFirstInSliceWorksWithOtherTypes(t *testing.T) {
	mons := []Monitor{{number: 1}, {number: 2, isPrimary: true}}

	got := firstInSlice(mons, func(m Monitor) bool { return m.isPrimary })

	if got == nil || got.number != 2 {
		t.Errorf("got %+v, want monitor 2", got)
	}
}

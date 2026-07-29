package main

import "testing"

func TestLayoutApplyRestoreEnabled(t *testing.T) {
	tests := []struct {
		name     string
		matchers []AppMatcher
		want     bool
	}{
		{"empty matchers", nil, false},
		{"empty slice", []AppMatcher{}, false},
		{"all auto", []AppMatcher{{AutoApply: true}, {AutoApply: true}}, false},
		{"one manual", []AppMatcher{{AutoApply: true}, {AutoApply: false}}, true},
		{"all manual", []AppMatcher{{AutoApply: false}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := LayoutConfig{Matchers: tt.matchers}
			if got := layoutApplyRestoreEnabled(layout); got != tt.want {
				t.Errorf("layoutApplyRestoreEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLayoutGeometrySubtitle(t *testing.T) {
	layout := LayoutConfig{
		Monitor: 2,
		OffsetX: 10,
		OffsetY: 20,
		Width:   3440,
		Height:  1440,
	}
	want := "Monitor 2 · 10,20 3440×1440"
	if got := layout.GeometrySubtitle(); got != want {
		t.Errorf("GeometrySubtitle() = %q, want %q", got, want)
	}
}

func TestLayoutMatcherCountLabel(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0 matchers"},
		{1, "1 matcher"},
		{2, "2 matchers"},
	}
	for _, tt := range tests {
		layout := LayoutConfig{Matchers: make([]AppMatcher, tt.n)}
		if got := layout.MatcherCountLabel(); got != tt.want {
			t.Errorf("MatcherCountLabel(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestLayoutNameValid(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Ultrawide", true},
		{"  Ultrawide  ", true},
		{"", false},
		{"   ", false},
	}
	for _, tt := range tests {
		if got := layoutNameValid(tt.name); got != tt.want {
			t.Errorf("layoutNameValid(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestCapturePreGeometry(t *testing.T) {
	matcher := AppMatcher{}
	rect := struct{ Left, Top, Right, Bottom int32 }{10, 20, 810, 620}
	capturePreGeometry(&matcher, rect.Left, rect.Top, rect.Right, rect.Bottom)
	if matcher.PreOffsetX != 10 || matcher.PreOffsetY != 20 || matcher.PreWidth != 800 || matcher.PreHeight != 600 {
		t.Errorf("matcher Pre* = %+v", matcher)
	}
}

func TestFindMatcherIndex(t *testing.T) {
	matchers := []AppMatcher{
		{WindowName: "Alpha", ExePath: `C:\a.exe`},
		{WindowName: "Bravo", ExePath: `C:\b.exe`},
	}
	if got := findMatcherIndex(matchers, AppMatcher{WindowName: "Bravo", ExePath: `C:\b.exe`}); got != 1 {
		t.Errorf("findMatcherIndex() = %d, want 1", got)
	}
	if got := findMatcherIndex(matchers, AppMatcher{WindowName: "Missing", ExePath: `C:\x.exe`}); got != -1 {
		t.Errorf("findMatcherIndex() = %d, want -1", got)
	}
}

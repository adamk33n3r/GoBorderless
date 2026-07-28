package ui

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestNewLayoutConfigRowBuildsEveryControl(t *testing.T) {
	row := NewLayoutConfigRow()

	if row.Title == nil {
		t.Fatal("Title is nil")
	}
	if row.Subtitle == nil {
		t.Fatal("Subtitle is nil")
	}
	if row.MatcherCount == nil {
		t.Fatal("MatcherCount is nil")
	}
	for name, btn := range map[string]any{
		"UpBtn":      row.UpBtn,
		"DownBtn":    row.DownBtn,
		"ApplyBtn":   row.ApplyBtn,
		"RestoreBtn": row.RestoreBtn,
		"EditBtn":    row.EditBtn,
		"DeleteBtn":  row.DeleteBtn,
	} {
		if btn == nil {
			t.Errorf("%s is nil", name)
		}
	}
}

func TestLayoutConfigRowTitleTruncates(t *testing.T) {
	row := NewLayoutConfigRow()
	if row.Title.Truncation != fyne.TextTruncateEllipsis {
		t.Errorf("Title Truncation = %v, want TextTruncateEllipsis", row.Title.Truncation)
	}
	if row.Subtitle.Truncation != fyne.TextTruncateEllipsis {
		t.Errorf("Subtitle Truncation = %v, want TextTruncateEllipsis", row.Subtitle.Truncation)
	}
}

func TestLayoutConfigRowButtonToolTips(t *testing.T) {
	row := NewLayoutConfigRow()
	tips := map[string]string{
		"Move up":   row.UpBtn.ToolTip(),
		"Move down": row.DownBtn.ToolTip(),
		"Apply":     row.ApplyBtn.ToolTip(),
		"Restore":   row.RestoreBtn.ToolTip(),
		"Edit":      row.EditBtn.ToolTip(),
		"Delete":    row.DeleteBtn.ToolTip(),
	}
	for want, got := range tips {
		if got != want {
			t.Errorf("tooltip = %q, want %q", got, want)
		}
	}
}

func TestLayoutConfigRowCreateRenderer(t *testing.T) {
	row := NewLayoutConfigRow()
	renderer := row.CreateRenderer()
	if renderer == nil {
		t.Fatal("CreateRenderer returned nil")
	}
	if len(renderer.Objects()) == 0 {
		t.Error("renderer has no objects")
	}
	if row.MinSize().IsZero() {
		t.Error("MinSize is zero; row would collapse in the list")
	}
}

func TestLayoutConfigRowTappedIsInert(t *testing.T) {
	row := NewLayoutConfigRow()
	row.Tapped(&fyne.PointEvent{})
}

func TestLayoutConfigRowDefaultCallbacksAreSafe(t *testing.T) {
	row := NewLayoutConfigRow()
	row.UpBtn.OnTapped()
	row.DownBtn.OnTapped()
	row.ApplyBtn.OnTapped()
	row.RestoreBtn.OnTapped()
	row.EditBtn.OnTapped()
	row.DeleteBtn.OnTapped()
}

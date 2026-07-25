package ui

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestNewAppSettingRowBuildsEveryControl(t *testing.T) {
	row := NewAppSettingRow()

	if row.Title == nil {
		t.Fatal("Title is nil")
	}
	if row.AutoApply == nil {
		t.Fatal("AutoApply is nil")
	}
	for name, btn := range map[string]any{
		"ApplyBtn":   row.ApplyBtn,
		"RestoreBtn": row.RestoreBtn,
		"EditBtn":    row.EditBtn,
		"DeleteBtn":  row.DeleteBtn,
	} {
		if btn == nil {
			t.Errorf("%s is nil", name)
		}
	}

	if row.AutoApply.Text != "Auto Apply" {
		t.Errorf("AutoApply label = %q, want %q", row.AutoApply.Text, "Auto Apply")
	}
	if row.AutoApply.Checked {
		t.Error("AutoApply should start unchecked")
	}
}

// Long "title | C:\path\to\game.exe" strings must ellipsize rather than push the
// action buttons out of the row.
func TestAppSettingRowTitleTruncates(t *testing.T) {
	row := NewAppSettingRow()

	if row.Title.Truncation != fyne.TextTruncateEllipsis {
		t.Errorf("Truncation = %v, want TextTruncateEllipsis", row.Title.Truncation)
	}
}

func TestAppSettingRowButtonToolTips(t *testing.T) {
	row := NewAppSettingRow()

	tips := map[string]string{
		"Apply":   row.ApplyBtn.ToolTip(),
		"Restore": row.RestoreBtn.ToolTip(),
		"Edit":    row.EditBtn.ToolTip(),
		"Delete":  row.DeleteBtn.ToolTip(),
	}
	for want, got := range tips {
		if got != want {
			t.Errorf("tooltip = %q, want %q", got, want)
		}
	}
}

func TestAppSettingRowCreateRenderer(t *testing.T) {
	row := NewAppSettingRow()

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

// The row is used as a list template, so tapping it must be inert (selection is
// handled by the enclosing widget.List).
func TestAppSettingRowTappedIsInert(t *testing.T) {
	row := NewAppSettingRow()

	row.Tapped(&fyne.PointEvent{}) // must not panic
}

// Callbacks are rebound by the list's update function on every reuse; the
// defaults must exist so an unbound row cannot nil-panic when clicked.
func TestAppSettingRowDefaultCallbacksAreSafe(t *testing.T) {
	row := NewAppSettingRow()

	row.ApplyBtn.OnTapped()
	row.RestoreBtn.OnTapped()
	row.EditBtn.OnTapped()
	row.DeleteBtn.OnTapped()
	row.AutoApply.OnChanged(true)
}

func TestNewMuxerPanel(t *testing.T) {
	if NewMuxerPanel() == nil {
		t.Fatal("NewMuxerPanel returned nil")
	}
}

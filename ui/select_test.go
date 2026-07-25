package ui

import (
	"fmt"
	"os"
	"testing"

	"fyne.io/fyne/v2/test"
)

// The widgets refresh themselves on every change, which needs a running app to
// resolve the theme, so stand up Fyne's headless test app for the package.
func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

// label stands in for the app's Window type: a plain Stringer option.
type label string

func (l label) String() string { return string(l) }

// display mirrors the shape of the app's Monitor type (a struct option whose
// String() is not just one of its fields).
type display struct {
	number  int
	primary bool
}

func (d display) String() string {
	if d.primary {
		return fmt.Sprintf("Display %d (Primary)", d.number)
	}
	return fmt.Sprintf("Display %d", d.number)
}

func TestNewSelectStringifiesOptions(t *testing.T) {
	s := NewSelect([]label{"beta", "alpha"}, func(label) {})

	want := []string{"beta", "alpha"}
	if got := s.Select.Options; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("underlying options = %v, want %v", got, want)
	}
	if s.Selected != nil {
		t.Errorf("Selected = %v, want nil before any selection", *s.Selected)
	}
}

func TestNewSelectWorksWithStructOptions(t *testing.T) {
	monitors := []display{{number: 1, primary: true}, {number: 2}}
	s := NewSelect(monitors, func(display) {})

	if got := s.Select.Options[0]; got != "Display 1 (Primary)" {
		t.Errorf("option 0 = %q, want %q", got, "Display 1 (Primary)")
	}
	if got := s.Select.Options[1]; got != "Display 2" {
		t.Errorf("option 1 = %q, want %q", got, "Display 2")
	}
}

func TestSetSelectedNotifiesAndTracksValue(t *testing.T) {
	var got []label
	s := NewSelect([]label{"alpha", "beta"}, func(l label) { got = append(got, l) })

	s.SetSelected("beta")

	if len(got) != 1 || got[0] != "beta" {
		t.Errorf("callback received %v, want [beta]", got)
	}
	if s.Selected == nil || *s.Selected != "beta" {
		t.Errorf("Selected = %v, want beta", s.Selected)
	}
	if s.Select.Selected != "beta" {
		t.Errorf("underlying Selected = %q, want %q", s.Select.Selected, "beta")
	}
}

// SetSelectedIndex is what the app uses to preselect the configured monitor, and
// it must leave Selected pointing into the live Options slice.
func TestSetSelectedIndexPointsIntoOptions(t *testing.T) {
	var calls int
	options := []display{{number: 1, primary: true}, {number: 2}}
	s := NewSelect(options, func(display) { calls++ })

	s.SetSelectedIndex(1)

	if s.Selected == nil {
		t.Fatal("Selected = nil, want option 1")
	}
	if s.Selected != &s.Options[1] {
		t.Errorf("Selected does not point into Options; got %p want %p", s.Selected, &s.Options[1])
	}
	if s.Select.Selected != "Display 2" {
		t.Errorf("underlying Selected = %q, want %q", s.Select.Selected, "Display 2")
	}
	if calls != 1 {
		t.Errorf("callback fired %d times, want 1", calls)
	}
}

func TestSetSelectedIndexOutOfRangeIsIgnored(t *testing.T) {
	var calls int
	s := NewSelect([]label{"alpha", "beta"}, func(label) { calls++ })

	for _, idx := range []int{-1, 2, 99} {
		s.SetSelectedIndex(idx)
		if s.Selected != nil {
			t.Errorf("SetSelectedIndex(%d) selected %v, want no selection", idx, *s.Selected)
			s.Selected = nil
		}
	}
	if calls != 0 {
		t.Errorf("callback fired %d times for out-of-range indexes, want 0", calls)
	}
}

func TestSetSelectedIndexOnEmptyOptions(t *testing.T) {
	s := NewSelect([]label{}, func(label) {})

	s.SetSelectedIndex(0) // must not panic

	if s.Selected != nil {
		t.Errorf("Selected = %v, want nil", *s.Selected)
	}
}

func TestClearSelected(t *testing.T) {
	var calls int
	s := NewSelect([]label{"alpha", "beta"}, func(label) { calls++ })
	s.SetSelectedIndex(0)
	calls = 0

	s.ClearSelected()

	if s.Selected != nil {
		t.Errorf("Selected = %v, want nil", *s.Selected)
	}
	if s.Select.Selected != "" {
		t.Errorf("underlying Selected = %q, want empty", s.Select.Selected)
	}
	if calls != 0 {
		t.Errorf("callback fired %d times while clearing, want 0", calls)
	}
}

// The new-config dialog swaps in a fresh window list every second; the widget's
// string options must follow, and a selection that vanished must not linger.
func TestSetOptionsReplacesBothViews(t *testing.T) {
	s := NewSelect([]label{"alpha", "beta"}, func(label) {})
	s.SetSelectedIndex(0)

	s.SetOptions([]label{"gamma"})

	if len(s.Options) != 1 || s.Options[0] != "gamma" {
		t.Errorf("typed options = %v, want [gamma]", s.Options)
	}
	if len(s.Select.Options) != 1 || s.Select.Options[0] != "gamma" {
		t.Errorf("underlying options = %v, want [gamma]", s.Select.Options)
	}
}

// Fyne drives OnChanged with a raw string when the user picks from the popup.
// Anything not in the current option list must clear the selection instead of
// leaving a stale pointer behind.
func TestOnChangedWithUnknownStringClearsSelection(t *testing.T) {
	var calls int
	s := NewSelect([]label{"alpha"}, func(label) { calls++ })
	s.SetSelectedIndex(0)
	calls = 0

	s.OnChanged("this option no longer exists")

	if s.Selected != nil {
		t.Errorf("Selected = %v, want nil", *s.Selected)
	}
	if s.Select.Selected != "" {
		t.Errorf("underlying Selected = %q, want empty", s.Select.Selected)
	}
	if calls != 0 {
		t.Errorf("callback fired %d times for an unknown option, want 0", calls)
	}
}

// Two windows can share a title and executable, so options are not guaranteed
// unique. Selecting by index keeps the index the caller asked for, even though
// the string round-trip through the widget can only find the first match.
func TestDuplicateOptionsKeepRequestedIndex(t *testing.T) {
	s := NewSelect([]label{"same", "same"}, func(label) {})

	s.SetSelectedIndex(1)

	if s.Selected != &s.Options[1] {
		t.Errorf("Selected = %p, want the requested index %p", s.Selected, &s.Options[1])
	}
}

// ...whereas a pick from the popup only carries the display string, so it can
// only ever resolve to the first duplicate.
func TestDuplicateOptionsFromPopupResolveToFirstMatch(t *testing.T) {
	s := NewSelect([]label{"same", "same"}, func(label) {})

	s.OnChanged("same")

	if s.Selected != &s.Options[0] {
		t.Errorf("Selected = %p, want first duplicate %p", s.Selected, &s.Options[0])
	}
}

// Documents current behaviour: SetSelected trusts its argument, so an option
// that is not in the list leaves the typed Selected set while the widget itself
// shows nothing. Callers should prefer SetSelectedIndex.
func TestSetSelectedWithUnknownOptionIsNotReflectedInWidget(t *testing.T) {
	s := NewSelect([]label{"alpha"}, func(label) {})

	s.SetSelected("not an option")

	if s.Selected == nil || *s.Selected != "not an option" {
		t.Errorf("Selected = %v, want the value passed in", s.Selected)
	}
	if s.Select.Selected != "" {
		t.Errorf("underlying Selected = %q, want empty; widget should reject unknown options", s.Select.Selected)
	}
}

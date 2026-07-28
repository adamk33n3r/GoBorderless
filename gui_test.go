package main

import (
	"errors"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/adamk33n3r/GoBorderless/ui"
	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Widgets refresh themselves on construction, which needs an app for theme
// lookups. Fyne's test app is headless.
func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

func TestMakeLayoutConfigWindowDoesNotPanicOnCreate(t *testing.T) {
	useMonitors(t, []Monitor{{number: 1, isPrimary: true, width: 1920, height: 1080}})
	settings := newSettings()
	settings.Defaults = AppConfigDefaults{
		Monitor: 1, Width: 1920, Height: 1080,
	}
	parent := test.NewTempWindow(t, nil)

	// SetSelectedIndex fires OnChanged during construction; that must not
	// call a still-nil setConfirmState (regression from the Create New crash).
	dialog := makeLayoutConfigWindow(settings, newLayoutFromDefaults(settings), true, -1, parent, func(*LayoutConfig) {})
	if dialog == nil {
		t.Fatal("makeLayoutConfigWindow returned nil")
	}
	dialog.Hide()
}

// fyne-tooltip requires AddPopUpToolTipLayer on the dialog's ModalPopUp before
// Show (and a window tool tip layer on the parent canvas first). Calling it
// after Show or re-centering Content desyncs the PopUp background from the
// dialog card.
func TestCustomDialogPopUpAllowsTooltipLayerBeforeShow(t *testing.T) {
	parent := test.NewTempWindow(t, nil)
	parent.SetContent(fynetooltip.AddWindowToolTipLayer(widget.NewLabel(""), parent.Canvas()))
	t.Cleanup(func() { fynetooltip.DestroyWindowToolTipLayer(parent.Canvas()) })

	d := dialog.NewCustomWithoutButtons("Test", widget.NewLabel("body"), parent)

	pop := customDialogPopUp(d)
	if pop == nil {
		t.Fatal("customDialogPopUp returned nil; cannot attach tooltip layer before Show")
	}
	fynetooltip.AddPopUpToolTipLayer(pop)

	d.Resize(fyne.NewSize(200, 150))
	d.Show()
	d.Hide()
	fynetooltip.DestroyPopUpToolTipLayer(pop)
}

func TestIntValidator(t *testing.T) {
	valid := []string{"0", "1920", "-1080", "+5", "007"}
	for _, s := range valid {
		if err := intValidator(s); err != nil {
			t.Errorf("intValidator(%q) = %v, want nil", s, err)
		}
	}

	invalid := []string{"", " ", "abc", "12.5", "1 920", "1,920", "12px", " 12", "12 ", "٣"}
	for _, s := range invalid {
		if err := intValidator(s); err == nil {
			t.Errorf("intValidator(%q) = nil, want an error", s)
		}
	}
}

func TestEntryTextToInt(t *testing.T) {
	tests := map[string]int32{
		"0":     0,
		"1920":  1920,
		"-1080": -1080,
		"":      0,
		"abc":   0, // validation runs separately; a bad value must not panic
		"12.5":  0,
	}

	for text, want := range tests {
		if got := entryTextToInt(text); got != want {
			t.Errorf("entryTextToInt(%q) = %d, want %d", text, got, want)
		}
	}
}

func TestFirstError(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")

	if got := FirstError(); got != nil {
		t.Errorf("FirstError() = %v, want nil", got)
	}
	if got := FirstError(nil, nil); got != nil {
		t.Errorf("FirstError(nil, nil) = %v, want nil", got)
	}
	if got := FirstError(nil, first, second); got != first {
		t.Errorf("FirstError() = %v, want the first non-nil error", got)
	}
	if got := FirstError(first, nil); got != first {
		t.Errorf("FirstError() = %v, want %v", got, first)
	}
}

// setOnFocusChanged pokes an unexported Fyne field through reflect+unsafe. If a
// Fyne upgrade renames or retypes that field, this is the test that catches it
// instead of the select-on-focus behaviour silently disappearing.
func TestSetOnFocusChangedReachesUnexportedField(t *testing.T) {
	entry := widget.NewEntry()

	var focusEvents []bool
	setOnFocusChanged(entry, func(focused bool) {
		focusEvents = append(focusEvents, focused)
	})

	entry.FocusGained()
	entry.FocusLost()

	if len(focusEvents) != 2 || !focusEvents[0] || focusEvents[1] {
		t.Fatalf("focus events = %v, want [true false]; the reflection hack no longer reaches onFocusChanged", focusEvents)
	}
}

// The entries select their whole contents when focused so a value can be typed
// straight over the old one.
func TestEntryFocusSelectsAllText(t *testing.T) {
	entry := widget.NewEntry()
	entry.SetText("1920")
	setOnFocusChanged(entry, func(focused bool) {
		if focused {
			entry.DoubleTapped(&fyne.PointEvent{})
		}
	})

	entry.FocusGained()

	if entry.SelectedText() != "1920" {
		t.Errorf("SelectedText() = %q, want the whole value selected", entry.SelectedText())
	}
}

func TestSetViaReflectSetsUnexportedField(t *testing.T) {
	entry := widget.NewEntry()

	setViaReflect(entry, "onFocusChanged", reflect.ValueOf(func(bool) {}))

	// Reading it back through the same mechanism proves the write landed.
	field := reflect.ValueOf(entry).Elem().FieldByName("onFocusChanged")
	if !field.IsValid() {
		t.Fatal("onFocusChanged no longer exists on widget.Entry")
	}
	if field.IsNil() {
		t.Error("onFocusChanged is still nil after setViaReflect")
	}
}

// --- new/edit dialog validation -------------------------------------------

// setUpAppSettingForm fills the package-level dialog widgets with a valid
// configuration and restores whatever was there afterwards.
func setUpAppSettingForm(t *testing.T) {
	t.Helper()

	origApp, origDisplay, origMatch := applicationSelect, displaySelect, matchType
	origX, origY, origW, origH := xOffsetText, yOffsetText, widthText, heightText
	origConfirm, origFilter := confirmButton, filterApplications
	t.Cleanup(func() {
		applicationSelect, displaySelect, matchType = origApp, origDisplay, origMatch
		xOffsetText, yOffsetText, widthText, heightText = origX, origY, origW, origH
		confirmButton, filterApplications = origConfirm, origFilter
	})

	useMonitors(t, []Monitor{{number: 1, isPrimary: true, width: 1920, height: 1080}})

	applicationSelect = ui.NewSelect([]Window{{title: "Game", exePath: `C:\game.exe`}}, func(Window) {})
	applicationSelect.SetSelectedIndex(0)

	displaySelect = ui.NewSelect(monitors, func(Monitor) {})
	displaySelect.SetSelectedIndex(0)

	matchType = widget.NewRadioGroup(matchTypes, func(string) {})
	matchType.SetSelected(MatchWindowTitle.String())

	newEntry := func(value string) *widget.Entry {
		e := widget.NewEntry()
		e.Validator = intValidator
		e.SetText(value)
		return e
	}
	xOffsetText, yOffsetText = newEntry("0"), newEntry("0")
	widthText, heightText = newEntry("1920"), newEntry("1080")

	confirmButton = widget.NewButton("Create", func() {})
}

func TestIsValidWithCompleteForm(t *testing.T) {
	setUpAppSettingForm(t)

	if !isValid(true) {
		t.Error("isValid(new) = false for a fully filled in form")
	}
	if !isValid(false) {
		t.Error("isValid(edit) = false for a fully filled in form")
	}
}

// Editing an existing config has no application dropdown, so requiring a
// selection there would leave Save permanently disabled.
func TestIsValidWithoutApplicationSelection(t *testing.T) {
	setUpAppSettingForm(t)
	applicationSelect.ClearSelected()

	if isValid(true) {
		t.Error("isValid(new) = true without an application selected")
	}
	if !isValid(false) {
		t.Error("isValid(edit) = false; editing must not require the application dropdown")
	}
}

func TestIsValidRejectsBadNumbers(t *testing.T) {
	fields := map[string]func() *widget.Entry{
		"x offset": func() *widget.Entry { return xOffsetText },
		"y offset": func() *widget.Entry { return yOffsetText },
		"width":    func() *widget.Entry { return widthText },
		"height":   func() *widget.Entry { return heightText },
	}

	for name, field := range fields {
		for _, bad := range []string{"", "abc", "12.5"} {
			t.Run(name+" = "+strconv.Quote(bad), func(t *testing.T) {
				setUpAppSettingForm(t)
				field().SetText(bad)

				if isValid(true) {
					t.Errorf("isValid(new) = true with %s = %q", name, bad)
				}
				if isValid(false) {
					t.Errorf("isValid(edit) = true with %s = %q", name, bad)
				}
			})
		}
	}
}

func TestIsValidRequiresMatchTypeAndDisplay(t *testing.T) {
	setUpAppSettingForm(t)
	matchType.SetSelected("")
	if isValid(false) {
		t.Error("isValid() = true with no match type selected")
	}

	setUpAppSettingForm(t)
	displaySelect.ClearSelected()
	if isValid(false) {
		t.Error("isValid() = true with no display selected")
	}
}

// The globals are only populated while a dialog is open; isValid must tolerate
// being called before then rather than nil-panic.
func TestIsValidWithUninitialisedWidgets(t *testing.T) {
	setUpAppSettingForm(t)
	matchType = nil
	displaySelect = nil
	xOffsetText = nil

	if isValid(true) || isValid(false) {
		t.Error("isValid() = true with unpopulated widgets")
	}
}

func TestSetConfirmButtonStateFollowsValidity(t *testing.T) {
	setUpAppSettingForm(t)

	setConfirmButtonState(true)
	if confirmButton.Disabled() {
		t.Error("confirm button is disabled for a valid form")
	}

	widthText.SetText("not a number")
	setConfirmButtonState(true)
	if !confirmButton.Disabled() {
		t.Error("confirm button is enabled for an invalid form")
	}

	widthText.SetText("1920")
	setConfirmButtonState(true)
	if confirmButton.Disabled() {
		t.Error("confirm button stayed disabled after the form became valid again")
	}
}

// --- window dropdown ------------------------------------------------------

func TestGetWindowsForSelectSortsCaseInsensitively(t *testing.T) {
	origFilter := filterApplications
	t.Cleanup(func() { filterApplications = origFilter })
	filterApplications = widget.NewCheck("", func(bool) {})
	filterApplications.SetChecked(false)

	got := getWindowsForSelect([]Window{
		{title: "zebra", exePath: `C:\z.exe`},
		{title: "Apple", exePath: `C:\a.exe`},
		{title: "banana", exePath: `C:\b.exe`},
	})

	want := []string{"Apple", "banana", "zebra"}
	if len(got) != len(want) {
		t.Fatalf("got %d windows, want %d", len(got), len(want))
	}
	for i, title := range want {
		if got[i].title != title {
			t.Errorf("position %d = %q, want %q", i, got[i].title, title)
		}
	}
}

func TestGetWindowsForSelectKeepsEveryWindowWhenUnfiltered(t *testing.T) {
	origFilter := filterApplications
	t.Cleanup(func() { filterApplications = origFilter })
	filterApplications = widget.NewCheck("", func(bool) {})
	filterApplications.SetChecked(false)

	input := []Window{{title: "a"}, {title: "b"}, {title: "c"}}
	if got := getWindowsForSelect(input); len(got) != len(input) {
		t.Errorf("got %d windows, want all %d", len(got), len(input))
	}
}

// With the filter on, anything that does not report a normal frame is dropped;
// the zero HWNDs here have no style at all, which is the borderless case.
func TestGetWindowsForSelectDropsBorderlessWindows(t *testing.T) {
	origFilter := filterApplications
	t.Cleanup(func() { filterApplications = origFilter })
	filterApplications = widget.NewCheck("", func(bool) {})
	filterApplications.SetChecked(true)

	got := getWindowsForSelect([]Window{{title: "a"}, {title: "b"}})

	if len(got) != 0 {
		t.Errorf("got %d windows, want 0 once borderless windows are filtered", len(got))
	}
}

func TestGetWindowsForSelectHandlesEmptyInput(t *testing.T) {
	origFilter := filterApplications
	t.Cleanup(func() { filterApplications = origFilter })
	filterApplications = widget.NewCheck("", func(bool) {})

	for _, checked := range []bool{true, false} {
		filterApplications.SetChecked(checked)
		if got := getWindowsForSelect(nil); len(got) != 0 {
			t.Errorf("checked=%v: got %d windows, want 0", checked, len(got))
		}
	}
}

// --- theme ----------------------------------------------------------------

func TestForcedVariantPinsTheVariant(t *testing.T) {
	base := theme.DefaultTheme()
	forcedDark := &forcedVariant{Theme: base, variant: theme.VariantDark}
	forcedLight := &forcedVariant{Theme: base, variant: theme.VariantLight}

	// Whatever variant is asked for, the forced one is what comes back.
	for _, asked := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		if got, want := forcedDark.Color(theme.ColorNameBackground, asked), base.Color(theme.ColorNameBackground, theme.VariantDark); got != want {
			t.Errorf("forced dark asked for variant %d gave %v, want %v", asked, got, want)
		}
		if got, want := forcedLight.Color(theme.ColorNameBackground, asked), base.Color(theme.ColorNameBackground, theme.VariantLight); got != want {
			t.Errorf("forced light asked for variant %d gave %v, want %v", asked, got, want)
		}
	}

	if forcedDark.Color(theme.ColorNameBackground, theme.VariantLight) == forcedLight.Color(theme.ColorNameBackground, theme.VariantLight) {
		t.Error("forced light and dark produced the same background colour")
	}
}

// forcedVariant embeds fyne.Theme, so fonts/icons/sizes must still come from
// the wrapped theme.
func TestForcedVariantDelegatesNonColourLookups(t *testing.T) {
	base := theme.DefaultTheme()
	forced := &forcedVariant{Theme: base, variant: theme.VariantDark}

	if forced.Size(theme.SizeNamePadding) != base.Size(theme.SizeNamePadding) {
		t.Error("Size() was not delegated to the wrapped theme")
	}
	if forced.Icon(theme.IconNameConfirm) != base.Icon(theme.IconNameConfirm) {
		t.Error("Icon() was not delegated to the wrapped theme")
	}
}

func TestSetTheme(t *testing.T) {
	original := fyne.CurrentApp().Settings().Theme()
	t.Cleanup(func() { fyne.CurrentApp().Settings().SetTheme(original) })

	tests := map[string]fyne.ThemeVariant{
		"Light": theme.VariantLight,
		"Dark":  theme.VariantDark,
	}
	for name, wantVariant := range tests {
		setTheme(name)

		forced, ok := fyne.CurrentApp().Settings().Theme().(*forcedVariant)
		if !ok {
			t.Fatalf("setTheme(%q) installed %T, want *forcedVariant", name, fyne.CurrentApp().Settings().Theme())
		}
		if forced.variant != wantVariant {
			t.Errorf("setTheme(%q) pinned variant %d, want %d", name, forced.variant, wantVariant)
		}
	}

	setTheme("System")
	if _, ok := fyne.CurrentApp().Settings().Theme().(*forcedVariant); ok {
		t.Error(`setTheme("System") should hand back the unforced default theme`)
	}

	// Anything unrecognised (an older settings file, say) behaves like System.
	setTheme("Solarized")
	if _, ok := fyne.CurrentApp().Settings().Theme().(*forcedVariant); ok {
		t.Error("an unknown theme name should fall back to the default theme")
	}
}

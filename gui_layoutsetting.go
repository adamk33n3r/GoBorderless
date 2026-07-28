package main

import (
	"slices"
	"strconv"
	"strings"

	"github.com/adamk33n3r/GoBorderless/rx"
	"github.com/adamk33n3r/GoBorderless/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func launchLayoutConfigDialog(parent fyne.Window, isNew bool, settings *Settings, layoutIdx int, draft LayoutConfig, onClose func(saved *LayoutConfig)) {
	d := makeLayoutConfigWindow(settings, draft, isNew, layoutIdx, parent, onClose)
	d.Resize(fyne.NewSize(560, 580))
	d.Show()
}

func makeLayoutConfigWindow(settings *Settings, draft LayoutConfig, isNew bool, layoutIdx int, parent fyne.Window, onClose func(saved *LayoutConfig)) *dialog.CustomDialog {
	var layoutDialog *dialog.CustomDialog
	var windowSub rx.Subscription
	var matcherBox *fyne.Container
	var refreshMatchers func()
	var setConfirmState func()

	if draft.Matchers == nil {
		draft.Matchers = make([]AppMatcher, 0)
	} else {
		draft.Matchers = append([]AppMatcher(nil), draft.Matchers...)
		sortMatchers(draft.Matchers)
	}

	confirmButton := widget.NewButtonWithIcon("Create", theme.ConfirmIcon(), func() {
		windowSub.Unsubscribe()
		draft.Name = strings.TrimSpace(draft.Name)
		sortMatchers(draft.Matchers)
		layoutDialog.Hide()
		onClose(&draft)
	})
	confirmButton.Importance = widget.HighImportance
	confirmButton.Disable()
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		windowSub.Unsubscribe()
		layoutDialog.Hide()
		onClose(nil)
	})
	if !isNew {
		confirmButton.SetText("Save")
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Layout name")
	nameEntry.OnChanged = func(s string) {
		draft.Name = s
		setConfirmState()
	}

	monitorIdx := draft.Monitor - 1
	if monitorIdx < 0 {
		monitorIdx = slices.IndexFunc(monitors, func(m Monitor) bool { return m.isPrimary })
	}
	displaySelect := ui.NewSelect(monitors, func(selected Monitor) {
		draft.Monitor = selected.number
		setConfirmState()
	})
	displaySelect.PlaceHolder = "Select Display"

	xOffsetText := widget.NewEntry()
	xOffsetText.Validator = intValidator
	xOffsetText.OnChanged = func(s string) {
		draft.OffsetX = entryTextToInt(s)
		setConfirmState()
	}
	setOnFocusChanged(xOffsetText, func(focused bool) {
		if focused {
			xOffsetText.DoubleTapped(&fyne.PointEvent{})
		}
	})

	yOffsetText := widget.NewEntry()
	yOffsetText.Validator = intValidator
	yOffsetText.OnChanged = func(s string) {
		draft.OffsetY = entryTextToInt(s)
		setConfirmState()
	}
	setOnFocusChanged(yOffsetText, func(focused bool) {
		if focused {
			yOffsetText.DoubleTapped(&fyne.PointEvent{})
		}
	})

	widthText := widget.NewEntry()
	widthText.Validator = intValidator
	widthText.OnChanged = func(s string) {
		draft.Width = entryTextToInt(s)
		setConfirmState()
	}
	setOnFocusChanged(widthText, func(focused bool) {
		if focused {
			widthText.DoubleTapped(&fyne.PointEvent{})
		}
	})

	heightText := widget.NewEntry()
	heightText.Validator = intValidator
	heightText.OnChanged = func(s string) {
		draft.Height = entryTextToInt(s)
		setConfirmState()
	}
	setOnFocusChanged(heightText, func(focused bool) {
		if focused {
			heightText.DoubleTapped(&fyne.PointEvent{})
		}
	})

	textGrid := container.NewGridWithRows(2,
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("X Offset:"), xOffsetText),
			container.NewVBox(widget.NewLabel("Y Offset:"), yOffsetText),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Width:"), widthText),
			container.NewVBox(widget.NewLabel("Height:"), heightText),
		),
	)

	blackOverlayCheck := widget.NewCheck("Black Overlay", func(checked bool) {
		draft.BlackOverlay = checked
	})

	hideTaskbarCheck := widget.NewCheck("Hide Taskbar when active", func(checked bool) {
		draft.HideTaskbar = checked
	})

	// Assigned before SetText/SetSelectedIndex: those fire OnChanged immediately.
	setConfirmState = func() {
		valid := layoutNameValid(draft.Name) &&
			displaySelect.Selected != nil &&
			xOffsetText.Validate() == nil &&
			yOffsetText.Validate() == nil &&
			widthText.Validate() == nil &&
			heightText.Validate() == nil
		if valid {
			confirmButton.Enable()
		} else {
			confirmButton.Disable()
		}
	}

	nameEntry.SetText(draft.Name)
	displaySelect.SetSelectedIndex(monitorIdx)
	xOffsetText.SetText(strconv.Itoa(int(draft.OffsetX)))
	yOffsetText.SetText(strconv.Itoa(int(draft.OffsetY)))
	widthText.SetText(strconv.Itoa(int(draft.Width)))
	heightText.SetText(strconv.Itoa(int(draft.Height)))
	blackOverlayCheck.SetChecked(draft.BlackOverlay)
	hideTaskbarCheck.SetChecked(draft.HideTaskbar)

	matcherBox = container.NewVBox()
	refreshMatchers = func() {
		sortMatchers(draft.Matchers)
		objects := make([]fyne.CanvasObject, 0, len(draft.Matchers))
		if len(draft.Matchers) == 0 {
			objects = append(objects, widget.NewLabel("No matchers yet — add one from a running window."))
		}
		for i := range draft.Matchers {
			objects = append(objects, buildMatcherEditorRow(settings, &draft, layoutIdx, i, refreshMatchers))
		}
		matcherBox.Objects = objects
		matcherBox.Refresh()
	}

	filterBorderless := widget.NewCheck("Filter out borderless applications", nil)
	filterBorderless.SetChecked(true)

	currentWindowsMutex.Lock()
	windowsOpts := windowsForSelect(currentWindows, filterBorderless.Checked)
	currentWindowsMutex.Unlock()

	var windowSelect *ui.Select[Window]
	windowSelect = ui.NewSelect(windowsOpts, func(selected Window) {
		if slices.Index(windowsOpts, selected) == -1 {
			windowSelect.ClearSelected()
		}
	})
	windowSelect.PlaceHolder = "Select running window…"

	filterBorderless.OnChanged = func(checked bool) {
		currentWindowsMutex.Lock()
		windowsOpts = windowsForSelect(currentWindows, checked)
		currentWindowsMutex.Unlock()
		windowSelect.SetOptions(windowsOpts)
	}

	addMatcherBtn := widget.NewButtonWithIcon("Add Matcher", theme.ContentAddIcon(), func() {
		if windowSelect.Selected == nil {
			return
		}
		sel := *windowSelect.Selected
		draft.Matchers = append(draft.Matchers, AppMatcher{
			WindowName: sel.title,
			ExePath:    sel.exePath,
			MatchType:  settings.Defaults.MatchType,
		})
		sortMatchers(draft.Matchers)
		windowSelect.ClearSelected()
		refreshMatchers()
	})

	windowSub = windowObs.Subscribe(func(windows []Window) {
		if len(windows) == 0 {
			return
		}
		fyne.Do(func() {
			windowsOpts = windowsForSelect(windows, filterBorderless.Checked)
			windowSelect.SetOptions(windowsOpts)
			if windowSelect.Selected != nil && slices.Index(windowsOpts, *windowSelect.Selected) == -1 {
				windowSelect.ClearSelected()
			}
		})
	})

	refreshMatchers()
	setConfirmState()

	matchersSection := container.NewBorder(
		widget.NewLabel("App Matchers"),
		container.NewVBox(
			filterBorderless,
			container.NewBorder(nil, nil, nil, addMatcherBtn, windowSelect),
		),
		nil,
		nil,
		container.NewVScroll(matcherBox),
	)

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Name"),
			nameEntry,
			displaySelect,
			hideTaskbarCheck,
			textGrid,
			blackOverlayCheck,
			widget.NewSeparator(),
		),
		container.NewHBox(cancelButton, layout.NewSpacer(), confirmButton),
		nil,
		nil,
		matchersSection,
	)

	dialogName := "New Layout Config"
	if !isNew {
		dialogName = draft.Name
		if dialogName == "" {
			dialogName = "Edit Layout Config"
		}
	}
	layoutDialog = dialog.NewCustomWithoutButtons(dialogName, content, parent)
	return layoutDialog
}

func buildMatcherEditorRow(settings *Settings, draft *LayoutConfig, layoutIdx, idx int, onRemoved func()) fyne.CanvasObject {
	matcher := draft.Matchers[idx]

	title := widget.NewLabel(matcher.WindowName)
	title.TextStyle.Bold = true
	title.Truncation = fyne.TextTruncateEllipsis
	exe := widget.NewLabel(matcher.ExePath)
	exe.Truncation = fyne.TextTruncateEllipsis

	applyBtn := widget.NewButtonWithIcon("", theme.ViewFullScreenIcon(), nil)
	restoreBtn := widget.NewButtonWithIcon("", theme.ViewRestoreIcon(), nil)

	matchSelect := widget.NewSelect(matchTypes, func(selected string) {
		draft.Matchers[idx].MatchType = GetMatchTypeFromString(selected)
	})
	matchSelect.SetSelected(matcher.MatchType.String())

	autoApply := widget.NewCheck("Auto Apply", func(checked bool) {
		draft.Matchers[idx].AutoApply = checked
		updateMatcherActionButtons(applyBtn, restoreBtn, checked)
	})

	applyBtn.OnTapped = func() {
		m := draft.Matchers[idx]
		if m.AutoApply {
			return
		}
		win := firstInSlice(snapshotCurrentWindows(), func(w Window) bool {
			return matchWindow(w, m)
		})
		if win == nil {
			return
		}
		updated := applyMatcherWithLayout(*draft, m, *win)
		draft.Matchers[idx] = updated
		// Persist Pre* immediately so Restore works even if the dialog is cancelled.
		if layoutIdx >= 0 && layoutIdx < len(settings.Layouts) {
			for j := range settings.Layouts[layoutIdx].Matchers {
				sm := &settings.Layouts[layoutIdx].Matchers[j]
				if sm.WindowName == updated.WindowName && sm.ExePath == updated.ExePath {
					copyMatcherPre(sm, updated)
					settings.Save()
					break
				}
			}
		}
	}
	restoreBtn.OnTapped = func() {
		m := draft.Matchers[idx]
		if m.AutoApply {
			return
		}
		win := firstInSlice(snapshotCurrentWindows(), func(w Window) bool {
			return matchWindow(w, m)
		})
		if win == nil {
			return
		}
		restoreMatcherWithLayout(*draft, m, *win)
	}
	autoApply.SetChecked(matcher.AutoApply)
	updateMatcherActionButtons(applyBtn, restoreBtn, matcher.AutoApply)

	removeBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		draft.Matchers = slices.Delete(draft.Matchers, idx, idx+1)
		onRemoved()
	})

	controls := container.NewHBox(matchSelect, autoApply, applyBtn, restoreBtn, removeBtn)
	return container.NewVBox(
		container.NewBorder(nil, nil, nil, controls, container.NewVBox(title, exe)),
		widget.NewSeparator(),
	)
}

func updateMatcherActionButtons(applyBtn, restoreBtn *widget.Button, autoApply bool) {
	if autoApply {
		applyBtn.Disable()
		restoreBtn.Disable()
	} else {
		applyBtn.Enable()
		restoreBtn.Enable()
	}
}

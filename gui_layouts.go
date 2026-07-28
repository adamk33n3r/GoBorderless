package main

import (
	"fmt"

	"github.com/adamk33n3r/GoBorderless/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var layoutsList *widget.List

func buildLayoutsTab(parent fyne.Window, settings *Settings) fyne.CanvasObject {
	newLayoutBtn := widget.NewButtonWithIcon("Create New Layout Config", theme.ContentAddIcon(), func() {
		draft := newLayoutFromDefaults(settings)
		launchLayoutConfigDialog(parent, true, settings, -1, draft, func(saved *LayoutConfig) {
			if saved != nil {
				settings.AddLayout(*saved)
				settings.Save()
				layoutsList.Refresh()
			}
		})
	})

	layoutsList = widget.NewList(
		func() int { return len(settings.Layouts) },
		func() fyne.CanvasObject { return ui.NewLayoutConfigRow() },
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			updateLayoutListRow(parent, settings, lii, co.(*ui.LayoutConfigRow))
		},
	)
	layoutsList.OnSelected = func(id widget.ListItemID) {
		layoutsList.UnselectAll()
	}

	return container.NewBorder(nil, newLayoutBtn, nil, nil, layoutsList)
}

func newLayoutFromDefaults(settings *Settings) LayoutConfig {
	monitor := settings.Defaults.Monitor
	if monitor < 1 {
		for _, m := range monitors {
			if m.isPrimary {
				monitor = m.number
				break
			}
		}
	}
	return LayoutConfig{
		Monitor:      monitor,
		OffsetX:      settings.Defaults.OffsetX,
		OffsetY:      settings.Defaults.OffsetY,
		Width:        settings.Defaults.Width,
		Height:       settings.Defaults.Height,
		BlackOverlay: settings.Defaults.BlackOverlay,
		HideTaskbar:  settings.Defaults.HideTaskbar,
		Matchers:     make([]AppMatcher, 0),
	}
}

func updateLayoutListRow(parent fyne.Window, settings *Settings, lii widget.ListItemID, row *ui.LayoutConfigRow) {
	layoutCfg := settings.Layouts[lii]
	name := layoutCfg.Name
	if name == "" {
		name = "(unnamed)"
	}
	row.Title.SetText(name)
	row.Title.SetToolTip(name)
	row.Subtitle.SetText(layoutCfg.GeometrySubtitle())
	row.MatcherCount.SetText(layoutCfg.MatcherCountLabel())

	if layoutApplyRestoreEnabled(layoutCfg) {
		row.ApplyBtn.Enable()
		row.RestoreBtn.Enable()
	} else {
		row.ApplyBtn.Disable()
		row.RestoreBtn.Disable()
	}

	if lii == 0 {
		row.UpBtn.Disable()
	} else {
		row.UpBtn.Enable()
	}
	if lii >= len(settings.Layouts)-1 {
		row.DownBtn.Disable()
	} else {
		row.DownBtn.Enable()
	}

	row.UpBtn.OnTapped = func() {
		settings.MoveLayout(lii, -1)
		settings.Save()
		layoutsList.Refresh()
	}
	row.DownBtn.OnTapped = func() {
		settings.MoveLayout(lii, 1)
		settings.Save()
		layoutsList.Refresh()
	}
	row.ApplyBtn.OnTapped = func() {
		applyLayoutManualMatchers(settings, lii, snapshotCurrentWindows())
		layoutsList.Refresh()
	}
	row.RestoreBtn.OnTapped = func() {
		restoreLayoutManualMatchers(settings, lii, snapshotCurrentWindows())
	}
	row.EditBtn.OnTapped = func() {
		layoutCfg := settings.Layouts[lii]
		launchLayoutConfigDialog(parent, false, settings, lii, layoutCfg, func(saved *LayoutConfig) {
			if saved != nil {
				settings.Layouts[lii] = *saved
				settings.Save()
				layoutsList.Refresh()
			}
		})
	}
	row.DeleteBtn.OnTapped = func() {
		layoutCfg := settings.Layouts[lii]
		restoreAllLayoutMatchers(layoutCfg, snapshotCurrentWindows())
		settings.RemoveLayout(lii)
		settings.Save()
		layoutsList.Refresh()
	}
}

func snapshotCurrentWindows() []Window {
	currentWindowsMutex.Lock()
	defer currentWindowsMutex.Unlock()
	out := make([]Window, len(currentWindows))
	copy(out, currentWindows)
	return out
}

// applyLayoutManualMatchers applies geometry to running windows for matchers
// that do not have Auto Apply. Silent partial apply; Pre* captured with the
// !isBorderless guard and persisted.
func applyLayoutManualMatchers(settings *Settings, layoutIdx int, windows []Window) {
	if layoutIdx < 0 || layoutIdx >= len(settings.Layouts) {
		return
	}
	layoutCfg := settings.Layouts[layoutIdx]
	changed := false
	for i := range layoutCfg.Matchers {
		if layoutCfg.Matchers[i].AutoApply {
			continue
		}
		win := firstInSlice(windows, func(w Window) bool {
			return matchWindow(w, layoutCfg.Matchers[i])
		})
		if win == nil {
			continue
		}
		fmt.Println("layout apply for:", layoutCfg.Matchers[i].WindowName)
		before := layoutCfg.Matchers[i]
		layoutCfg.Matchers[i] = applyMatcherWithLayout(layoutCfg, layoutCfg.Matchers[i], *win)
		if layoutCfg.Matchers[i] != before {
			changed = true
		}
	}
	if changed {
		settings.Layouts[layoutIdx] = layoutCfg
		settings.Save()
	}
}

func restoreLayoutManualMatchers(settings *Settings, layoutIdx int, windows []Window) {
	if layoutIdx < 0 || layoutIdx >= len(settings.Layouts) {
		return
	}
	layoutCfg := settings.Layouts[layoutIdx]
	for i := range layoutCfg.Matchers {
		if layoutCfg.Matchers[i].AutoApply {
			continue
		}
		win := firstInSlice(windows, func(w Window) bool {
			return matchWindow(w, layoutCfg.Matchers[i])
		})
		if win == nil {
			continue
		}
		fmt.Println("layout restore for:", layoutCfg.Matchers[i].WindowName)
		restoreWindow(*win, applyPayloadFromLayout(layoutCfg, layoutCfg.Matchers[i]))
	}
}

func restoreAllLayoutMatchers(layoutCfg LayoutConfig, windows []Window) {
	for _, matcher := range layoutCfg.Matchers {
		win := firstInSlice(windows, func(w Window) bool {
			return matchWindow(w, matcher)
		})
		if win == nil {
			continue
		}
		restoreWindow(*win, applyPayloadFromLayout(layoutCfg, matcher))
	}
}

// applyMatcherWithLayout applies one matcher using layout geometry, capturing
// Pre* when needed. Returns the (possibly updated) matcher.
func applyMatcherWithLayout(layoutCfg LayoutConfig, matcher AppMatcher, win Window) AppMatcher {
	if !isBorderless(win) {
		rect := getWindowRect(win.hwnd)
		capturePreGeometry(&matcher, int32(rect.Left), int32(rect.Top), int32(rect.Right), int32(rect.Bottom))
	}
	makeBorderless(win, applyPayloadFromLayout(layoutCfg, matcher))
	return matcher
}

func restoreMatcherWithLayout(layoutCfg LayoutConfig, matcher AppMatcher, win Window) {
	restoreWindow(win, applyPayloadFromLayout(layoutCfg, matcher))
}

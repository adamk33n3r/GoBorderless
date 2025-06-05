package main

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/adamk33n3r/GoBorderless/ui"
)

func buildDefaultsTab(settings *Settings) *fyne.Container {
	defaultXOffsetLabel := widget.NewLabel("X Offset:")
	defaultXOffset := widget.NewEntry()
	defaultXOffset.Validator = intValidator
	defaultXOffset.SetText(strconv.Itoa(int(settings.Defaults.OffsetX)))
	defaultXOffset.OnChanged = func(s string) {
		settings.Defaults.OffsetX = entryTextToInt(s)
		settings.Save()
	}
	setOnFocusChanged(defaultXOffset, func(focused bool) {
		if focused {
			defaultXOffset.DoubleTapped(&fyne.PointEvent{})
		}
	})

	defaultYOffsetLabel := widget.NewLabel("Y Offset:")
	defaultYOffset := widget.NewEntry()
	defaultYOffset.Validator = intValidator
	defaultYOffset.SetText(strconv.Itoa(int(settings.Defaults.OffsetY)))
	defaultYOffset.OnChanged = func(s string) {
		settings.Defaults.OffsetY = entryTextToInt(s)
		settings.Save()
	}
	setOnFocusChanged(defaultYOffset, func(focused bool) {
		if focused {
			defaultYOffset.DoubleTapped(&fyne.PointEvent{})
		}
	})

	defaultWidthLabel := widget.NewLabel("Width:")
	defaultWidth := widget.NewEntry()
	defaultWidth.SetText(strconv.Itoa(int(settings.Defaults.Width)))
	defaultWidth.Validator = intValidator
	defaultWidth.OnChanged = func(s string) {
		settings.Defaults.Width = entryTextToInt(s)
		settings.Save()
	}
	setOnFocusChanged(defaultWidth, func(focused bool) {
		if focused {
			defaultWidth.DoubleTapped(&fyne.PointEvent{})
		}
	})

	defaultHeightLabel := widget.NewLabel("Height:")
	defaultHeight := widget.NewEntry()
	defaultHeight.Validator = intValidator
	defaultHeight.SetText(strconv.Itoa(int(settings.Defaults.Height)))
	defaultHeight.OnChanged = func(s string) {
		settings.Defaults.Height = entryTextToInt(s)
		settings.Save()
	}
	setOnFocusChanged(defaultHeight, func(focused bool) {
		if focused {
			defaultHeight.DoubleTapped(&fyne.PointEvent{})
		}
	})

	defaultDisplay := ui.NewSelect(monitors, func(selected Monitor) {
		settings.Defaults.Monitor = selected.number
		settings.Save()
	})
	defaultDisplay.SetSelectedIndex(settings.Defaults.Monitor - 1)

	defaultMatchTypeLabel := widget.NewLabel("Match Type")
	defaultMatchType := widget.NewRadioGroup(matchTypes, func(selected string) {
		settings.Defaults.MatchType = GetMatchTypeFromString(selected)
		settings.Save()
	})
	defaultMatchType.SetSelected(settings.Defaults.MatchType.String())
	defaultMatchType.Horizontal = true
	defaultMatchType.Required = true

	defaultsGrid := container.NewGridWithRows(2,
		container.NewGridWithColumns(2,
			container.NewVBox(defaultXOffsetLabel, defaultXOffset),
			container.NewVBox(defaultYOffsetLabel, defaultYOffset),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(defaultWidthLabel, defaultWidth),
			container.NewVBox(defaultHeightLabel, defaultHeight),
		),
	)
	return container.NewPadded(container.NewVBox(
		defaultDisplay,
		defaultMatchTypeLabel,
		defaultMatchType,
		defaultsGrid,
	))
}

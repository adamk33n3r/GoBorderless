package main

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	"github.com/adamk33n3r/GoBorderless/rx"
	"github.com/adamk33n3r/GoBorderless/ui"
	"github.com/lxn/win"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	applicationSelect *ui.Select[Window]
	displaySelect     *ui.Select[Monitor]
	matchType         *widget.RadioGroup
	xOffsetText       *widget.Entry
	yOffsetText       *widget.Entry
	widthText         *widget.Entry
	heightText        *widget.Entry
	confirmButton     *widget.Button
)

func isValid(isNew bool) bool {
	return (!isNew || (applicationSelect != nil && applicationSelect.Selected != nil)) &&
		displaySelect != nil && displaySelect.Selected != nil &&
		matchType != nil && matchType.Selected != "" &&
		xOffsetText != nil && xOffsetText.Validate() == nil &&
		yOffsetText != nil && yOffsetText.Validate() == nil &&
		widthText != nil && widthText.Validate() == nil &&
		heightText != nil && heightText.Validate() == nil
}

func setConfirmButtonState(isNew bool) {
	if isValid(isNew) {
		confirmButton.Enable()
	} else {
		confirmButton.Disable()
	}
}

func entryTextToInt(s string) int32 {
	intVal, _ := strconv.Atoi(s)
	return int32(intVal)
}

func setViaReflect(obj any, fieldName string, val reflect.Value) {
	rs := reflect.ValueOf(obj).Elem()
	rf := rs.FieldByName(fieldName)
	rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
	rf.Set(val)
}

func setOnFocusChanged(entry *widget.Entry, onFocusChanged func(focused bool)) {
	setViaReflect(entry, "onFocusChanged", reflect.ValueOf(onFocusChanged))
}

func getWindowsForSelect(allWindows []Window) []Window {
	copyOfWindows := make([]Window, 0, len(allWindows))
	// Filter out windows that don't have normal borders cause they're probably not "real" windows
	// This will also filter out windows that we've already removed borders from
	// Perhaps we should also check the list of existing configs?
	for _, window := range allWindows {
		style := getWindowStyle(window.hwnd)
		if style&win.WS_CAPTION > 0 &&
			((style&win.WS_BORDER) > 0 || (style&win.WS_THICKFRAME) > 0) {
			copyOfWindows = append(copyOfWindows, window)
		}
	}
	slices.SortFunc(copyOfWindows, func(a Window, b Window) int {
		return strings.Compare(strings.ToLower(a.String()), strings.ToLower(b.String()))
	})
	return copyOfWindows
}

func makeAppSettingWindow(settings *Settings, appSetting AppSetting, isNew bool, parent fyne.Window, onClose func(newSetting *AppSetting)) *dialog.CustomDialog {
	currentWindowsMutex.Lock()
	windowsForSelect := getWindowsForSelect(currentWindows)
	currentWindowsMutex.Unlock()

	var appSettingDialog *dialog.CustomDialog
	var windowSub rx.Subscription

	confirmButton = widget.NewButtonWithIcon("Create", theme.ConfirmIcon(), func() {
		if isNew {
			windowSub.Unsubscribe()
		}
		appSettingDialog.Hide()
		onClose(&appSetting)
	})
	confirmButton.Importance = widget.HighImportance
	confirmButton.Disable()
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		if isNew {
			windowSub.Unsubscribe()
		}
		appSettingDialog.Hide()
		onClose(nil)
	})
	if !isNew {
		confirmButton.SetText("Save")
	}

	applicationSelect = ui.NewSelect(windowsForSelect, func(selected Window) {
		if slices.Index(windowsForSelect, selected) == -1 {
			fmt.Println("Selected application no longer exists in the updated window list, resetting selection.")
			applicationSelect.ClearSelected()
			return
		}
		fmt.Println("Selected Application:", selected)
		appSetting.WindowName = selected.title
		appSetting.ExePath = selected.exePath

		setConfirmButtonState(isNew)
	})
	applicationSelect.PlaceHolder = "Select Application"

	monitorIdx := appSetting.Monitor - 1
	if isNew {
		monitorIdx = settings.Defaults.Monitor - 1
		if monitorIdx < 0 {
			monitorIdx = slices.IndexFunc(monitors, func(m Monitor) bool {
				return m.isPrimary
			})
		}
	}
	displaySelect = ui.NewSelect(monitors, func(selected Monitor) {
		appSetting.Monitor = selected.number

		setConfirmButtonState(isNew)
	})
	displaySelect.PlaceHolder = "Select Display"
	displaySelect.SetSelectedIndex(monitorIdx)

	matchType = widget.NewRadioGroup(matchTypes, func(selected string) {
		appSetting.MatchType = GetMatchTypeFromString(selected)

		setConfirmButtonState(isNew)
	})
	if isNew {
		matchType.SetSelected(settings.Defaults.MatchType.String())
	} else {
		matchType.SetSelected(appSetting.MatchType.String())
	}
	matchType.Horizontal = true
	matchType.Required = true

	// Textboxes with labels
	xOffsetLabel := widget.NewLabel("X Offset:")
	xOffsetText = widget.NewEntry()
	xOffsetText.Validator = intValidator
	xOffsetText.OnChanged = func(s string) {
		appSetting.OffsetX = entryTextToInt(s)

		setConfirmButtonState(isNew)
	}
	setOnFocusChanged(xOffsetText, func(focused bool) {
		if focused {
			xOffsetText.DoubleTapped(&fyne.PointEvent{})
		}
	})
	xOffsetText.SetPlaceHolder("0")
	if isNew {
		xOffsetText.SetText(strconv.Itoa(int(settings.Defaults.OffsetX)))
	} else {
		xOffsetText.SetText(strconv.Itoa(int(appSetting.OffsetX)))
	}

	yOffsetLabel := widget.NewLabel("Y Offset:")
	yOffsetText = widget.NewEntry()
	yOffsetText.Validator = intValidator
	yOffsetText.OnChanged = func(s string) {
		appSetting.OffsetY = entryTextToInt(s)

		setConfirmButtonState(isNew)
	}
	setOnFocusChanged(yOffsetText, func(focused bool) {
		if focused {
			yOffsetText.DoubleTapped(&fyne.PointEvent{})
		}
	})
	yOffsetText.SetPlaceHolder("0")
	if isNew {
		yOffsetText.SetText(strconv.Itoa(int(settings.Defaults.OffsetY)))
	} else {
		yOffsetText.SetText(strconv.Itoa(int(appSetting.OffsetY)))
	}

	widthLabel := widget.NewLabel("Width:")
	widthText = widget.NewEntry()
	widthText.Validator = intValidator
	widthText.OnChanged = func(s string) {
		appSetting.Width = entryTextToInt(s)

		setConfirmButtonState(isNew)
	}
	setOnFocusChanged(widthText, func(focused bool) {
		if focused {
			widthText.DoubleTapped(&fyne.PointEvent{})
		}
	})
	widthText.SetPlaceHolder("1920")
	if isNew {
		widthText.SetText(strconv.Itoa(int(settings.Defaults.Width)))
	} else {
		widthText.SetText(strconv.Itoa(int(appSetting.Width)))
	}

	heightLabel := widget.NewLabel("Height:")
	heightText = widget.NewEntry()
	heightText.Validator = intValidator
	heightText.OnChanged = func(s string) {
		appSetting.Height = entryTextToInt(s)

		setConfirmButtonState(isNew)
	}
	setOnFocusChanged(heightText, func(focused bool) {
		if focused {
			heightText.DoubleTapped(&fyne.PointEvent{})
		}
	})
	heightText.SetPlaceHolder("1080")
	if isNew {
		heightText.SetText(strconv.Itoa(int(settings.Defaults.Height)))
	} else {
		heightText.SetText(strconv.Itoa(int(appSetting.Height)))
	}

	// 2x2 grid for labeled textboxes
	textGrid := container.NewGridWithRows(2,
		container.NewGridWithColumns(2,
			container.NewVBox(xOffsetLabel, xOffsetText),
			container.NewVBox(yOffsetLabel, yOffsetText),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(widthLabel, widthText),
			container.NewVBox(heightLabel, heightText),
		),
	)

	if isNew {
		fmt.Println("subscribing to windows observable")
		// TODO: make it work like subject where it outputs last received data on subscription

		windowSub = windowObs.Subscribe(func(windows []Window) {
			if len(windows) == 0 {
				// This is probably a fluke, so let's skip it
				return
			}
			fyne.Do(func() {
				windowsForSelect = getWindowsForSelect(windows)
				applicationSelect.SetOptions(windowsForSelect)

				if applicationSelect.Selected != nil && slices.Index(windowsForSelect, *applicationSelect.Selected) == -1 {
					fmt.Println("Selected application no longer exists in the updated window list, resetting selection.")
					applicationSelect.ClearSelected()
				}
			})
		})
	}

	content := container.NewVBox(
		displaySelect,
		widget.NewLabel("Match Type"),
		matchType,
		textGrid,
		widget.NewLabel(""), // spacer
		container.NewHBox(cancelButton, layout.NewSpacer(), confirmButton),
	)
	if isNew {
		content.Objects = append([]fyne.CanvasObject{applicationSelect}, content.Objects...)
	}

	dialogName := "New App Config"
	if !isNew {
		dialogName = appSetting.Display()
	}
	appSettingDialog = dialog.NewCustomWithoutButtons(dialogName, content, parent)
	return appSettingDialog
}

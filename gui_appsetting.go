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
)

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

func makeAppSettingWindow(appSetting AppSetting, isNew bool, parent fyne.Window, onClose func(newSetting *AppSetting)) *dialog.ConfirmDialog {
	monitorIdx := appSetting.Monitor - 1
	if isNew {
		monitorIdx = slices.IndexFunc(monitors, func(m Monitor) bool {
			return m.isPrimary
		})
	}
	selectedMonitor := monitors[monitorIdx]

	currentWindowsMutex.Lock()
	windowsForSelect := getWindowsForSelect(currentWindows)
	currentWindowsMutex.Unlock()

	applicationSelect = ui.NewSelect(windowsForSelect, func(selected Window) {
		if slices.Index(windowsForSelect, selected) == -1 {
			fmt.Println("Selected application no longer exists in the updated window list, resetting selection.")
			applicationSelect.ClearSelected()
		}
		fmt.Println("Selected Application:", selected)
		appSetting.WindowName = selected.title
		appSetting.ExePath = selected.exePath
	})
	applicationSelect.PlaceHolder = "Select Application"

	displaySelect = ui.NewSelect(monitors, func(selected Monitor) {
		// resLabel.SetText(fmt.Sprintf("Display Resolution is %dx%d", selectedMonitor.width, selectedMonitor.height))
		appSetting.Monitor = selected.number
	})
	displaySelect.PlaceHolder = "Select Display"
	displaySelect.SetSelectedIndex(monitorIdx)

	matchType = widget.NewRadioGroup(matchTypes, func(selected string) {
		appSetting.MatchType = GetMatchTypeFromString(selected)
	})
	if isNew {
		matchType.SetSelected(matchTypes[0])
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
	}
	setOnFocusChanged(xOffsetText, func(focused bool) {
		if focused {
			xOffsetText.DoubleTapped(&fyne.PointEvent{})
		}
	})
	xOffsetText.SetPlaceHolder("0")
	if isNew {
		xOffsetText.SetText("0")
	} else {
		xOffsetText.SetText(strconv.Itoa(int(appSetting.OffsetX)))
	}

	yOffsetLabel := widget.NewLabel("Y Offset:")
	yOffsetText = widget.NewEntry()
	yOffsetText.Validator = intValidator
	yOffsetText.OnChanged = func(s string) {
		appSetting.OffsetY = entryTextToInt(s)
	}
	setOnFocusChanged(yOffsetText, func(focused bool) {
		if focused {
			yOffsetText.DoubleTapped(&fyne.PointEvent{})
		}
	})
	yOffsetText.SetPlaceHolder("0")
	if isNew {
		yOffsetText.SetText("0")
	} else {
		yOffsetText.SetText(strconv.Itoa(int(appSetting.OffsetY)))
	}

	widthLabel := widget.NewLabel("Width:")
	widthText = widget.NewEntry()
	widthText.Validator = intValidator
	widthText.OnChanged = func(s string) {
		appSetting.Width = entryTextToInt(s)
	}
	setOnFocusChanged(widthText, func(focused bool) {
		if focused {
			widthText.DoubleTapped(&fyne.PointEvent{})
		}
	})
	widthText.SetPlaceHolder("1920")
	if isNew {
		widthText.SetText(fmt.Sprintf("%d", selectedMonitor.width))
	} else {
		widthText.SetText(strconv.Itoa(int(appSetting.Width)))
	}

	heightLabel := widget.NewLabel("Height:")
	heightText = widget.NewEntry()
	heightText.Validator = intValidator
	heightText.OnChanged = func(s string) {
		appSetting.Height = entryTextToInt(s)
	}
	setOnFocusChanged(heightText, func(focused bool) {
		if focused {
			heightText.DoubleTapped(&fyne.PointEvent{})
		}
	})
	heightText.SetPlaceHolder("1080")
	if isNew {
		heightText.SetText(fmt.Sprintf("%d", selectedMonitor.height))
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

	// makeBorderlessBtn = widget.NewButton("Apply", func() {
	// 	fmt.Println("Button 1 clicked")
	// 	matchTypeSelected := GetMatchTypeFromString(matchType.Selected)
	// 	x, errX := strconv.Atoi(xOffsetText.Text)
	// 	y, errY := strconv.Atoi(yOffsetText.Text)
	// 	w, errW := strconv.Atoi(widthText.Text)
	// 	h, errH := strconv.Atoi(heightText.Text)
	// 	if errX != nil || errY != nil || errW != nil || errH != nil {
	// 		fmt.Println("Invalid input(s)")
	// 		dialog.NewError(FirstError(errX, errY, errW, errH), window).Show()
	// 		return
	// 	}
	// 	fmt.Println(x, y, w, h)
	// 	updatedWindowsMutex.Lock()
	// 	copyOfList := make([]Window, len(updatedWindows))
	// 	copy(copyOfList, updatedWindows)
	// 	updatedWindowsMutex.Unlock()
	// 	selectedApp := applicationSelect.Selected
	// 	originalRect := getWindowRect(selectedApp.hwnd)
	// 	appSetting := AppSetting{
	// 		WindowName: selectedApp.title,
	// 		ExePath:    selectedApp.exePath,
	// 		PreWidth:   int32(originalRect.Right - originalRect.Left),
	// 		PreHeight:  int32(originalRect.Bottom - originalRect.Top),
	// 		OffsetX:    int32(x),
	// 		OffsetY:    int32(y),
	// 		Width:      int32(w),
	// 		Height:     int32(h),
	// 		Monitor:    int32(selectedMonitor.number),
	// 		MatchType:  matchTypeSelected,
	// 	}
	// 	// idx := slices.IndexFunc(copyOfList, func(win Window) bool {
	// 	// 	return matchWindow(win, appSetting)
	// 	// })
	// 	// if idx == -1 {
	// 	// 	dialog.NewError(fmt.Errorf("No matching window found"), window).Show()
	// 	// 	return
	// 	// }
	// 	// makeBorderless(copyOfList[idx], int32(x), int32(y), int32(w), int32(h))

	// 	settings.AddApp(appSetting)
	// 	settings.Save()
	// })
	// makeBorderlessBtn.Disable()
	// removeBorderlessBtn = widget.NewButton("Remove", func() {
	// 	fmt.Println("Button 2 clicked")
	// 	updatedWindowsMutex.Lock()
	// 	copyOfList := make([]Window, len(updatedWindows))
	// 	copy(copyOfList, updatedWindows)
	// 	updatedWindowsMutex.Unlock()
	// 	appSettingIdx := slices.IndexFunc(settings.Apps, func(appSetting AppSetting) bool {
	// 		return appSetting.WindowName == applicationSelect.Selected.title && appSetting.ExePath == applicationSelect.Selected.exePath
	// 	})
	// 	if appSettingIdx == -1 {
	// 		dialog.NewError(fmt.Errorf("No matching application setting found"), window).Show()
	// 		return
	// 	}
	// 	appSetting := settings.Apps[appSettingIdx]
	// 	idx := slices.IndexFunc(copyOfList, func(win Window) bool { return matchWindow(win, appSetting) })
	// 	if idx == -1 {
	// 		dialog.NewError(fmt.Errorf("No matching window found"), window).Show()
	// 		return
	// 	}
	// 	restoreWindow(copyOfList[idx], appSetting)
	// 	settings.RemoveApp(appSettingIdx)
	// 	settings.Save()
	// })
	// removeBorderlessBtn.Disable()

	// testStep := 0
	var _ = widget.NewButton("TEST", func() {
		// for _, win := range copyOfList {
		// 	if win.title == "Calculator" {
		// 		switch testStep {
		// 		case 0:
		// 			makeBorderless(win, AppSetting{
		// 				Monitor: 1,
		// 				OffsetX: 0,
		// 				OffsetY: 0,
		// 				Width:   400,
		// 				Height:  1200,
		// 			})
		// 			testStep++
		// 		case 1:
		// 			restoreWindow(win, AppSetting{
		// 				PreWidth:  0,
		// 				PreHeight: 0,
		// 			})
		// 			testStep = 0
		// 		}
		// 	}
		// }
	})

	// var application fyne.Widget
	// if isNew {
	// 	application = applicationSelect
	// } else {
	// 	// application = widget.NewLabel(appSetting.Display())
	// }
	// form := widget.NewForm(
	// 	widget.NewFormItem("Display", displaySelect),
	// 	widget.NewFormItem("Match Type", matchType),
	// )
	// if isNew {
	// 	form.SubmitText = "Create"
	// } else {
	// 	form.SubmitText = "Save"
	// }
	// form.OnSubmit = func() {
	// 	fmt.Println("submit")
	// }
	// form.OnCancel = func() {
	// 	fmt.Println("cancel")
	// }

	// Layout
	content := container.NewVBox(
		// centeredLabel,
		// application,
		displaySelect,
		widget.NewLabel("Match Type"),
		matchType,
		textGrid,
		// form,
		// testBtn,
	)
	if isNew {
		content.Objects = append([]fyne.CanvasObject{applicationSelect}, content.Objects...)
	}

	var windowSub rx.Subscription
	if isNew {
		fmt.Println("subscribing to windows observable")
		// TODO: make it work like subject where it outputs last received data on subscription

		windowSub = windowObs.Subscribe(func(windows []Window) {
			// fmt.Println("windows updated")
			if len(windows) == 0 {
				// fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
				// fmt.Println("windows array is empty")
				// This is probably a fluke, so let's skip it
				return
			}
			// for _, win := range windows {
			// 	if win.title == "NoMoreBorderGo" {
			// 		rect := getWindowRect(win.hwnd)
			// 		fmt.Println("NoMoreBorderGo window rect:", rect)
			// 	}
			// }
			fyne.Do(func() {
				windowsForSelect := getWindowsForSelect(windows)
				applicationSelect.SetOptions(windowsForSelect)

				if applicationSelect.Selected != nil && slices.Index(windowsForSelect, *applicationSelect.Selected) == -1 {
					fmt.Println("Selected application no longer exists in the updated window list, resetting selection.")
					applicationSelect.ClearSelected()
				}
			})
		})
	}

	if isNew {
		return dialog.NewCustomConfirm("New App Config", "Create", "Cancel", content, func(saved bool) {
			windowSub.Unsubscribe()
			if saved {
				onClose(&appSetting)
			} else {
				onClose(nil)
			}
		}, parent)
	} else {
		return dialog.NewCustomConfirm(appSetting.Display(), "Save", "Cancel", content, func(saved bool) {
			if saved {
				onClose(&appSetting)
			} else {
				onClose(nil)
			}
		}, parent)
	}
}

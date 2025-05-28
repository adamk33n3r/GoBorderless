package main

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/adamk33n3r/GoBorderless/rx"
	"github.com/adamk33n3r/GoBorderless/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows/registry"
)

var (
	windowCountLabel      *widget.Label
	applicationSelect     *ui.Select[Window]
	displaySelect         *ui.Select[Monitor]
	matchType             *widget.RadioGroup
	xOffsetText           *widget.Entry
	yOffsetText           *widget.Entry
	widthText             *widget.Entry
	heightText            *widget.Entry
	startWithWindowsCheck *widget.Check
	makeBorderlessBtn     *widget.Button
	removeBorderlessBtn   *widget.Button
)

func FirstError(args ...error) error {
	for _, arg := range args {
		if arg != nil {
			return arg
		}
	}
	return nil
}

func intValidator(s string) error {
	if s == "" {
		return nil
	}
	if _, err := strconv.Atoi(s); err != nil {
		return fmt.Errorf("invalid width")
	}
	return nil
}

var updatedWindows = make([]Window, 0) // Temporary list to store window titles
var updatedWindowsMutex sync.Mutex
var windowObs = rx.FromChannel(chWindowList)
var selectedMonitor Monitor

func buildApp(settings *Settings) {
	myApp := app.New()
	myWindow := myApp.NewWindow(APP_NAME)
	fmt.Println(monitors)
	primaryMonitorIdx := slices.IndexFunc(monitors, func(m Monitor) bool {
		return m.isPrimary
	})
	selectedMonitor = monitors[primaryMonitorIdx]

	resLabel := widget.NewLabel(fmt.Sprintf("Display Resolution is %dx%d", selectedMonitor.width, selectedMonitor.height))

	windowCountLabel = widget.NewLabel(fmt.Sprintf("Window Count: %d", 0))

	// Center the label
	centeredLabel := container.New(layout.NewHBoxLayout(),
		layout.NewSpacer(),
		resLabel,
		windowCountLabel,
		layout.NewSpacer(),
	)

	// Dropdowns
	applicationSelect = ui.NewSelect(updatedWindows, func(selected Window) {
		fmt.Println("Selected Application:", selected)
		// if selected != nil {
		makeBorderlessBtn.Enable()
		removeBorderlessBtn.Enable()
		// }
	})
	applicationSelect.PlaceHolder = "Select Application"
	fmt.Println("FIRST VALUE", applicationSelect.Select.Selected)

	displaySelect = ui.NewSelect(monitors, func(selected Monitor) {
		resLabel.SetText(fmt.Sprintf("Display Resolution is %dx%d", selectedMonitor.width, selectedMonitor.height))
		selectedMonitor = selected
	})
	displaySelect.PlaceHolder = "Select Display"
	displaySelect.SetSelectedIndex(primaryMonitorIdx)

	matchType = widget.NewRadioGroup(matchTypes, func(selected string) {
		switch selected {
		case "Window Title":
		case "Exe Path":
		case "Both":
		default:
		}
	})
	matchType.SetSelected(matchTypes[0])
	matchType.Horizontal = true
	matchType.Required = true

	// Textboxes with labels
	xOffsetLabel := widget.NewLabel("X Offset:")
	xOffsetText = widget.NewEntry()
	xOffsetText.Validator = intValidator
	xOffsetText.SetPlaceHolder("0")
	xOffsetText.SetText("0")

	yOffsetLabel := widget.NewLabel("Y Offset:")
	yOffsetText = widget.NewEntry()
	yOffsetText.Validator = intValidator
	yOffsetText.SetPlaceHolder("0")
	yOffsetText.SetText("0")

	widthLabel := widget.NewLabel("Width:")
	widthText = widget.NewEntry()
	widthText.Validator = intValidator
	widthText.SetPlaceHolder("1920")
	widthText.SetText(fmt.Sprintf("%d", selectedMonitor.width))
	widthText.SetText("400")

	heightLabel := widget.NewLabel("Height:")
	heightText = widget.NewEntry()
	heightText.Validator = intValidator
	heightText.SetPlaceHolder("1080")
	heightText.SetText(fmt.Sprintf("%d", selectedMonitor.height))
	heightText.SetText("400")

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

	regName := "NoMoreBorderGo"
	regKey := "Software\\Microsoft\\Windows\\CurrentVersion\\Run"
	appExePath, _ := os.Executable()
	// Checkbox
	startWithWindowsCheck = widget.NewCheck("Start with Windows", func(checked bool) {
		openKey, _ := registry.OpenKey(registry.CURRENT_USER, regKey, registry.ALL_ACCESS)
		defer openKey.Close()
		if checked {
			openKey.SetStringValue(regName, appExePath)
		} else {
			openKey.DeleteValue(regName)
		}
	})
	openKey, _ := registry.OpenKey(registry.CURRENT_USER, regKey, registry.ALL_ACCESS)
	_, _, err := openKey.GetStringValue(regName)
	startWithWindowsCheck.SetChecked(err == nil)
	openKey.Close()

	// Buttons
	makeBorderlessBtn = widget.NewButton("Apply", func() {
		fmt.Println("Button 1 clicked")
		matchTypeSelected := GetMatchTypeFromString(matchType.Selected)
		x, errX := strconv.Atoi(xOffsetText.Text)
		y, errY := strconv.Atoi(yOffsetText.Text)
		w, errW := strconv.Atoi(widthText.Text)
		h, errH := strconv.Atoi(heightText.Text)
		if errX != nil || errY != nil || errW != nil || errH != nil {
			fmt.Println("Invalid input(s)")
			dialog.NewError(FirstError(errX, errY, errW, errH), myWindow).Show()
			return
		}
		fmt.Println(x, y, w, h)
		updatedWindowsMutex.Lock()
		copyOfList := make([]Window, len(updatedWindows))
		copy(copyOfList, updatedWindows)
		updatedWindowsMutex.Unlock()
		selectedApp := applicationSelect.Selected
		originalRect := getWindowRect(selectedApp.hwnd)
		appSetting := AppSetting{
			WindowName: selectedApp.title,
			ExePath:    selectedApp.exePath,
			PreWidth:   int32(originalRect.Right - originalRect.Left),
			PreHeight:  int32(originalRect.Bottom - originalRect.Top),
			OffsetX:    int32(x),
			OffsetY:    int32(y),
			Width:      int32(w),
			Height:     int32(h),
			Monitor:    int32(selectedMonitor.number),
			MatchType:  matchTypeSelected,
		}
		// idx := slices.IndexFunc(copyOfList, func(win Window) bool {
		// 	return matchWindow(win, appSetting)
		// })
		// if idx == -1 {
		// 	dialog.NewError(fmt.Errorf("No matching window found"), myWindow).Show()
		// 	return
		// }
		// makeBorderless(copyOfList[idx], int32(x), int32(y), int32(w), int32(h))
		settings.AddApp(appSetting)
		settings.Save()
	})
	makeBorderlessBtn.Disable()
	removeBorderlessBtn = widget.NewButton("Remove", func() {
		fmt.Println("Button 2 clicked")
		updatedWindowsMutex.Lock()
		copyOfList := make([]Window, len(updatedWindows))
		copy(copyOfList, updatedWindows)
		updatedWindowsMutex.Unlock()
		appSettingIdx := slices.IndexFunc(settings.Apps, func(appSetting AppSetting) bool {
			return appSetting.WindowName == applicationSelect.Selected.title && appSetting.ExePath == applicationSelect.Selected.exePath
		})
		if appSettingIdx == -1 {
			dialog.NewError(fmt.Errorf("No matching application setting found"), myWindow).Show()
			return
		}
		appSetting := settings.Apps[appSettingIdx]
		idx := slices.IndexFunc(copyOfList, func(win Window) bool { return matchWindow(win, appSetting) })
		if idx == -1 {
			dialog.NewError(fmt.Errorf("No matching window found"), myWindow).Show()
			return
		}
		restoreWindow(copyOfList[idx], appSetting)
		settings.RemoveApp(appSettingIdx)
		settings.Save()
	})
	removeBorderlessBtn.Disable()

	// testStep := 0
	// testBtn := widget.NewButton("TEST", func() {
	// 	updatedWindowsMutex.Lock()
	// 	copyOfList := make([]Window, len(updatedWindows))
	// 	copy(copyOfList, updatedWindows)
	// 	updatedWindowsMutex.Unlock()
	// 	for _, win := range copyOfList {
	// 		if win.title == "Calculator" {
	// 			switch testStep {
	// 			case 0:
	// 				makeBorderless(win, AppSetting{
	// 					Monitor: 1,
	// 					OffsetX: 0,
	// 					OffsetY: 0,
	// 					Width:   400,
	// 					Height:  1200,
	// 				})
	// 				testStep++
	// 			case 1:
	// 				restoreWindow(win, AppSetting{
	// 					PreWidth:  0,
	// 					PreHeight: 0,
	// 				})
	// 				testStep = 0
	// 			}
	// 		}
	// 	}
	// })

	// Layout
	content := container.NewVBox(
		centeredLabel,
		applicationSelect,
		displaySelect,
		matchType,
		textGrid,
		startWithWindowsCheck,
		container.NewHBox(makeBorderlessBtn, removeBorderlessBtn),
	)

	windowObs.Subscribe(func(windows []Window) {
		// for _, win := range windows {
		// 	if win.title == "NoMoreBorderGo" {
		// 		rect := getWindowRect(win.hwnd)
		// 		fmt.Println("NoMoreBorderGo window rect:", rect)
		// 	}
		// }
		fyne.Do(func() {
			slices.SortFunc(windows, func(a Window, b Window) int {
				return strings.Compare(strings.ToLower(a.String()), strings.ToLower(b.String()))
			})
			windowCountLabel.SetText(fmt.Sprintf("Window Count: %d", len(windows)))
			applicationSelect.SetOptions(windows)

			if applicationSelect.Selected != nil && slices.Index(windows, *applicationSelect.Selected) == -1 {
				fmt.Println("Selected application no longer exists in the updated window list, resetting selection.")
				applicationSelect.ClearSelected()
				makeBorderlessBtn.Disable()
				removeBorderlessBtn.Disable()
			}
		})
	})

	windowObs.Subscribe(func(windows []Window) {
		updatedWindowsMutex.Lock()
		updatedWindows = windows
		updatedWindowsMutex.Unlock()
	})

	myWindow.SetContent(content)
	myWindow.CenterOnScreen()
	myWindow.Resize(fyne.NewSize(400, 400))
	myWindow.ShowAndRun()
}

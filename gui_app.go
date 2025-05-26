package main

import (
	"NoMoreBorderGo/rx"
	"NoMoreBorderGo/ui"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

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
var monitors []Monitor

func buildApp(settings *Settings) {
	myApp := app.New()
	myWindow := myApp.NewWindow("NoMoreBorderGo")
	monitors = getMonitors()
	fmt.Println(monitors)
	selectedMonitor = monitors[0]

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
	})
	applicationSelect.PlaceHolder = "Select Application"

	displaySelect = ui.NewSelect(monitors, func(selected Monitor) {
		resLabel.SetText(fmt.Sprintf("Display Resolution is %dx%d", selectedMonitor.width, selectedMonitor.height))
	})
	displaySelect.PlaceHolder = "Select Display"
	displaySelect.SetSelectedIndex(0)

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
	exePath, _ := os.Executable()
	// Checkbox
	startWithWindowsCheck = widget.NewCheck("Start with Windows", func(checked bool) {
		openKey, _ := registry.OpenKey(registry.CURRENT_USER, regKey, registry.ALL_ACCESS)
		defer openKey.Close()
		if checked {
			openKey.SetStringValue(regName, exePath)
		} else {
			openKey.DeleteValue(regName)
		}
	})
	openKey, _ := registry.OpenKey(registry.CURRENT_USER, regKey, registry.ALL_ACCESS)
	_, _, err := openKey.GetStringValue(regName)
	startWithWindowsCheck.SetChecked(err == nil)
	openKey.Close()

	// Buttons
	makeBorderless := widget.NewButton("Apply", func() {
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
		exeSplitIdx := strings.LastIndex(applicationSelect.Selected, " - ")
		title := applicationSelect.Selected[:exeSplitIdx]
		exePath := applicationSelect.Selected[exeSplitIdx+3:] // +3 to skip " - "
		var filterFunc func(Window) bool
		switch matchTypeSelected {
		case MatchWindowTitle:
			fmt.Println("Match by Window Title:", title)
			filterFunc = func(w Window) bool { return w.title == title }
		case MatchExePath:
			fmt.Println("Match by Exe Path:", exePath)
			filterFunc = func(w Window) bool { return w.exePath == exePath }
		case MatchBoth:
			fmt.Println("Match by Both:(", title, exePath, ")")
			filterFunc = func(w Window) bool { return w.title == title && w.exePath == exePath }
		case MatchEither:
			fmt.Println("Match by Either:(", title, exePath, ")")
			filterFunc = func(w Window) bool { return w.title == title || w.exePath == exePath }
		default:
			fmt.Println("Unknown match type")
		}
		idx := slices.IndexFunc(copyOfList, filterFunc)
		if idx == -1 {
			dialog.NewError(fmt.Errorf("No matching window found"), myWindow).Show()
			return
		}
		makeBorderless(copyOfList[idx], int32(x), int32(y), int32(w), int32(h))
		addToSettings(settings, AppSetting{
			WindowName: title,
			ExePath:    exePath,
			PreHeight:  int32(h),
			PreWidth:   int32(w),
			OffsetX:    int32(x),
			OffsetY:    int32(y),
			Monitor:    int32(selectedMonitor.number),
		})
	})
	removeBorderless := widget.NewButton("Remove", func() {
		fmt.Println("Button 2 clicked")
		updatedWindowsMutex.Lock()
		copyOfList := make([]Window, len(updatedWindows))
		copy(copyOfList, updatedWindows)
		updatedWindowsMutex.Unlock()
		idx := slices.IndexFunc(copyOfList, func(w Window) bool { return w.title == applicationSelect.Selected })
		restoreWindow(copyOfList[idx])
	})

	// Layout
	content := container.NewVBox(
		centeredLabel,
		applicationSelect,
		displaySelect,
		matchType,
		textGrid,
		startWithWindowsCheck,
		container.NewHBox(makeBorderless, removeBorderless),
	)

	windowObs.Subscribe(func(windows []Window) {
		fyne.Do(func() {
			slices.SortFunc(windows, func(a Window, b Window) int {
				return strings.Compare(strings.ToLower(a.String()), strings.ToLower(b.String()))
			})
			windowCountLabel.SetText(fmt.Sprintf("Window Count: %d", len(windows)))
			applicationSelect.SetOptions(windows)
		})
	})

	windowObs.Subscribe(func(windows []Window) {
		// fmt.Println("Window List Updated:", len(windows))
		updatedWindowsMutex.Lock()
		updatedWindows = windows
		updatedWindowsMutex.Unlock()
	})

	myWindow.SetContent(content)
	myWindow.CenterOnScreen()
	myWindow.Resize(fyne.NewSize(400, 400))
	myWindow.ShowAndRun()
}

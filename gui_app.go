package main

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/adamk33n3r/GoBorderless/rx"
	"github.com/adamk33n3r/GoBorderless/ui"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	"golang.org/x/sys/windows/registry"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var windowObs = rx.FromChannel(chWindowList)

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

var settingsList *widget.List

var currentWindows = make([]Window, 0) // Temporary list to store window titles
var currentWindowsMutex sync.Mutex

func launchAppSettingDialog(parent fyne.Window, new bool, appSetting AppSetting, onClose func(newSetting *AppSetting)) {
	dialog := makeAppSettingWindow(appSetting, new, parent, onClose)
	dialog.Show()
}

func buildApp(settings *Settings) {
	goBorderlessApp := app.New()
	before := time.Now()
	mainWindow := goBorderlessApp.NewWindow(APP_NAME)
	fmt.Println("NewWindow took:", time.Since(before))
	fmt.Println(monitors)

	windowObs.Subscribe(func(windows []Window) {
		// fmt.Println("MainApp: windows updated")
		currentWindowsMutex.Lock()
		currentWindows = windows
		currentWindowsMutex.Unlock()
	})

	newAppConfig := widget.NewButtonWithIcon("Create New App Config", theme.ContentAddIcon(), func() {
		newAppSetting := AppSetting{}
		launchAppSettingDialog(mainWindow, true, newAppSetting, func(newSetting *AppSetting) {
			if newSetting != nil {
				fmt.Println(newSetting)
				settings.AddApp(*newSetting)
				settings.Save()
				settingsList.Refresh()
			}
		})
	})

	appName := widget.NewLabel("Go Borderless!")
	appName.TextStyle.Bold = true
	appName.SizeName = "headingText"

	settingsList = widget.NewList(func() int {
		return len(settings.Apps)
	}, func() fyne.CanvasObject {
		return ui.NewAppSettingRow()
	}, func(lii widget.ListItemID, co fyne.CanvasObject) {
		appSetting := settings.Apps[lii]
		row := co.(*ui.AppSettingRow)
		row.Title.SetText(appSetting.Display())
		row.Title.SetToolTip(appSetting.Display())
		row.AutoApply.SetChecked(appSetting.AutoApply)
		row.AutoApply.OnChanged = func(checked bool) {
			appSetting := settings.Apps[lii]
			if checked {
				row.ApplyBtn.Disable()
				row.RestoreBtn.Disable()
			} else {
				row.ApplyBtn.Enable()
				row.RestoreBtn.Enable()
			}
			appSetting.AutoApply = checked
			settings.Apps[lii] = appSetting
			settings.Save()
		}
		row.ApplyBtn.OnTapped = func() {
			appSetting := settings.Apps[lii]
			fmt.Println("clicked apply for:", appSetting.Display())
			win := firstInSlice(currentWindows, func(win Window) bool { return matchWindow(win, appSetting) })
			if !isBorderless(*win) {
				originalRect := getWindowRect(win.hwnd)
				appSetting.PreWidth = int32(originalRect.Right - originalRect.Left)
				appSetting.PreHeight = int32(originalRect.Bottom - originalRect.Top)
				appSetting.PreOffsetX = int32(originalRect.Left)
				appSetting.PreOffsetY = int32(originalRect.Top)
				settings.Apps[lii] = appSetting
				settings.Save()
			}
			makeBorderless(*win, appSetting)
		}
		row.RestoreBtn.OnTapped = func() {
			appSetting := settings.Apps[lii]
			fmt.Println("clicked undo for:", appSetting.Display())
			win := firstInSlice(currentWindows, func(win Window) bool { return matchWindow(win, appSetting) })
			restoreWindow(*win, appSetting)
		}
		if appSetting.AutoApply {
			row.ApplyBtn.Disable()
			row.RestoreBtn.Disable()
		}
		row.EditBtn.OnTapped = func() {
			// Need to fetch again from array to "reset" the values since this update func is only called on occasion
			appSetting := settings.Apps[lii]
			launchAppSettingDialog(mainWindow, false, appSetting, func(newSetting *AppSetting) {
				if newSetting != nil {
					fmt.Println(newSetting)
					settings.Apps[lii] = *newSetting
					settings.Save()
				}
			})
		}
		row.DeleteBtn.OnTapped = func() {
			appSetting := settings.Apps[lii]
			idx := slices.IndexFunc(currentWindows, func(win Window) bool { return matchWindow(win, appSetting) })
			if idx == -1 {
				fmt.Println("No matching window found")
				return
			}
			restoreWindow(currentWindows[idx], appSetting)
			settings.RemoveApp(lii)
			settings.Save()
			settingsList.Refresh()
		}
		// co.(*fyne.Container).Objects[0].(*widget.Label).SetText(settings.Apps[lii].WindowName)
		// co.(*fyne.Container).Objects[2].(*widget.Button).OnTapped = func() {
		// 	appSetting := settings.Apps[lii]
		// 	launchAppSettingDialog(mainWindow, false, &appSetting, func(saved bool) {
		// 		if saved {
		// 			fmt.Println("was saved?")
		// 			fmt.Println(appSetting)
		// 			settings.Apps[lii] = appSetting
		// 			settings.Save()
		// 		}
		// 	})
		// }
	})
	// appTiles := make([]fyne.CanvasObject, 0)
	// for _, appSetting := range settings.Apps {
	// 	appTiles = append(appTiles, widget.NewButton(appSetting.WindowName, func() {}))
	// }
	// appGrid := container.NewGridWrap(fyne.NewSquareSize(100), appTiles...)
	settingsList.OnSelected = func(id widget.ListItemID) {
		fmt.Println("selected:", id)
		settingsList.UnselectAll()
		// appSetting := settings.Apps[id]
		// launchAppSettingDialog(mainWindow, false, &appSetting, func(saved bool) {
		// 	if saved {
		// 		fmt.Println("was saved?")
		// 	}
		// })
	}
	// settingsList.Resize(fyne.NewSize())

	centeredAppLabel := container.NewHBox(
		layout.NewSpacer(),
		appName,
		layout.NewSpacer(),
	)

	content := container.NewBorder(
		nil,
		newAppConfig,
		nil,
		nil,
		settingsList,
		// appGrid,
	)

	regName := "NoMoreBorderGo"
	regKey := "Software\\Microsoft\\Windows\\CurrentVersion\\Run"
	appExePath, _ := os.Executable()
	// Checkbox
	startWithWindowsCheck := widget.NewCheck("Start with Windows", func(checked bool) {
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

	settingsContent := container.NewVBox(
		startWithWindowsCheck,
	)

	appTabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Apps", theme.ListIcon(), content),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), settingsContent),
	)

	mainWindow.SetContent(fynetooltip.AddWindowToolTipLayer(container.NewBorder(
		centeredAppLabel,
		nil,
		nil,
		nil,
		appTabs,
	), mainWindow.Canvas()))
	mainWindow.CenterOnScreen()
	mainWindow.Resize(fyne.NewSquareSize(620))
	// myWindow.Resize(fyne.NewSize(400, 400))
	fmt.Println("running app...")
	mainWindow.ShowAndRun()
}

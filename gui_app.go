package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/adamk33n3r/GoBorderless/res"
	"github.com/adamk33n3r/GoBorderless/rx"
	"github.com/adamk33n3r/GoBorderless/ui"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	"github.com/lxn/win"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/driver/desktop"
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
		return fmt.Errorf("invalid number")
	}
	if _, err := strconv.Atoi(s); err != nil {
		return fmt.Errorf("invalid number")
	}
	return nil
}

var settingsList *widget.List

var currentWindows = make([]Window, 0) // Temporary list to store window titles
var currentWindowsMutex sync.Mutex

func launchAppSettingDialog(parent fyne.Window, new bool, settings *Settings, appSetting AppSetting, onClose func(newSetting *AppSetting)) {
	dialog := makeAppSettingWindow(settings, appSetting, new, parent, onClose)
	dialog.Show()
}

func buildApp(settings *Settings) fyne.App {
	fyneApp := app.New()
	fyneApp.SetIcon(res.ResIconPng)
	before := time.Now()
	mainWindow := fyneApp.NewWindow(APP_NAME)
	fmt.Println("NewWindow took:", time.Since(before))

	windowObs.Subscribe(func(windows []Window) {
		// fmt.Println("MainApp: windows updated")
		currentWindowsMutex.Lock()
		currentWindows = windows
		currentWindowsMutex.Unlock()
	})

	newAppConfig := widget.NewButtonWithIcon("Create New App Config", theme.ContentAddIcon(), func() {
		newAppSetting := AppSetting{}
		launchAppSettingDialog(mainWindow, true, settings, newAppSetting, func(newSetting *AppSetting) {
			if newSetting != nil {
				fmt.Println(newSetting)
				settings.AddApp(*newSetting)
				settings.Save()
				settingsList.Refresh()
			}
		})
	})

	appName := widget.NewLabel("Go Borderless! v" + fyneApp.Metadata().Version)
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
			if win == nil {
				return
			}
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
			if win == nil {
				return
			}
			restoreWindow(*win, appSetting)
		}
		if appSetting.AutoApply {
			row.ApplyBtn.Disable()
			row.RestoreBtn.Disable()
		}
		row.EditBtn.OnTapped = func() {
			// Need to fetch again from array to "reset" the values since this update func is only called on occasion
			appSetting := settings.Apps[lii]
			launchAppSettingDialog(mainWindow, false, settings, appSetting, func(newSetting *AppSetting) {
				if newSetting != nil {
					fmt.Println(newSetting)
					settings.Apps[lii] = *newSetting
					settings.Save()
				}
			})
		}
		row.DeleteBtn.OnTapped = func() {
			appSetting := settings.Apps[lii]
			win := firstInSlice(currentWindows, func(win Window) bool { return matchWindow(win, appSetting) })
			if win != nil {
				restoreWindow(*win, appSetting)
			}
			settings.RemoveApp(lii)
			settings.Save()
			settingsList.Refresh()
		}
	})
	settingsList.OnSelected = func(id widget.ListItemID) {
		fmt.Println("selected:", id)
		settingsList.UnselectAll()
	}

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
	)

	appTabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Apps", theme.ListIcon(), content),
		container.NewTabItemWithIcon("Defaults", theme.ViewRestoreIcon(), buildDefaultsTab(settings)),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), buildSettingsTab(settings)),
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
	mainWindow.SetFixedSize(true)

	fmt.Println("Running app...")
	mainWindow.Show()
	handleWindowsInit(fyneApp, mainWindow, settings)

	return fyneApp
}

func handleWindowsInit(fyneApp fyne.App, window fyne.Window, settings *Settings) {
	window.(driver.NativeWindow).RunNative(func(context any) {
		switch ctx := context.(type) {
		case driver.WindowsWindowContext:
			windowHWND := win.HWND(ctx.HWND)
			if desk, ok := fyneApp.(desktop.App); ok {
				m := fyne.NewMenu(APP_NAME,
					fyne.NewMenuItem("Show", func() {
						window.Show()
						win.ShowWindow(windowHWND, win.SW_RESTORE)
					}))
				desk.SetSystemTrayMenu(m)
			}

			window.SetCloseIntercept(func() {
				if !settings.CloseToTray {
					fyneApp.Quit()
					return
				}
				window.Hide()
			})

			// Intercept minimize
			go func() {
				for {
					time.Sleep(250 * time.Millisecond)
					if !settings.MinimizeToTray {
						continue
					}
					if win.IsIconic(windowHWND) {
						fyne.DoAndWait(func() {
							window.Hide()
						})
					}
				}
			}()

			if settings.StartMinimized {
				win.ShowWindow(windowHWND, win.SW_MINIMIZE)
			}
		default:
			fmt.Println("not running in windows....don't do that")
			fyneApp.Quit()
		}
	})
}

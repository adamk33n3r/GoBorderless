package main

import (
	"fmt"
	"math/rand"
	"slices"
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

var hwnds = make(map[fyne.Window]win.HWND)

/**
 * Get's the HWND for a fyne window by temporarily setting it's title to a random string in order to find it.
 */
func getFyneHWND(w fyne.Window) win.HWND {
	if h, ok := hwnds[w]; ok {
		return h
	}

	randomTitle := make([]byte, 128)
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := range randomTitle {
		randomTitle[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	originalTitle := w.Title()
	defer w.SetTitle(originalTitle)

	w.SetTitle(string(randomTitle))

	// Try by foreground
	if h := win.GetForegroundWindow(); h != 0 {
		t := getWindowTitle(uintptr(h))
		if string(randomTitle) == t {
			hwnds[w] = h
			return h
		}
	}

	enumWindows(func(hwnd, lparam uintptr) uintptr {
		title := getWindowTitle(hwnd)
		if string(randomTitle) == title {
			hwnds[w] = win.HWND(hwnd)
			return 0
		}
		return 1
	}, nil)

	if hwnds[w] != 0 {
		return hwnds[w]
	}

	return win.HWND(0)
}

func buildApp(settings *Settings) fyne.App {
	fyneApp := app.New()
	fyneApp.SetIcon(res.ResIconPng)
	before := time.Now()
	mainWindow := fyneApp.NewWindow(APP_NAME)
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

	fmt.Println("running app...")
	mainWindow.Show()
	mainWindowHWND := getFyneHWND(mainWindow)
	if mainWindowHWND != 0 {
		if desk, ok := fyneApp.(desktop.App); ok {
			m := fyne.NewMenu(APP_NAME,
				fyne.NewMenuItem("Show", func() {
					fmt.Println("Show")
					mainWindow.Show()
					win.ShowWindow(mainWindowHWND, win.SW_RESTORE)
				}))
			desk.SetSystemTrayMenu(m)
		}

		mainWindow.SetCloseIntercept(func() {
			if !settings.CloseToTray {
				fyneApp.Quit()
				return
			}
			fmt.Println("intercept close, hiding window")
			mainWindow.Hide()
		})

		// Intercept minimize
		go func() {
			for {
				time.Sleep(250 * time.Millisecond)
				if !settings.MinimizeToTray {
					continue
				}
				if win.IsIconic(mainWindowHWND) {
					fyne.DoAndWait(func() {
						mainWindow.Hide()
					})
				}
			}
		}()

		if settings.StartMinimized {
			win.ShowWindow(mainWindowHWND, win.SW_MINIMIZE)
		}
	} else {
		fmt.Println("couldn't find window handle")
	}

	return fyneApp
}

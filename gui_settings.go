package main

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows/registry"
)

func setStartupReg(enabled bool) {
	regKey := "Software\\Microsoft\\Windows\\CurrentVersion\\Run"
	appExePath, _ := os.Executable()
	openKey, _ := registry.OpenKey(registry.CURRENT_USER, regKey, registry.ALL_ACCESS)
	defer openKey.Close()

	if enabled {
		openKey.SetStringValue(APP_NAME, appExePath)
	} else {
		openKey.DeleteValue(APP_NAME)
	}
}

func buildSettingsTab(settings *Settings) *fyne.Container {
	startWithWindowsCheck := widget.NewCheck("Start with Windows", func(checked bool) {
		settings.StartWithWindows = checked
		settings.Save()
		setStartupReg(checked)
	})
	startWithWindowsCheck.SetChecked(settings.StartWithWindows)
	setStartupReg(settings.StartWithWindows)

	closeToTray := widget.NewCheck("Close to Tray", func(checked bool) {
		settings.CloseToTray = checked
		settings.Save()
	})
	closeToTray.SetChecked(settings.CloseToTray)

	minimizeToTray := widget.NewCheck("Minimize to Tray", func(checked bool) {
		settings.MinimizeToTray = checked
		settings.Save()
	})
	minimizeToTray.SetChecked(settings.MinimizeToTray)

	startMinimized := widget.NewCheck("Start Minimized", func(checked bool) {
		settings.StartMinimized = checked
		settings.Save()
	})
	startMinimized.SetChecked(settings.StartMinimized)

	theme := widget.NewSelect([]string{"System", "Light", "Dark"}, func(selected string) {
		settings.Theme = selected
		settings.Save()
		setTheme(selected)
	})
	theme.SetSelected(settings.Theme)

	checks := container.NewGridWithColumns(2,
		container.NewVBox(startWithWindowsCheck, minimizeToTray),
		container.NewVBox(startMinimized, closeToTray),
	)
	form := widget.NewForm(
		widget.NewFormItem("Theme:", theme),
		widget.NewFormItem("", checks),
	)

	return container.NewVBox(
		form,
	)
}

func setTheme(themeName string) {
	switch themeName {
	case "Light":
		fyne.CurrentApp().Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
	case "Dark":
		fyne.CurrentApp().Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})
	default:
		fyne.CurrentApp().Settings().SetTheme(theme.DefaultTheme())
	}
}

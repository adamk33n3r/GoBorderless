package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var (
	appFolder    = filepath.Join(getDocumentsFolder(), APP_NAME)
	settingsPath = filepath.Join(appFolder, "settings.json")
)

type MatchType int

const (
	MatchWindowTitle MatchType = iota
	MatchExePath
	MatchBoth
	MatchEither
)

var matchTypes = []string{"Window Title", "Exe Path", "Both", "Either"}

func (m MatchType) String() string {
	switch m {
	case MatchWindowTitle:
		return matchTypes[0]
	case MatchExePath:
		return matchTypes[1]
	case MatchBoth:
		return matchTypes[2]
	case MatchEither:
		return matchTypes[3]
	default:
		return "Unknown"
	}
}
func GetMatchTypeFromString(s string) MatchType {
	switch s {
	case matchTypes[0]:
		return MatchWindowTitle
	case matchTypes[1]:
		return MatchExePath
	case matchTypes[2]:
		return MatchBoth
	case matchTypes[3]:
		return MatchEither
	default:
		return MatchWindowTitle // Default to MatchWindowTitle if unknown
	}
}

type AppSetting struct {
	WindowName string    `json:"windowName"`
	ExePath    string    `json:"exePath"`
	MatchType  MatchType `json:"matchType"`
	AutoApply  bool      `json:"autoApply"`
	Monitor    int       `json:"monitor"`
	PreWidth   int32     `json:"preWidth"`
	PreHeight  int32     `json:"preHeight"`
	PreOffsetX int32     `json:"preOffsetX"`
	PreOffsetY int32     `json:"preOffsetY"`
	Width      int32     `json:"width"`
	Height     int32     `json:"height"`
	OffsetX    int32     `json:"offsetX"`
	OffsetY    int32     `json:"offsetY"`
}

func (as AppSetting) Display() string {
	return fmt.Sprintf("%s | %s", as.WindowName, as.ExePath)
}

type Settings struct {
	Apps             []AppSetting `json:"apps"`
	Theme            string       `json:"theme"`
	StartWithWindows bool         `json:"startWithWindows"`
}

func newSettings() *Settings {
	return &Settings{
		Apps:             make([]AppSetting, 0),
		Theme:            "System",
		StartWithWindows: false,
	}
}

func loadSettings() (*Settings, error) {
	os.MkdirAll(appFolder, os.ModeDir)
	bytes, err := os.ReadFile(settingsPath)
	// No settings file found, create default settings
	if err != nil {
		return newSettings(), nil
	}

	var settings *Settings
	if err := json.Unmarshal(bytes, &settings); err != nil {
		return newSettings(), err
	}
	settings.sortApps()
	return settings, nil
}

func (settings *Settings) sortApps() {
	slices.SortFunc(settings.Apps, func(a AppSetting, b AppSetting) int {
		return strings.Compare(a.WindowName, b.WindowName)
	})
}

func (settings *Settings) Save() error {
	bytes, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, bytes, 0666)
}

func backUpSettingsFile() error {
	baseBackupPath := settingsPath + ".bak"
	backupPath := baseBackupPath
	i := 1
	for {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		backupPath = fmt.Sprintf("%s.%d", baseBackupPath, i)
		i++
	}
	if _, err := os.Stat(settingsPath); err == nil {
		if err := os.Rename(settingsPath, backupPath); err != nil {
			return fmt.Errorf("failed to back up settings: %w", err)
		}
	}
	return nil
}

func (settings *Settings) AddApp(app AppSetting) {
	settings.Apps = append(settings.Apps, app)
	settings.sortApps()
}

func (settings *Settings) RemoveApp(appSettingIdx int) {
	settings.Apps = slices.Delete(settings.Apps, appSettingIdx, appSettingIdx+1)
}

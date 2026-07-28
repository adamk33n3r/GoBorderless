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

// AppMatcher is the identity half of an App Config and each entry under a Layout Config.
type AppMatcher struct {
	WindowName string    `json:"windowName"`
	ExePath    string    `json:"exePath"`
	MatchType  MatchType `json:"matchType"`
	AutoApply  bool      `json:"autoApply"`
	PreOffsetX int32     `json:"preOffsetX"`
	PreOffsetY int32     `json:"preOffsetY"`
	PreWidth   int32     `json:"preWidth"`
	PreHeight  int32     `json:"preHeight"`
}

// AppConfig is a stand-alone per-app entry: matcher identity plus its own geometry/flags.
// Anonymous embed keeps the "apps" JSON shape flat (same keys as before the rename).
type AppConfig struct {
	AppMatcher
	BlackOverlay bool  `json:"blackOverlay"`
	HideTaskbar  bool  `json:"hideTaskbar"`
	Monitor      int   `json:"monitor"`
	OffsetX      int32 `json:"offsetX"`
	OffsetY      int32 `json:"offsetY"`
	Width        int32 `json:"width"`
	Height       int32 `json:"height"`
}

func (as AppConfig) Display() string {
	return fmt.Sprintf("%s | %s", as.WindowName, as.ExePath)
}

// LayoutConfig is shared geometry/flags plus a list of App Matchers.
type LayoutConfig struct {
	Name         string       `json:"name"`
	Monitor      int          `json:"monitor"`
	OffsetX      int32        `json:"offsetX"`
	OffsetY      int32        `json:"offsetY"`
	Width        int32        `json:"width"`
	Height       int32        `json:"height"`
	BlackOverlay bool         `json:"blackOverlay"`
	HideTaskbar  bool         `json:"hideTaskbar"`
	Matchers     []AppMatcher `json:"matchers"`
}

type AppConfigDefaults struct {
	Monitor      int       `json:"monitor"`
	MatchType    MatchType `json:"matchType"`
	BlackOverlay bool      `json:"blackOverlay"`
	HideTaskbar  bool      `json:"hideTaskbar"`
	OffsetX      int32     `json:"offsetX"`
	OffsetY      int32     `json:"offsetY"`
	Width        int32     `json:"width"`
	Height       int32     `json:"height"`
}

type Settings struct {
	Apps             []AppConfig       `json:"apps"`
	Layouts          []LayoutConfig    `json:"layouts"`
	Theme            string            `json:"theme"`
	StartWithWindows bool              `json:"startWithWindows"`
	CloseToTray      bool              `json:"closeToTray"`
	MinimizeToTray   bool              `json:"minimizeToTray"`
	StartMinimized   bool              `json:"startMinimized"`
	Defaults         AppConfigDefaults `json:"defaults"`
}

func newSettings() *Settings {
	return &Settings{
		Apps:             make([]AppConfig, 0),
		Layouts:          make([]LayoutConfig, 0),
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
	// A file containing literal "null" unmarshals cleanly into a nil pointer.
	if settings == nil {
		return newSettings(), nil
	}
	if settings.Apps == nil {
		settings.Apps = make([]AppConfig, 0)
	}
	if settings.Layouts == nil {
		settings.Layouts = make([]LayoutConfig, 0)
	}
	for i := range settings.Layouts {
		if settings.Layouts[i].Matchers == nil {
			settings.Layouts[i].Matchers = make([]AppMatcher, 0)
		}
		sortMatchers(settings.Layouts[i].Matchers)
	}
	settings.sortApps()
	return settings, nil
}

func (settings *Settings) sortApps() {
	slices.SortFunc(settings.Apps, func(a AppConfig, b AppConfig) int {
		return strings.Compare(a.WindowName, b.WindowName)
	})
}

func sortMatchers(matchers []AppMatcher) {
	slices.SortFunc(matchers, func(a AppMatcher, b AppMatcher) int {
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

func (settings *Settings) AddApp(app AppConfig) {
	settings.Apps = append(settings.Apps, app)
	settings.sortApps()
}

func (settings *Settings) RemoveApp(appConfigIdx int) {
	settings.Apps = slices.Delete(settings.Apps, appConfigIdx, appConfigIdx+1)
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// useTempSettings redirects the package-level settings location at a temp dir so
// tests never touch the real Documents\GoBorderless\settings.json.
func useTempSettings(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	origFolder, origPath := appFolder, settingsPath
	appFolder = dir
	settingsPath = filepath.Join(dir, "settings.json")
	t.Cleanup(func() {
		appFolder, settingsPath = origFolder, origPath
	})
	return dir
}

func writeSettingsFile(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(settingsPath, []byte(content), 0666); err != nil {
		t.Fatalf("writing settings file: %v", err)
	}
}

func TestMatchTypeString(t *testing.T) {
	tests := []struct {
		matchType MatchType
		want      string
	}{
		{MatchWindowTitle, "Window Title"},
		{MatchExePath, "Exe Path"},
		{MatchBoth, "Both"},
		{MatchEither, "Either"},
		{MatchType(42), "Unknown"},
		{MatchType(-1), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.matchType.String(); got != tt.want {
			t.Errorf("MatchType(%d).String() = %q, want %q", tt.matchType, got, tt.want)
		}
	}
}

func TestGetMatchTypeFromString(t *testing.T) {
	for _, mt := range []MatchType{MatchWindowTitle, MatchExePath, MatchBoth, MatchEither} {
		if got := GetMatchTypeFromString(mt.String()); got != mt {
			t.Errorf("round trip of %v gave %v", mt, got)
		}
	}

	// The radio group can only ever hand back one of matchTypes, so anything
	// else falls back to the safest option rather than an invalid enum value.
	for _, s := range []string{"", "Unknown", "window title"} {
		if got := GetMatchTypeFromString(s); got != MatchWindowTitle {
			t.Errorf("GetMatchTypeFromString(%q) = %v, want MatchWindowTitle", s, got)
		}
	}
}

func TestMatchTypesCoversEveryEnumValue(t *testing.T) {
	if len(matchTypes) != int(MatchEither)+1 {
		t.Fatalf("matchTypes has %d entries but the enum has %d values; the radio group and String() would disagree",
			len(matchTypes), int(MatchEither)+1)
	}
}

// MatchType is persisted as its integer value, so the const order in
// settings.go is a wire format. Changing it silently repoints every saved
// config at a different match rule.
func TestMatchTypeJSONWireFormatIsStable(t *testing.T) {
	want := map[MatchType]int{
		MatchWindowTitle: 0,
		MatchExePath:     1,
		MatchBoth:        2,
		MatchEither:      3,
	}

	for matchType, wantValue := range want {
		encoded, err := json.Marshal(AppSetting{MatchType: matchType})
		if err != nil {
			t.Fatalf("marshalling: %v", err)
		}

		var decoded struct {
			MatchType int `json:"matchType"`
		}
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("unmarshalling: %v", err)
		}
		if decoded.MatchType != wantValue {
			t.Errorf("%v encodes as %d, want %d", matchType, decoded.MatchType, wantValue)
		}
	}
}

func TestAppSettingDisplay(t *testing.T) {
	app := AppSetting{WindowName: "Some Game", ExePath: `C:\Games\game.exe`}

	if got, want := app.Display(), `Some Game | C:\Games\game.exe`; got != want {
		t.Errorf("Display() = %q, want %q", got, want)
	}
}

func TestNewSettingsDefaults(t *testing.T) {
	s := newSettings()

	if s.Apps == nil {
		t.Error("Apps is nil; JSON would encode null instead of []")
	}
	if len(s.Apps) != 0 {
		t.Errorf("Apps has %d entries, want 0", len(s.Apps))
	}
	if s.Theme != "System" {
		t.Errorf("Theme = %q, want %q", s.Theme, "System")
	}
	if s.StartWithWindows {
		t.Error("StartWithWindows should default to false")
	}
}

func TestLoadSettingsCreatesDefaultsWhenMissing(t *testing.T) {
	useTempSettings(t)

	settings, err := loadSettings()
	if err != nil {
		t.Fatalf("loadSettings() error = %v, want nil for a missing file", err)
	}
	if settings == nil {
		t.Fatal("loadSettings() returned nil settings")
	}
	if len(settings.Apps) != 0 || settings.Theme != "System" {
		t.Errorf("got %+v, want defaults", settings)
	}
	if _, err := os.Stat(appFolder); err != nil {
		t.Errorf("app folder was not created: %v", err)
	}
}

func TestLoadSettingsReadsExistingFile(t *testing.T) {
	useTempSettings(t)
	writeSettingsFile(t, `{
		"apps": [
			{"windowName": "Zed Game", "exePath": "C:\\zed.exe", "matchType": 3, "autoApply": true,
			 "monitor": 2, "offsetX": 10, "offsetY": 20, "width": 1920, "height": 1080,
			 "preOffsetX": 1, "preOffsetY": 2, "preWidth": 800, "preHeight": 600}
		],
		"theme": "Dark",
		"startWithWindows": true,
		"closeToTray": true,
		"minimizeToTray": true,
		"startMinimized": true,
		"defaults": {"monitor": 1, "matchType": 2, "offsetX": 5, "offsetY": 6, "width": 2560, "height": 1440}
	}`)

	settings, err := loadSettings()
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	if len(settings.Apps) != 1 {
		t.Fatalf("loaded %d apps, want 1", len(settings.Apps))
	}
	app := settings.Apps[0]
	want := AppSetting{
		WindowName: "Zed Game", ExePath: `C:\zed.exe`, MatchType: MatchEither, AutoApply: true,
		Monitor: 2, OffsetX: 10, OffsetY: 20, Width: 1920, Height: 1080,
		PreOffsetX: 1, PreOffsetY: 2, PreWidth: 800, PreHeight: 600,
	}
	if app != want {
		t.Errorf("app = %+v, want %+v", app, want)
	}

	if settings.Theme != "Dark" || !settings.StartWithWindows || !settings.CloseToTray ||
		!settings.MinimizeToTray || !settings.StartMinimized {
		t.Errorf("top level settings not loaded: %+v", settings)
	}
	wantDefaults := AppSettingDefaults{Monitor: 1, MatchType: MatchBoth, OffsetX: 5, OffsetY: 6, Width: 2560, Height: 1440}
	if settings.Defaults != wantDefaults {
		t.Errorf("defaults = %+v, want %+v", settings.Defaults, wantDefaults)
	}
}

func TestLoadSettingsSortsApps(t *testing.T) {
	useTempSettings(t)
	writeSettingsFile(t, `{"apps": [
		{"windowName": "Zed"}, {"windowName": "Alpha"}, {"windowName": "Mid"}
	]}`)

	settings, err := loadSettings()
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	want := []string{"Alpha", "Mid", "Zed"}
	for i, name := range want {
		if settings.Apps[i].WindowName != name {
			t.Errorf("app %d = %q, want %q", i, settings.Apps[i].WindowName, name)
		}
	}
}

// A corrupt file must surface an error (main.go backs the file up when it sees
// one) while still handing back usable settings so the app can start.
func TestLoadSettingsOnCorruptFile(t *testing.T) {
	for name, content := range map[string]string{
		"truncated": `{"apps": [`,
		"garbage":   `not json at all`,
		"empty":     ``,
		"wrongType": `{"apps": "should be an array"}`,
	} {
		t.Run(name, func(t *testing.T) {
			useTempSettings(t)
			writeSettingsFile(t, content)

			settings, err := loadSettings()
			if err == nil {
				t.Error("expected an error for a corrupt file")
			}
			if settings == nil {
				t.Fatal("settings must not be nil even when loading fails")
			}
			if settings.Theme != "System" {
				t.Errorf("Theme = %q, want defaults after a failed load", settings.Theme)
			}
		})
	}
}

// "null" is valid JSON that unmarshals into a nil *Settings.
func TestLoadSettingsOnNullFile(t *testing.T) {
	useTempSettings(t)
	writeSettingsFile(t, `null`)

	settings, err := loadSettings()
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if settings == nil {
		t.Fatal("loadSettings() returned nil settings")
	}
	if settings.Theme != "System" {
		t.Errorf("Theme = %q, want defaults", settings.Theme)
	}
}

// Older settings files predate the newer fields; they must keep loading.
func TestLoadSettingsToleratesMissingAndUnknownFields(t *testing.T) {
	useTempSettings(t)
	writeSettingsFile(t, `{"apps": [{"windowName": "Game"}], "somethingFromTheFuture": 42}`)

	settings, err := loadSettings()
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if len(settings.Apps) != 1 {
		t.Fatalf("loaded %d apps, want 1", len(settings.Apps))
	}
	if settings.Apps[0].MatchType != MatchWindowTitle || settings.Apps[0].Monitor != 0 {
		t.Errorf("missing fields should be zero values, got %+v", settings.Apps[0])
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	useTempSettings(t)

	original := &Settings{
		Apps: []AppSetting{
			{WindowName: "Alpha", ExePath: `C:\a.exe`, MatchType: MatchBoth, AutoApply: true,
				Monitor: 1, OffsetX: -10, OffsetY: -20, Width: 1920, Height: 1080,
				PreOffsetX: 100, PreOffsetY: 200, PreWidth: 640, PreHeight: 480},
			{WindowName: "Beta", ExePath: `C:\b.exe`, MatchType: MatchExePath, Monitor: 2},
		},
		Theme:            "Light",
		StartWithWindows: true,
		CloseToTray:      true,
		MinimizeToTray:   false,
		StartMinimized:   true,
		Defaults:         AppSettingDefaults{Monitor: 2, MatchType: MatchEither, Width: 3840, Height: 2160},
	}

	if err := original.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := loadSettings()
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if !reflect.DeepEqual(original, loaded) {
		t.Errorf("round trip changed settings:\n got %+v\nwant %+v", loaded, original)
	}
}

func TestSaveWritesIndentedJSON(t *testing.T) {
	useTempSettings(t)

	settings := newSettings()
	settings.AddApp(AppSetting{WindowName: "Game"})
	if err := settings.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	if !strings.Contains(string(raw), "\n  \"apps\"") {
		t.Errorf("settings file is not indented for hand editing:\n%s", raw)
	}
}

func TestSaveOverwritesPreviousContents(t *testing.T) {
	useTempSettings(t)

	full := newSettings()
	full.AddApp(AppSetting{WindowName: "Game"})
	if err := full.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := newSettings().Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := loadSettings()
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if len(loaded.Apps) != 0 {
		t.Errorf("loaded %d apps after saving empty settings, want 0", len(loaded.Apps))
	}
}

func TestBackUpSettingsFileRotates(t *testing.T) {
	dir := useTempSettings(t)

	for i, wantBackup := range []string{"settings.json.bak", "settings.json.bak.1", "settings.json.bak.2"} {
		writeSettingsFile(t, `{"round":`+string(rune('0'+i))+`}`)

		if err := backUpSettingsFile(); err != nil {
			t.Fatalf("backUpSettingsFile() error = %v", err)
		}
		if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
			t.Error("settings file should have been moved aside")
		}

		backup := filepath.Join(dir, wantBackup)
		content, err := os.ReadFile(backup)
		if err != nil {
			t.Fatalf("expected backup %s: %v", wantBackup, err)
		}
		if !strings.Contains(string(content), string(rune('0'+i))) {
			t.Errorf("%s holds %q, want the file from round %d", wantBackup, content, i)
		}
	}
}

func TestBackUpSettingsFileWithNothingToBackUp(t *testing.T) {
	dir := useTempSettings(t)

	if err := backUpSettingsFile(); err != nil {
		t.Fatalf("backUpSettingsFile() error = %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("created %d files with no settings to back up, want 0", len(entries))
	}
}

func TestAddAppKeepsListSorted(t *testing.T) {
	settings := newSettings()

	for _, name := range []string{"Mid", "Zed", "Alpha"} {
		settings.AddApp(AppSetting{WindowName: name})
	}

	want := []string{"Alpha", "Mid", "Zed"}
	for i, name := range want {
		if settings.Apps[i].WindowName != name {
			t.Errorf("app %d = %q, want %q", i, settings.Apps[i].WindowName, name)
		}
	}
}

func TestRemoveApp(t *testing.T) {
	settings := newSettings()
	for _, name := range []string{"Alpha", "Mid", "Zed"} {
		settings.AddApp(AppSetting{WindowName: name})
	}

	settings.RemoveApp(1)

	if len(settings.Apps) != 2 {
		t.Fatalf("have %d apps, want 2", len(settings.Apps))
	}
	if settings.Apps[0].WindowName != "Alpha" || settings.Apps[1].WindowName != "Zed" {
		t.Errorf("after removing index 1 got %q and %q", settings.Apps[0].WindowName, settings.Apps[1].WindowName)
	}
}

func TestRemoveAppAtEnds(t *testing.T) {
	settings := newSettings()
	for _, name := range []string{"Alpha", "Mid", "Zed"} {
		settings.AddApp(AppSetting{WindowName: name})
	}

	settings.RemoveApp(0)
	settings.RemoveApp(len(settings.Apps) - 1)

	if len(settings.Apps) != 1 || settings.Apps[0].WindowName != "Mid" {
		t.Errorf("remaining apps = %+v, want just Mid", settings.Apps)
	}
}

func TestSortAppsIsStableAcrossSaves(t *testing.T) {
	useTempSettings(t)

	settings := newSettings()
	settings.AddApp(AppSetting{WindowName: "Zed"})
	settings.AddApp(AppSetting{WindowName: "Alpha"})
	if err := settings.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := loadSettings()
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	if !reflect.DeepEqual(settings.Apps, loaded.Apps) {
		t.Errorf("app order changed across a save/load:\n got %+v\nwant %+v", loaded.Apps, settings.Apps)
	}
}

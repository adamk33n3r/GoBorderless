# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

**Run (development):**
```sh
go run .
```

**Live reload with air:**
```sh
air
```
Uses `.air.toml` — builds via `fyne package --exe ./tmp/GoBorderless.exe` on file changes.

**Build release package:**
```sh
fyne package --release
```
Requires `fyne` CLI: `go install fyne.io/tools/cmd/fyne@latest`
Note: The project uses a forked Fyne (`github.com/adamk33n3r/fyne/v2`) via a `go.mod` replace directive for correct icon support.

## Architecture

This is a Windows-only desktop application written in Go using the [Fyne](https://fyne.io/) GUI framework. It removes window borders from running applications to create a borderless windowed mode.

### Core data flow

1. `main.go` — Entry point. Loads `Settings`, enumerates monitors, launches `scanWindows` as a goroutine, then calls `buildApp` to start the Fyne UI.
2. `scanWindows` (in `main.go`) — Goroutine that runs every second: enumerates all visible windows, filters them, sends updates via `chWindowList` channel, and auto-applies borderless mode for any `AppSetting` with `AutoApply` enabled.
3. `chWindowList` (channel) → `windowObs` (Observable) — The window list channel is wrapped in a custom `rx.Observable` (see `rx/observable.go`) so multiple subscribers (main list, app config dialog) can receive updates.

### Key files

- [main.go](main.go) — `Window` struct, window enumeration/filtering, `scanWindows` goroutine, `matchWindow` logic
- [borderless.go](borderless.go) — `makeBorderless`, `restoreWindow`, `isBorderless` — manipulates `WS_CAPTION`/`WS_THICKFRAME` window styles via Win32
- [overlay.go](overlay.go) — Black overlay (letterbox) feature: creates a Win32 `WS_POPUP` window covering the full monitor with a transparent hole cut via `SetWindowRgn`/`CombineRgn(RGN_DIFF)` where the app sits; managed per app HWND in `activeOverlays` map; lifecycle (hide on minimize, destroy on app close/restore) handled in `scanWindows`
- [winapi.go](winapi.go) — All Win32 API wrappers: window style get/set, process path lookup, monitor enumeration, mutex management, Documents folder path, `setWindowRgn`
- [settings.go](settings.go) — `Settings` and `AppSetting` structs, JSON persistence to `~/Documents/GoBorderless/settings.json`, `MatchType` enum (WindowTitle, ExePath, Both, Either)
- [gui_app.go](gui_app.go) — `buildApp`: constructs the main Fyne window with a 3-tab layout (Apps, Defaults, Settings), list of app configs, auto-update logic via `go-selfupdate`
- [gui_appsetting.go](gui_appsetting.go) — `makeAppSettingWindow`: the "New/Edit App Config" dialog
- [gui_settings.go](gui_settings.go) — Settings tab UI, Windows startup registry key management
- [gui_defaults.go](gui_defaults.go) — Defaults tab UI
- [theme.go](theme.go) — Custom `forcedVariant` theme wrapper for Light/Dark/System theme support
- [rx/observable.go](rx/observable.go) — Minimal custom Rx-style observable with fan-out from a channel to multiple subscribers
- [ui/appsettingrow.go](ui/appsettingrow.go) — Custom Fyne widget for a row in the app list (title, auto-apply checkbox, apply/restore/edit/delete buttons)
- [ui/select.go](ui/select.go) — Generic typed `Select` widget wrapper
- [res/bundled.go](res/bundled.go) — Fyne-bundled resources (icon etc.)

### Settings persistence

Settings are saved as JSON at `~/Documents/GoBorderless/settings.json`. On corrupt/unreadable settings, the file is backed up with a `.bak` suffix and fresh defaults are used.

### Single-instance enforcement

A named Windows mutex (`GoBorderless_InstanceMutex`) is created on startup. If it already exists, the running instance's window is brought to foreground and the new process exits.

### Window matching

Each `AppSetting` has a `MatchType` that controls how a running window is matched: by window title, exe path, both, or either.

### Black overlay (letterbox)

`AppSetting.BlackOverlay` (bool) enables a solid black overlay that covers the monitor area around the app, hiding the desktop. Implemented as a single borderless `WS_POPUP | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE` Win32 window covering the entire monitor, with a transparent hole punched using `SetWindowRgn` + `CombineRgn(RGN_DIFF)`. Clicking the black area calls `SetForegroundWindow` on the app. The overlay is placed just below the app in z-order via `SetWindowPos(overlayHwnd, appHwnd, ...)`. Lifecycle is driven by `scanWindows`: destroyed when the app closes or Restore is pressed, hidden when minimized, shown when restored. Each overlay runs its own Win32 message loop in a goroutine pinned to an OS thread via `runtime.LockOSThread`. Active overlays are tracked in `activeOverlays map[win.HWND]*Overlay` (keyed by app HWND), protected by `overlayMu`. A global default (`AppSettingDefaults.BlackOverlay`) is configurable in the Defaults tab.

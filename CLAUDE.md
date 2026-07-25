# CLAUDE.md

Guidance for AI assistants working in this repository.

## What this is

GoBorderless is a **Windows-only desktop app** (Go + [Fyne](https://fyne.io/)) that strips
borders/title bars off other applications' windows and repositions/resizes them — mainly for games
with poor alt-tab behavior. Users create per-app "configs" that can be applied manually or
auto-applied once a second by a background scanner. Ships as a single portable `GoBorderless.exe`.

Module: `github.com/adamk33n3r/GoBorderless` · Go 1.24 · License: GPL-3.0

## Repository layout

```
main.go              Entry point: settings load, single-instance mutex, monitor enum,
                     scanWindows() goroutine, buildApp(). Also Window type + match logic
                     and ALWAYS_HIDDEN_PROCESSESS filter list.
borderless.go        isBorderless / makeBorderless / restoreWindow — the actual window-style work.
winapi.go            All Win32 interop: EnumWindows, window text/rect/style, monitor enum,
                     SHGetKnownFolderPath, mutex helpers. Mix of lxn/win and raw LazyDLL procs.
settings.go          Settings/AppSetting/AppSettingDefaults structs, MatchType enum,
                     JSON load/save/backup.
theme.go             forcedVariant — wraps a fyne.Theme to pin light/dark.
util.go              Tiny generic helpers (firstInSlice).
gui_app.go           Main window, app list, tray/minimize/close handling, self-update flow.
gui_appsetting.go    New/Edit app-config dialog.
gui_defaults.go      "Defaults" tab.
gui_settings.go      "Settings" tab (theme, tray options, Run-at-startup registry key).
ui/                  Reusable widgets: generic Select[T], AppSettingRow list row, MuxerPanel (stub).
rx/                  Hand-rolled minimal observable (multicast fan-out over channels).
res/                 Icons + generated bundled.go (fyne bundle output) + bundle.bat.
other/               Standalone `package main` repro for a past EnumWindows/slice bug.
                     NOT part of the app; don't wire it into the build.
assets/              Screenshots and source art for the README. Not compiled.
```

The root package is a single flat `package main`; the `gui_*.go` split is by screen, not by layer.

## Build, run, lint

All real building happens **on Windows** — the app needs cgo (OpenGL/GLFW via Fyne) and Win32 APIs.

```sh
go run .                                     # quick dev run (Windows)
air                                          # live-reload dev loop, uses .air.toml
fyne package --release                       # release build -> GoBorderless.exe (build.bat does this)
go install fyne.io/tools/cmd/fyne@latest     # but see the fork note below
res\bundle.bat                               # regenerate res/bundled.go from the icons
```

Important environment facts:

- **The `fyne` CLI must come from [adamk33n3r/fyne-tools](https://github.com/adamk33n3r/fyne-tools)**
  until the `.ico` icon support is merged upstream. The release workflow clones and
  `go install ./cmd/fyne` from that fork.
- `go.mod` has `replace fyne.io/fyne/v2 v2.6.1 => github.com/adamk33n3r/fyne/v2 v2.7.0` — a
  personal Fyne fork. Do not drop this replace when bumping deps.
- On Linux/macOS you can only type-check the pure-Go packages:
  `GOOS=windows go build ./ui/... ./rx/... ./res/...` succeeds; `go build ./...` (root package)
  fails with `build constraints exclude all Go files in .../go-gl/gl` because cross-compiling the
  GL driver needs a Windows cgo toolchain. That failure is expected in this container — it is not
  a code problem you introduced.
- There are **no tests** in the repo (`*_test.go` count: 0). Don't claim tests pass; if you add
  logic that is testable without Win32, adding tests is welcome but keep them OS-independent.
- Lint: `golangci-lint` (staticcheck is the configured lint tool in `.vscode/settings.json`). No
  config file is committed, so defaults apply. Keep `gofmt` clean — `gofmt -l .` must print nothing.
- `unsafe.Pointer` conversions that govet flags legitimately carry `//nolint:govet`; follow that
  pattern rather than restructuring correct Win32 interop to appease the linter.

## Release process

1. Bump `Version` (and `Build`) in `FyneApp.toml`. This is the only place the version lives; the
   UI reads it via `fyneApp.Metadata().Version` and the self-updater compares against it.
2. Push a tag matching `v*.*.*`. `.github/workflows/build-and-release.yml` runs on
   `windows-latest`, generates `CHANGELOG.md` from `git log` since the previous tag, builds with
   the forked `fyne` CLI, and publishes `GoBorderless.exe` to a GitHub Release.
3. Running instances pick it up through `checkUpdate` in `gui_app.go` (go-selfupdate against the
   `adamk33n3r/GoBorderless` slug, filtering assets by `goborderless.exe`).

Release asset naming and the tag pattern are load-bearing for auto-update — don't rename the exe.

## Architecture notes that matter when editing

**Window scan loop.** `scanWindows(settings)` runs in a goroutine forever: enumerate all HWNDs →
`getWindowData` filters (invisible, empty title, zero-size, blacklisted process/title) → push the
slice onto `chWindowList` → apply borderless to every config with `AutoApply` → sleep 1s.

**rx fan-out.** `windowObs = rx.FromChannel(chWindowList)` (package-level in `gui_app.go`) multicasts
that slice. Channels are unbuffered and the fan-out send is synchronous, so **a subscriber callback
that blocks stalls the whole scan loop**. Subscriptions must be `Unsubscribe()`d — the app-setting
dialog does this on both Confirm and Cancel; leaking one leaks a goroutine and slows every scan.
`rx` has no replay/subject semantics, so a new subscriber sees nothing until the next tick — that's
why the dialog also reads `currentWindows` directly at construction time.

**UI-thread rules (Fyne).** Any widget mutation from a non-UI goroutine must be wrapped in
`fyne.Do(...)` or `fyne.DoAndWait(...)`. The observable callbacks run on rx goroutines, so anything
touching widgets there needs the wrapper. See the `windowObs.Subscribe` in `makeAppSettingWindow`
and the minimize-watcher in `handleWindowsInit`.

**Shared mutable state.** `currentWindows` is guarded by `currentWindowsMutex` — lock around every
read/write. `monitors`, `settingsList`, `MUTEX`, and the widget vars at the top of
`gui_appsetting.go` are package-level globals; the app-setting dialog is effectively a singleton
because of them. If you ever allow two dialogs at once, those globals must move into a struct first.

**Settings persistence.** Lives at `%USERPROFILE%\Documents\GoBorderless\settings.json`. The code
saves eagerly — nearly every checkbox/entry `OnChanged` calls `settings.Save()`. Keep that habit.
A corrupt file is rotated to `settings.json.bak[.N]` by `backUpSettingsFile()` at startup.

**JSON compatibility.** `MatchType` serializes as its integer value, so the const block order in
`settings.go` is a wire format — **append new match types at the end only**. Same caution for
renaming any `json:"..."` tag on `AppSetting`/`Settings`; old settings files must keep loading.

**Monitor indexing.** `Monitor.number` and `AppSetting.Monitor` are **1-based** (index into
`monitors` is `Monitor - 1`). `0` means "unset" and is backfilled from the primary monitor at
startup. Off-by-one here panics on `monitors[-1]`.

**Geometry save/restore.** Before making a window borderless, the current rect is captured into the
`Pre*` fields (`PreOffsetX/Y`, `PreWidth/Height`) and persisted, so Restore can put it back. Both
the manual Apply button and the auto-apply loop do this, and both guard it with `!isBorderless(win)`
so an already-borderless window doesn't overwrite the saved original geometry. Preserve that guard.

**Borderless detection** is style-based: a window counts as bordered when it has `WS_CAPTION` plus
`WS_BORDER` or `WS_THICKFRAME`. The same expression drives the "Filter out borderless applications"
checkbox in the dialog — keep the two in sync if you change it.

**Single instance.** A named mutex (`GoBorderless_InstanceMutex`) is created at startup; on
`ERROR_ALREADY_EXISTS` the app finds the existing window by title and foregrounds it instead of
starting. `restartApp` must `closeMutex(MUTEX)` before spawning the new process, or the restart
silently no-ops.

**Reflection hack.** `setOnFocusChanged` in `gui_appsetting.go` writes Fyne's unexported
`widget.Entry.onFocusChanged` via `reflect` + `unsafe` to get select-all-on-focus. This will break
on Fyne upgrades — if entries stop selecting on focus after a dep bump, look here first.

**Generated code.** `res/bundled.go` is `fyne bundle` output ("DO NOT EDIT"); regenerate with
`res/bundle.bat`. `fyne_metadata_init.go` appears during `fyne package` and is a build artifact —
don't commit it (air already excludes it from watching).

## Conventions

- Commit messages mostly follow Conventional Commits (`feat:`, `fix:`, `chore:`); recent history is
  consistent about it, so match that.
- Windows-specific interop belongs in `winapi.go`, not scattered through GUI files.
- Reusable widgets go in `ui/` and must not import the root package (it's `package main`) — pass
  data in via generics/callbacks, as `ui.Select[T SelectOption]` does with `fmt.Stringer`.
- `ui.Select[T]` keeps its own `Selected *T` alongside the embedded `widget.Select`; use its
  `SetSelected`/`SetSelectedIndex`/`ClearSelected` so both stay in sync, never the embedded ones.
- Tooltips come from `github.com/dweymouth/fyne-tooltip` (`ttwidget.*`); the main window content is
  wrapped in `fynetooltip.AddWindowToolTipLayer` — new tooltip'd widgets need that layer present.
- User-visible strings are inline literals; there's no i18n layer.
- `fmt.Println` is the logging mechanism. Fine to keep, but don't add noisy per-frame logging inside
  the 1-second scan loop.

## Gotchas for non-Windows agents

You cannot run or fully compile this app in a Linux container. Verify changes with
`gofmt -l .` and, for the pure-Go packages, `GOOS=windows go build ./ui/... ./rx/... ./res/...` and
`golangci-lint run ./ui/... ./rx/...` (both clean as of this writing). The root package can't be
built or linted off Windows. Reason carefully about the Win32 paths instead of assuming a build check would have
caught it, and say plainly in your summary that a full Windows build/run was not performed.

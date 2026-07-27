# Layout Config extension points (current code)

Research against primary Go sources in this repo. Product decisions already fixed (Layout Configs alongside App Configs; App Config wins; layout list order; first matcher wins; no migration) are treated as given — this doc maps **today’s** paths and shapes, then names natural plug-in points. It does not choose product answers for open risks.

## Summary

Today there is a single config list: `Settings.Apps []AppSetting`. Each `AppSetting` owns matching identity (`WindowName` / `ExePath` / `MatchType`), Auto Apply, geometry (`Monitor` + monitor-relative offsets/size), flags (`BlackOverlay`, `HideTaskbar`), and Pre\* restore geometry (absolute screen coords). Matching is exact string equality via `matchWindow`. Manual Apply/Restore and Auto Apply in `scanWindows` both capture Pre\* then call `makeBorderless` / `restoreWindow`. Overlays are keyed by app HWND in a process-global map and driven by the same scan loop. Persistence is one JSON file under Documents. Layout Configs must either split the current `AppSetting` blob or add a parallel type and a resolution layer that still feeds the existing apply/overlay/taskbar functions.

## Data shapes

### `Window` (runtime, not persisted)

Source: [`main.go`](../../main.go) — `Window`

| Field | Role |
| --- | --- |
| `hwnd` | Win32 handle; overlay map key |
| `title` | From `GetWindowTextW` |
| `exePath` | Full process image path via `QueryFullProcessImageName` |

Built by `getWindowData` after `EnumWindows`: visible, non-empty title, non-zero size, process path resolvable, not in `ALWAYS_HIDDEN_PROCESSESS` ([`main.go`](../../main.go) `getWindowData`, `shouldHideWindow`).

### `MatchType`

Source: [`settings.go`](../../settings.go)

```go
MatchWindowTitle = 0  // "Window Title"
MatchExePath     = 1  // "Exe Path"
MatchBoth        = 2  // "Both"
MatchEither      = 3  // "Either"
```

JSON stores the numeric enum. UI strings via `MatchType.String` / `GetMatchTypeFromString` and package `matchTypes`.

### `AppSetting` (persisted App Config)

Source: [`settings.go`](../../settings.go) — `AppSetting`

| JSON field | Go field | Role today |
| --- | --- | --- |
| `windowName` | `WindowName` | Captured title; match identity |
| `exePath` | `ExePath` | Captured full exe path; match identity |
| `matchType` | `MatchType` | Which identity fields participate |
| `autoApply` | `AutoApply` | `scanWindows` applies when true |
| `blackOverlay` | `BlackOverlay` | Passed into `makeBorderless` → `createOverlay` |
| `hideTaskbar` | `HideTaskbar` | Contributes to global taskbar hide OR |
| `monitor` | `Monitor` | 1-based index into package `monitors` |
| `offsetX/Y` | `OffsetX/Y` | **Monitor-relative** position for apply |
| `width` / `height` | `Width` / `Height` | Target size for apply |
| `preOffsetX/Y` | `PreOffsetX/Y` | **Absolute screen** origin before apply |
| `preWidth` / `preHeight` | `PreWidth` / `PreHeight` | Size before apply |

`Display()` formats `"title | exePath"` for list rows and edit dialog titles ([`settings.go`](../../settings.go)).

**Coordinate asymmetry (important):** Apply uses `borderlessRect`: `OffsetX + monitor.left`, `OffsetY + monitor.top` ([`borderless.go`](../../borderless.go)). Pre\* are written from `getWindowRect` (`GetWindowRect` — screen space) and restored with `setWindowPos` using Pre\* unchanged ([`main.go`](../../main.go) / [`gui_app.go`](../../gui_app.go) / [`borderless.go`](../../borderless.go)).

### `AppSettingDefaults`

Source: [`settings.go`](../../settings.go) — `AppSettingDefaults`

Defaults for **new** App Config dialog fields and Defaults tab: `Monitor`, `MatchType`, `BlackOverlay`, `HideTaskbar`, `OffsetX/Y`, `Width`, `Height`. No Pre\*, no AutoApply, no WindowName/ExePath.

On startup, if `Defaults.Monitor == 0` or width/height are 0, primary monitor values are filled and settings are saved ([`main.go`](../../main.go) `main`).

### `Settings`

Source: [`settings.go`](../../settings.go) — `Settings`

| JSON field | Role |
| --- | --- |
| `apps` | Ordered (then alpha-sorted) App Config list |
| `theme` | UI theme string |
| `startWithWindows`, `closeToTray`, `minimizeToTray`, `startMinimized` | App chrome / startup |
| `defaults` | `AppSettingDefaults` |

There is **no** layout list, matcher list, or order-preserving custom sort beyond `sortApps` (alphabetical by `WindowName`).

### `Overlay` (runtime)

Source: [`overlay.go`](../../overlay.go) — `Overlay`

```go
type Overlay struct {
    hwnd    win.HWND  // overlay window
    appHwnd win.HWND  // target app
    monitor Monitor
}
```

Global: `activeOverlays map[win.HWND]*Overlay` keyed by **app** HWND, guarded by `overlayMu`.

### `Monitor` (runtime)

Source: [`winapi.go`](../../winapi.go) — `Monitor`

1-based `number` from enum order; `left`/`top`/`width`/`height` from `MONITORINFO.RcMonitor`; `isPrimary` from `MONITORINFOF_PRIMARY`. Package var `monitors` set once in `main` via `getMonitors()`.

## Matching

### Predicate

Source: [`main.go`](../../main.go) — `matchWindow`

Exact equality only (case-sensitive; no substring/wildcard). Confirmed by [`main_test.go`](../../main_test.go) `TestMatchWindow`, `TestMatchWindowIsExact`, empty-setting non-match.

| MatchType | Condition |
| --- | --- |
| `MatchWindowTitle` | `win.title == appSetting.WindowName` |
| `MatchExePath` | `win.exePath == appSetting.ExePath` |
| `MatchBoth` | title **and** exe |
| `MatchEither` | title **or** exe |
| unknown | `false` |

### Call sites

1. **Auto Apply** — nested loops in `scanWindows` ([`main.go`](../../main.go)).
2. **Manual Apply / Restore / Delete** — `firstInSlice(currentWindows, matchWindow)` ([`gui_app.go`](../../gui_app.go), helper [`util.go`](../../util.go) `firstInSlice`).
3. **Hide-taskbar second pass** — `matchWindow` + `isBorderless` ([`main.go`](../../main.go)).

### Window list feed

`scanWindows` → non-blocking send on `chWindowList` → `rx.FromChannel` as `windowObs` ([`gui_app.go`](../../gui_app.go)). Main list keeps `currentWindows` under `currentWindowsMutex`. New App Config dialog also subscribes to refresh the application select ([`gui_appsetting.go`](../../gui_appsetting.go)).

### Capture of identity

On **create**, selecting a running `Window` writes `appSetting.WindowName` and `ExePath` ([`gui_appsetting.go`](../../gui_appsetting.go)). Edit dialog does **not** show the application select — title/exe are not re-captured in UI; only match type, geometry, and flags are editable.

### Enumeration filters (not matchers)

`ALWAYS_HIDDEN_PROCESSESS` + `shouldHideWindow` remove system/browser/launcher/self from the pick list and scan data ([`main.go`](../../main.go)). Separate from App Config matching.

### Precedence today (relevant to “App Config wins / first matcher wins”)

- App list is **sorted by `WindowName`** on load and on `AddApp` ([`settings.go`](../../settings.go) `sortApps`). User list order is not preserved.
- Auto Apply: for each AutoApply config in `settings.Apps` order, first matching window in `windowData` wins **for that config**, then `break` (one window per config per tick). **Multiple configs can still apply to the same HWND** if several match — there is no cross-config “first wins” or App-vs-Layout winner.
- Manual Apply: first matching window in `currentWindows` order (`firstInSlice`).

## Manual Apply / Restore (including Pre\* persistence)

### Apply UI path

Source: [`gui_app.go`](../../gui_app.go) — `settingsList` update, `row.ApplyBtn.OnTapped`

1. Resolve `appSetting := settings.Apps[lii]`.
2. `win := firstInSlice(..., matchWindow)`.
3. If window not already borderless (`!isBorderless`):
   - Read `getWindowRect(win.hwnd)`.
   - Set `PreWidth/PreHeight/PreOffsetX/PreOffsetY`.
   - Write back `settings.Apps[lii]` and `settings.Save()`.
4. Call `makeBorderless(*win, appSetting)`.

Auto Apply checkbox disables Apply and Restore buttons when checked ([`gui_app.go`](../../gui_app.go)).

### Restore UI path

Same file, `row.RestoreBtn.OnTapped`:

1. Find matching window.
2. `restoreWindow(*win, appSetting)` — no Save (Pre\* already on disk).

### Delete path

`DeleteBtn`: if a matching window exists, `restoreWindow` first, then `RemoveApp` + `Save` ([`gui_app.go`](../../gui_app.go)).

### `makeBorderless`

Source: [`borderless.go`](../../borderless.go)

1. Strip caption/thickframe via `borderlessStyle` + `setWindowStyle`.
2. Resolve monitor index `Monitor - 1`; **out-of-range → return early** (style already changed; no move/overlay).
3. `setWindowPos` to `borderlessRect` (monitor-relative offsets).
4. If `BlackOverlay`, `createOverlay(window.hwnd, appSetting)`.

Idempotent on repeat calls ([`borderless_test.go`](../../borderless_test.go) `TestMakeBorderlessIsIdempotent`) — Auto Apply re-invokes every second.

### `restoreWindow`

Source: [`borderless.go`](../../borderless.go)

1. No-op if not borderless (protects against stale Pre\* teleport).
2. Restore style with `restoredStyle` (`WS_OVERLAPPEDWINDOW` OR).
3. `setWindowPos` with Pre\* absolute geometry.
4. `destroyOverlay(window.hwnd)`.
5. If `HideTaskbar`, call `restoreTaskbar()` immediately (may fight the next scan tick’s OR logic — see risks).

### Borderless detection

`isBorderlessStyle`: not (caption **and** (border **or** thickframe)) ([`borderless.go`](../../borderless.go)). Shared with App Config window filter (`getWindowsForSelect` keeps only bordered windows when filter is on) ([`gui_appsetting.go`](../../gui_appsetting.go)).

## Auto Apply in `scanWindows`

Source: [`main.go`](../../main.go) — `scanWindows`

Period: 1 second. Shared `*Settings` with the UI (same pointer from `main`).

Per tick:

1. Enumerate → `getWindowData` → push to `chWindowList` (drop if buffer full).
2. **Auto Apply loop:** for each `settings.Apps[i]` with `AutoApply`:
   - For each window, if `matchWindow`:
     - If not borderless: capture Pre\* into that index, `settings.Save()`.
     - `makeBorderless(win, appSetting)` (always, including already-borderless).
     - If `HideTaskbar`, set `shouldHideTaskbar = true`.
     - `break` (stop searching windows for **this** config).
3. **Hide-taskbar for manual-applied apps:** for each config with `HideTaskbar`, if matched window is borderless, set `shouldHideTaskbar`.
4. `hideTaskbar()` or `restoreTaskbar()` based on OR of flags.
5. **Overlay lifecycle** (below).
6. Sleep 1s.

Pre\* is only captured when transitioning from bordered → applying; once borderless, Auto Apply keeps repositioning via `makeBorderless` without rewriting Pre\*.

## Black overlay lifecycle

Sources: [`overlay.go`](../../overlay.go), call sites in [`borderless.go`](../../borderless.go) and [`main.go`](../../main.go).

### Create

`makeBorderless` → `createOverlay(appHwnd, appSetting)` when `BlackOverlay`:

- Registers class once (`GoBorderlessOverlay`, black brush).
- Inserts placeholder in `activeOverlays` under mutex (duplicate create = no-op).
- Uses `monitors[appSetting.Monitor-1]` **without** the bounds check present in `makeBorderless`.
- Goroutine + `runtime.LockOSThread`: create `WS_POPUP` | `WS_EX_TOOLWINDOW` | `WS_EX_NOACTIVATE` covering monitor; `applyOverlayRegion` punches hole with `OffsetX/Y` + `Width/Height` (monitor-local); z-order just below app; `ShowWindow(SW_SHOWNOACTIVATE)`; message loop until quit.

Click on black: `SetForegroundWindow` on app HWND (`overlayWndProc`).

### Destroy

- `restoreWindow` → `destroyOverlay`.
- `scanWindows`: if app HWND missing from current `windowData` → `destroyOverlay`.
- `destroyOverlay`: delete map entry, `PostMessage(WM_CLOSE)` → `WM_DESTROY` → `PostQuitMessage`.

### Hide / show / z-order (scan-driven)

For each active overlay key:

| Condition | Action |
| --- | --- |
| App HWND not in scan list | `destroyOverlay` |
| `IsIconic(appHwnd)` | `hideOverlay` |
| Else | `showOverlay` + `syncOverlayZOrder` |

Region is **not** recomputed each tick — hole tracks config geometry at create time, not live window moves.

### Taskbar (related flag, not overlay)

Global auto-hide via AppBar API ([`borderless.go`](../../borderless.go) / [`winapi.go`](../../winapi.go)). Original state saved in `main` (`saveTaskbarState`) and on first hide. Exit path: `restoreTaskbarOnExit` from close intercept when not close-to-tray ([`gui_app.go`](../../gui_app.go)).

## Settings persistence

Sources: [`settings.go`](../../settings.go), startup in [`main.go`](../../main.go), tests in [`settings_test.go`](../../settings_test.go).

| Concern | Behavior |
| --- | --- |
| Path | `{Documents}/GoBorderless/settings.json` via `getDocumentsFolder` + `APP_NAME` |
| Load | `MkdirAll`; missing file → `newSettings()` (empty apps, Theme `"System"`); corrupt JSON → `newSettings()` **and** error (caller backs up); literal `null` → defaults |
| Backup | `backUpSettingsFile` renames to `.bak`, `.bak.1`, … on load error ([`main.go`](../../main.go)) |
| Save | `json.MarshalIndent` + write `0666`; called from UI handlers, Auto Apply Pre\* capture, and once after default-monitor fill at startup |
| Sort | `loadSettings` and `AddApp` call `sortApps` (alphabetical `WindowName`) |
| Forward/back compat | Unknown JSON fields ignored by `encoding/json`; missing fields zero-value ([`settings_test.go`](../../settings_test.go) `TestLoadSettingsToleratesMissingAndUnknownFields`) |

`RemoveApp` does not re-sort (order remains alpha among remaining). No schema version field.

## Natural extension points for Layout Configs

Given decided product shape (Layout owns name + geometry + black overlay + hide taskbar; App Matchers own title/exe/match/auto/Pre\*; App Config wins; layout list order; first matcher wins; no migration):

### 1. Split or compose the config model (`settings.go`)

Today one struct mixes matcher fields and layout fields. Natural split:

- **Layout-owned (decided):** name, `Monitor`/`Offset*`/`Width`/`Height`, `BlackOverlay`, `HideTaskbar`.
- **Matcher-owned (decided):** `WindowName`, `ExePath`, `MatchType`, `AutoApply`, Pre\*.

Stand-alone App Config can remain a flat `AppSetting` (current shape) for no-migration, or become “matcher + inline layout.” Layout Config is a new persisted type + `Settings.Layouts` (or similar) array. `AppSettingDefaults` today seeds **both** geometry and match defaults — Defaults tab ([`gui_defaults.go`](../../gui_defaults.go)) will need a clear story for what still applies to Layout vs App Matcher create flows (open risk, not decided here).

### 2. Resolution layer before apply (`matchWindow` + scan/UI find)

Extract a function that, given a `Window` (or for Auto Apply, given all windows), returns the winning **apply payload** (geometry + flags + Pre\* storage target):

- Prefer matching App Config over any Layout matcher (product).
- Among layouts: list order; among matchers in a layout: first wins (product).
- Today’s `matchWindow(Window, AppSetting)` can stay as the identity predicate if matchers expose the same four fields; callers pass a thin view instead of full `AppSetting`.

Plug sites:

- `scanWindows` Auto Apply loop ([`main.go`](../../main.go) ~189–211).
- Apply / Restore / Delete in [`gui_app.go`](../../gui_app.go).
- Hide-taskbar aggregation loops in `scanWindows`.

### 3. Keep `makeBorderless` / `restoreWindow` / overlay API shape

Those functions already take `(Window, AppSetting)` but only use geometry, `BlackOverlay`, `HideTaskbar`, and Pre\* ([`borderless.go`](../../borderless.go)). Natural extension: a smaller struct (or interface) for “apply parameters” so Layout geometry + matcher Pre\* can be assembled without inventing a second apply pipeline. `createOverlay` / `applyOverlayRegion` already only need monitor + hole rect from that payload ([`overlay.go`](../../overlay.go)).

### 4. Pre\* persistence target

Pre\* today live on the `AppSetting` row and are written via index into `settings.Apps` ([`main.go`](../../main.go), [`gui_app.go`](../../gui_app.go)). For Layout matchers, Pre\* must hang off the **matcher** (per product) and Save must update that nested object. Same capture gate: only when `!isBorderless`.

### 5. Ordering / sorting

`sortApps` alpha-sorts App Configs — conflicts with any “user list order / first wins” story for Apps, and Layouts will need **non-sorting** persistence of slice order. Extension: stop sorting layouts; decide whether Apps remain alpha (current) while layouts are ordered (product says layout list order).

### 6. UI surfaces

| Surface | File | Extension |
| --- | --- | --- |
| Apps tab list + Apply/Restore/Auto | [`gui_app.go`](../../gui_app.go), [`ui/appsettingrow.go`](../../ui/appsettingrow.go) | Parallel Layouts list / tab, or combined; row needs layout-aware Apply |
| New/Edit App Config dialog | [`gui_appsetting.go`](../../gui_appsetting.go) | Geometry fields may move to Layout editor; matcher dialog keeps capture + match + auto |
| Defaults | [`gui_defaults.go`](../../gui_defaults.go) | Geometry defaults likely feed Layout create; MatchType may feed Matcher create |
| Tabs shell | [`gui_app.go`](../../gui_app.go) `NewAppTabs` | Natural place for a Layouts tab alongside Apps |

Window picker + `windowObs` subscription pattern for capturing title/exe is reusable for Layout matchers ([`gui_appsetting.go`](../../gui_appsetting.go)).

### 7. Settings I/O

`loadSettings` / `Save` / `backUpSettingsFile` ([`settings.go`](../../settings.go)): add layout array with omitempty-friendly zero values so old files keep loading without migration. Avoid putting required new fields on existing `apps` entries.

### 8. Overlay / taskbar without structural change

Lifecycle remains HWND-keyed and scan-driven. Multiple matchers sharing one layout’s geometry still create one overlay per applied HWND (current map). Taskbar remains a global OR of “any winning config with HideTaskbar and (auto or already borderless)” — extend the loops to walk layouts after Apps with App-wins filtering.

## Open risks / ambiguities for later tickets

Do not invent product answers; these are gaps between decided Layout Config rules and current code:

1. **App Config list order vs alpha sort** — Product “first matcher wins” / App-over-Layout assumes ordered lists; Apps are always re-sorted by `WindowName` ([`settings.go`](../../settings.go) `sortApps`). Whether App Configs keep alphabetical order (and thus “wins” is name-order among Apps) is undecided.

2. **Same window matches multiple App Configs** — Auto Apply does not stop after the first config that matched a HWND; later matching configs also call `makeBorderless`. Layout “first wins” needs new cross-config arbitration that does not exist today.

3. **Where Defaults apply after the split** — `AppSettingDefaults` mixes geometry, MatchType, BlackOverlay, HideTaskbar ([`settings.go`](../../settings.go), [`gui_defaults.go`](../../gui_defaults.go)). Which defaults seed Layout create vs Matcher create vs remaining App Config create is unspecified.

4. **Editing App Config title/exe** — Edit path cannot re-pick the window ([`gui_appsetting.go`](../../gui_appsetting.go)); Layout matcher UX may need capture-only-on-add or a new re-capture control.

5. **`makeBorderless` monitor bounds vs overlay** — Invalid/zero `Monitor` returns after style strip without move ([`borderless.go`](../../borderless.go)); `createOverlay` indexes `monitors[Monitor-1]` unchecked ([`overlay.go`](../../overlay.go)). Layout configs with stale monitor indexes inherit this.

6. **Restore + HideTaskbar race** — Manual `restoreWindow` calls `restoreTaskbar()` immediately; next scan may re-hide if another config still wants it ([`borderless.go`](../../borderless.go), [`main.go`](../../main.go)). Multi-app / multi-layout HideTaskbar makes this more visible.

7. **Overlay region staleness** — Hole is fixed at create from config offsets; Auto Apply repositions the app every second but does not update the region. If Layout geometry is edited live while an overlay exists, behavior is undefined (destroy/recreate not automatic).

8. **Shared `*Settings` mutation** — UI and `scanWindows` mutate the same slice and call `Save` concurrently without a settings mutex. Nested layout/matcher updates increase contention risk.

9. **Pre\* absolute vs Offset\* relative** — Documented asymmetry; Layout editors must not treat Pre\* as monitor-relative when displaying “restore position.”

10. **Stand-alone App Config vs Layout ownership of BlackOverlay/HideTaskbar** — Product puts those flags on Layout; today’s App Config owns them on the same struct. Whether App Config keeps its own flags (and thus can diverge from any layout) after Layouts ship is a consistency question for implementation tickets.

11. **No schema version** — Forward-compatible JSON helps no-migration, but there is no explicit version field to gate future breaking changes ([`settings.go`](../../settings.go)).

## Key symbols index

| Symbol | File |
| --- | --- |
| `AppSetting`, `AppSettingDefaults`, `Settings`, `MatchType`, `loadSettings`, `Save`, `sortApps`, `AddApp`, `RemoveApp` | `settings.go` |
| `Window`, `matchWindow`, `scanWindows`, `getWindowData`, `shouldHideWindow` | `main.go` |
| `makeBorderless`, `restoreWindow`, `borderlessRect`, `isBorderless`, taskbar helpers | `borderless.go` |
| `Overlay`, `createOverlay`, `destroyOverlay`, `hideOverlay`, `showOverlay`, `syncOverlayZOrder`, `applyOverlayRegion`, `activeOverlays` | `overlay.go` |
| `Monitor`, `getMonitors`, `getWindowRect`, `setWindowPos`, `getDocumentsFolder` | `winapi.go` |
| `buildApp`, Apply/Restore/AutoApply list wiring, `windowObs` | `gui_app.go` |
| `makeAppSettingWindow` | `gui_appsetting.go` |
| `buildDefaultsTab` | `gui_defaults.go` |
| `firstInSlice` | `util.go` |
| `AppSettingRow` | `ui/appsettingrow.go` |
| Domain glossary (App Config / Layout Config / App Matcher) | `CONTEXT.md` |

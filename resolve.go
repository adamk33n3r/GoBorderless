package main

// ApplyPayload is the geometry, flags, and Pre* restore rect consumed by
// makeBorderless / restoreWindow. Built from an App Config or a Layout Config
// plus the winning App Matcher.
type ApplyPayload struct {
	Monitor      int
	OffsetX      int32
	OffsetY      int32
	Width        int32
	Height       int32
	BlackOverlay bool
	HideTaskbar  bool
	PreOffsetX   int32
	PreOffsetY   int32
	PreWidth     int32
	PreHeight    int32
}

func applyPayloadFromApp(app AppConfig) ApplyPayload {
	return ApplyPayload{
		Monitor:      app.Monitor,
		OffsetX:      app.OffsetX,
		OffsetY:      app.OffsetY,
		Width:        app.Width,
		Height:       app.Height,
		BlackOverlay: app.BlackOverlay,
		HideTaskbar:  app.HideTaskbar,
		PreOffsetX:   app.PreOffsetX,
		PreOffsetY:   app.PreOffsetY,
		PreWidth:     app.PreWidth,
		PreHeight:    app.PreHeight,
	}
}

func applyPayloadFromLayout(layout LayoutConfig, matcher AppMatcher) ApplyPayload {
	return ApplyPayload{
		Monitor:      layout.Monitor,
		OffsetX:      layout.OffsetX,
		OffsetY:      layout.OffsetY,
		Width:        layout.Width,
		Height:       layout.Height,
		BlackOverlay: layout.BlackOverlay,
		HideTaskbar:  layout.HideTaskbar,
		PreOffsetX:   matcher.PreOffsetX,
		PreOffsetY:   matcher.PreOffsetY,
		PreWidth:     matcher.PreWidth,
		PreHeight:    matcher.PreHeight,
	}
}

// applyWinner is the winning App Config or Layout matcher for one window.
// Exactly one of appIdx or layoutIdx is >= 0 when ok is true.
type applyWinner struct {
	payload   ApplyPayload
	appIdx    int // -1 when a layout wins
	layoutIdx int // -1 when an app wins
	matcher   AppMatcher
}

// resolveWinner returns the winning apply source for a window.
// App Configs beat any Layout Config; among layouts, list order; within a
// layout, first matcher after alphabetical sort by window title.
func resolveWinner(settings *Settings, win Window) (applyWinner, bool) {
	for i, app := range settings.Apps {
		if matchWindow(win, app.AppMatcher) {
			return applyWinner{payload: applyPayloadFromApp(app), appIdx: i, layoutIdx: -1}, true
		}
	}
	for layoutIdx, layout := range settings.Layouts {
		matchers := append([]AppMatcher(nil), layout.Matchers...)
		sortMatchers(matchers)
		for _, matcher := range matchers {
			if matchWindow(win, matcher) {
				return applyWinner{
					payload:   applyPayloadFromLayout(layout, matcher),
					appIdx:    -1,
					layoutIdx: layoutIdx,
					matcher:   matcher,
				}, true
			}
		}
	}
	return applyWinner{}, false
}

// resolveApply returns the winning apply payload for a window.
func resolveApply(settings *Settings, win Window) (ApplyPayload, bool) {
	winner, ok := resolveWinner(settings, win)
	if !ok {
		return ApplyPayload{}, false
	}
	return winner.payload, true
}

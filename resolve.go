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

// resolveApply returns the winning apply payload for a window.
// App Configs beat any Layout Config; among layouts, list order; within a
// layout, first matcher after alphabetical sort by window title.
func resolveApply(settings *Settings, win Window) (ApplyPayload, bool) {
	for _, app := range settings.Apps {
		if matchWindow(win, app.AppMatcher) {
			return applyPayloadFromApp(app), true
		}
	}
	for _, layout := range settings.Layouts {
		matchers := append([]AppMatcher(nil), layout.Matchers...)
		sortMatchers(matchers)
		for _, matcher := range matchers {
			if matchWindow(win, matcher) {
				return applyPayloadFromLayout(layout, matcher), true
			}
		}
	}
	return ApplyPayload{}, false
}

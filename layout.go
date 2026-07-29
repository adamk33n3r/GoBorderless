package main

import (
	"fmt"
	"strings"
)

// layoutApplyRestoreEnabled reports whether the layout-row Apply/Restore buttons
// should be enabled. Disabled when there are no matchers, or every matcher has
// Auto Apply on (nothing useful / safe for the row actions to do).
func layoutApplyRestoreEnabled(layout LayoutConfig) bool {
	for _, m := range layout.Matchers {
		if !m.AutoApply {
			return true
		}
	}
	return false
}

func (layout LayoutConfig) GeometrySubtitle() string {
	return fmt.Sprintf("Monitor %d · %d,%d %d×%d",
		layout.Monitor, layout.OffsetX, layout.OffsetY, layout.Width, layout.Height)
}

func (layout LayoutConfig) MatcherCountLabel() string {
	n := len(layout.Matchers)
	if n == 1 {
		return "1 matcher"
	}
	return fmt.Sprintf("%d matchers", n)
}

func layoutNameValid(name string) bool {
	return strings.TrimSpace(name) != ""
}

// capturePreGeometry writes absolute window rect into the matcher's Pre* fields.
func capturePreGeometry(matcher *AppMatcher, left, top, right, bottom int32) {
	matcher.PreOffsetX = left
	matcher.PreOffsetY = top
	matcher.PreWidth = right - left
	matcher.PreHeight = bottom - top
}

func copyMatcherPre(dst *AppMatcher, src AppMatcher) {
	dst.PreOffsetX = src.PreOffsetX
	dst.PreOffsetY = src.PreOffsetY
	dst.PreWidth = src.PreWidth
	dst.PreHeight = src.PreHeight
}

// findMatcherIndex locates a matcher in persisted layout order by identity.
func findMatcherIndex(matchers []AppMatcher, matcher AppMatcher) int {
	for i, m := range matchers {
		if m.WindowName == matcher.WindowName && m.ExePath == matcher.ExePath {
			return i
		}
	}
	return -1
}

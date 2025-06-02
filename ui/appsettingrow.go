package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

type AppSettingRow struct {
	widget.BaseWidget
	ListItemID widget.ListItemID
	// Selected   bool
	// Focused    bool
	// Hovered    bool

	Title      *ttwidget.Label
	AutoApply  *widget.Check
	ApplyBtn   *ttwidget.Button
	RestoreBtn *ttwidget.Button
	EditBtn    *ttwidget.Button
	DeleteBtn  *ttwidget.Button

	// OnTapped        func()
	// OnDoubleTapped  func()
	// OnFocusNeighbor func(up bool)

	// tappedAt      int64
	// focusedRect   canvas.Rectangle
	// selectionRect canvas.Rectangle
	// hoverRect     canvas.Rectangle
}

// func (asr *AppSettingRow) SetOnTapped(f func()) {
// 	asr.OnTapped = f
// }

// func (asr *AppSettingRow) ItemID() widget.ListItemID {
// 	return asr.ListItemID
// }

// func (asr *AppSettingRow) SetItemID(id widget.ListItemID) {
// 	asr.ListItemID = id
// }

// func (asr *AppSettingRow) EnsureUnfocused() {
// 	if asr.Focused {
// 		c := fyne.CurrentApp().Driver().CanvasForObject(asr)
// 		if c != nil {
// 			c.Unfocus()
// 		}
// 	}
// 	asr.Focused = false
// }

func (asr *AppSettingRow) Tapped(*fyne.PointEvent) {
	// prevTap := asr.tappedAt
	// asr.tappedAt = time.Now().UnixMilli()
	// if asr.tappedAt-prevTap < fyne.CurrentApp().Driver().DoubleTapDelay().Milliseconds() {
	// 	if asr.OnDoubleTapped != nil {
	// 		asr.OnDoubleTapped()
	// 	}
	// } else {
	// 	if asr.OnTapped != nil {
	// 		asr.OnTapped()
	// 	}
	// }
}

// func (asr *AppSettingRow) FocusGained() {
// 	asr.Focused = true
// 	asr.focusedRect.FillColor = theme.Color(theme.ColorNameHover)
// 	asr.focusedRect.Refresh()
// }

// func (asr *AppSettingRow) FocusLost() {
// 	asr.Focused = false
// 	asr.focusedRect.FillColor = color.Transparent
// 	asr.focusedRect.Refresh()
// }

// func (asr *AppSettingRow) MouseIn(e *desktop.MouseEvent) {
// 	fmt.Println("MouseIn")
// 	asr.Hovered = true
// 	asr.hoverRect.FillColor = theme.Color(theme.ColorNameHover)
// 	asr.hoverRect.Refresh()
// }

// func (asr *AppSettingRow) MouseMoved(e *desktop.MouseEvent) {
// }

// func (asr *AppSettingRow) MouseOut() {
// 	asr.Hovered = false
// 	asr.hoverRect.FillColor = color.Transparent
// 	asr.hoverRect.Refresh()
// }

// func (asr *AppSettingRow) Refresh() {
// 	asr.updateBackgroundRendering()
// 	asr.BaseWidget.Refresh()
// }

// func (asr *AppSettingRow) updateBackgroundRendering() {
// 	if asr.Selected {
// 		asr.selectionRect.FillColor = theme.Color(theme.ColorNameSelection)
// 	} else {
// 		asr.selectionRect.FillColor = color.Transparent
// 	}
// 	if asr.Focused {
// 		asr.focusedRect.FillColor = theme.Color(theme.ColorNameHover)
// 	} else {
// 		asr.focusedRect.FillColor = color.Transparent
// 	}
// 	if asr.Hovered {
// 		asr.hoverRect.FillColor = theme.Color(theme.ColorNameHover)
// 	} else {
// 		asr.hoverRect.FillColor = color.Transparent
// 	}
// }

func NewAppSettingRow() *AppSettingRow {
	label := ttwidget.NewLabel("template")
	label.SetToolTip("template")
	row := &AppSettingRow{
		Title:      label,
		AutoApply:  widget.NewCheck("Auto Apply", func(checked bool) {}),
		ApplyBtn:   ttwidget.NewButtonWithIcon("", theme.ContentRedoIcon(), func() {}),
		RestoreBtn: ttwidget.NewButtonWithIcon("", theme.ContentUndoIcon(), func() {}),
		EditBtn:    ttwidget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {}),
		DeleteBtn:  ttwidget.NewButtonWithIcon("", theme.DeleteIcon(), func() {}),
	}
	row.Title.Truncation = fyne.TextTruncateEllipsis
	row.ApplyBtn.SetToolTip("Apply")
	row.RestoreBtn.SetToolTip("Restore")
	row.EditBtn.SetToolTip("Edit")
	row.DeleteBtn.SetToolTip("Delete")
	row.ExtendBaseWidget(row)

	return row
}

func (row *AppSettingRow) CreateRenderer() fyne.WidgetRenderer {
	// row.selectionRect.CornerRadius = theme.SelectionRadiusSize()
	// row.focusedRect.CornerRadius = theme.SelectionRadiusSize()
	// row.hoverRect.CornerRadius = theme.SelectionRadiusSize()
	// row.updateBackgroundRendering()
	c := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(
			row.AutoApply,
			row.ApplyBtn,
			row.RestoreBtn,
			row.EditBtn,
			row.DeleteBtn,
		),
		row.Title,
	)
	return widget.NewSimpleRenderer(
		c,
	)
}

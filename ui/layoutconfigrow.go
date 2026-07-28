package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// LayoutConfigRow is one row in the Layouts tab list: name, geometry subtitle,
// matcher count, reorder, and Apply/Restore/Edit/Delete. No Auto Apply — that
// lives on individual App Matchers.
type LayoutConfigRow struct {
	widget.BaseWidget

	Title        *ttwidget.Label
	Subtitle     *widget.Label
	MatcherCount *widget.Label
	UpBtn        *ttwidget.Button
	DownBtn      *ttwidget.Button
	ApplyBtn     *ttwidget.Button
	RestoreBtn   *ttwidget.Button
	EditBtn      *ttwidget.Button
	DeleteBtn    *ttwidget.Button
}

func (row *LayoutConfigRow) Tapped(*fyne.PointEvent) {}

func NewLayoutConfigRow() *LayoutConfigRow {
	title := ttwidget.NewLabel("template")
	title.SetToolTip("template")
	title.TextStyle.Bold = true
	title.Truncation = fyne.TextTruncateEllipsis

	subtitle := widget.NewLabel("subtitle")
	subtitle.Truncation = fyne.TextTruncateEllipsis

	matcherCount := widget.NewLabel("0 matchers")

	row := &LayoutConfigRow{
		Title:        title,
		Subtitle:     subtitle,
		MatcherCount: matcherCount,
		UpBtn:        ttwidget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {}),
		DownBtn:      ttwidget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {}),
		ApplyBtn:     ttwidget.NewButtonWithIcon("", theme.ViewFullScreenIcon(), func() {}),
		RestoreBtn:   ttwidget.NewButtonWithIcon("", theme.ViewRestoreIcon(), func() {}),
		EditBtn:      ttwidget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {}),
		DeleteBtn:    ttwidget.NewButtonWithIcon("", theme.DeleteIcon(), func() {}),
	}
	row.UpBtn.SetToolTip("Move up")
	row.DownBtn.SetToolTip("Move down")
	row.ApplyBtn.SetToolTip("Apply")
	row.RestoreBtn.SetToolTip("Restore")
	row.EditBtn.SetToolTip("Edit")
	row.DeleteBtn.SetToolTip("Delete")
	row.ExtendBaseWidget(row)
	return row
}

func (row *LayoutConfigRow) CreateRenderer() fyne.WidgetRenderer {
	left := container.NewVBox(
		container.NewBorder(nil, nil, nil, row.MatcherCount, row.Title),
		row.Subtitle,
	)
	actions := container.NewHBox(
		row.UpBtn,
		row.DownBtn,
		row.ApplyBtn,
		row.RestoreBtn,
		row.EditBtn,
		row.DeleteBtn,
	)
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, nil, actions, left))
}

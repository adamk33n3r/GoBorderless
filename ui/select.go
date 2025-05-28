package ui

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2/widget"
)

type SelectOption interface {
	fmt.Stringer
}

type Select[T SelectOption] struct {
	widget.Select
	Options  []T
	Selected *T
}

func (s *Select[T]) SetOptions(options []T) {
	s.Options = options
	s.Select.Options = s.optionsAsStrings()
}
func (s *Select[T]) SetSelected(option T) {
	s.Select.SetSelected(option.String())
	s.Selected = &option
}
func (s *Select[T]) SetSelectedIndex(index int) {
	if index < 0 || index >= len(s.Options) {
		return
	}
	s.Select.SetSelected(s.Options[index].String())
	s.Selected = &s.Options[index]
}
func (s *Select[T]) ClearSelected() {
	s.Select.ClearSelected()
	s.Selected = nil
}
func (s *Select[T]) optionsAsStrings() []string {
	stringOptions := make([]string, len(s.Options))
	for i, option := range s.Options {
		stringOptions[i] = option.String()
	}
	return stringOptions
}

/**
 * Wrapper around fyne.widget.Select to genericize the options.
 */
func NewSelect[T SelectOption](options []T, changed func(T)) *Select[T] {
	s := &Select[T]{
		Options: options,
	}
	s.Select.Options = s.optionsAsStrings()
	s.OnChanged = func(item string) {
		index := slices.Index(s.Select.Options, item)
		if index != -1 {
			changed(s.Options[index])
			s.Selected = &s.Options[index]
		}
	}
	s.ExtendBaseWidget(s)
	return s
}

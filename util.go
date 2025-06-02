package main

import (
	"slices"
)

func firstInSlice[S ~[]E, E any](slice S, cb func(E) bool) *E {
	idx := slices.IndexFunc(slice, cb)
	if idx == -1 {
		return nil
	}
	return &slice[idx]
}

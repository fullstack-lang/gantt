package models

import "time"

type Bar struct {
	Name  string
	Start time.Time
	End   time.Time

	// if not Zero, is taken into account when drawing the bar
	OptionnalColor  string
	OptionnalStroke string
	FillOpacity     float64
	StrokeWidth     float64
	StrokeDashArray string
}

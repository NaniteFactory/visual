package actors

import (
	"image/color"

	"github.com/faiface/pixel"
	kuji "github.com/nanitefactory/amidakuji/glossary"
	"golang.org/x/image/colornames"
)

// FPSWatch implements Actor.
type FPSWatch struct {
	*kuji.FPSWatch
}

// NewFPSWatch is a constructor.
func NewFPSWatch(
	additionalCaption string, _pos pixel.Vec,
	_anchorY kuji.AnchorY, _anchorX kuji.AnchorX, // This is because the order is usually Y then X in spoken language.
	_colorBg, _colorTxt color.Color,
) *FPSWatch {
	return &FPSWatch{kuji.NewFPSWatch(additionalCaption, _pos, _anchorY, _anchorX, _colorBg, _colorTxt)}
}

// NewFPSWatchSimple is a constructor.
func NewFPSWatchSimple(_pos pixel.Vec, _anchorY kuji.AnchorY, _anchorX kuji.AnchorX) *FPSWatch {
	return NewFPSWatch("", _pos, _anchorY, _anchorX, colornames.Black, colornames.White)
}

// Update implements the Updater interface that kuji.FPSWatch lacks of.
func (watch *FPSWatch) Update(_ float64) {
	// empty.
}

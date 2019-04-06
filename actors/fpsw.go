package actors

import (
	"image/color"

	"github.com/faiface/pixel"
	"github.com/nanitefactory/visual/super"
	"golang.org/x/image/colornames"
)

// FPSWatch implements HUD.
type FPSWatch struct {
	*super.FPSWatch
	futureAnchorY super.AnchorY // what reflects on screen resize
	futureAnchorX super.AnchorX // what reflects on screen resize
}

// NewFPSWatch is a constructor.
func NewFPSWatch(
	additionalCaption string, _pos pixel.Vec,
	_anchorY super.AnchorY, _anchorX super.AnchorX, // This is because the order is usually Y then X in spoken language.
	_colorBg, _colorTxt color.Color,
) *FPSWatch {
	return &FPSWatch{
		FPSWatch:      super.NewFPSWatch(additionalCaption, _pos, _anchorY, _anchorX, _colorBg, _colorTxt),
		futureAnchorX: _anchorX,
		futureAnchorY: _anchorY,
	}
}

// NewFPSWatchSimple is a constructor.
func NewFPSWatchSimple(_pos pixel.Vec, _anchorY super.AnchorY, _anchorX super.AnchorX) *FPSWatch {
	return NewFPSWatch("", _pos, _anchorY, _anchorX, colornames.Black, colornames.White)
}

// Update implements the Updater interface that super.FPSWatch lacks of.
func (watch *FPSWatch) Update(_ float64) {
	// empty.
}

// PosOnScreen implements the HUD interface that super.FPSWatch lacks of.
func (watch *FPSWatch) PosOnScreen(width, height float64) {
	watch.SetPos(pixel.V(width, height), watch.futureAnchorY, watch.futureAnchorX)
}

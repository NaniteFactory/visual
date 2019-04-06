package super

import (
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/text"
)

// -------------------------------------------------------------------------
// Anchors

// AnchorY - Top, Middle, Bottom
type AnchorY int

// enum AnchorY
const (
	Top AnchorY = 1 + iota
	Middle
	Bottom
)

// AnchorX - Left, Center, Right
type AnchorX int

// enum AnchorX
const (
	Left AnchorX = 1 + iota
	Center
	Right
)

// AnchorTxt positions a text.Text label with an anchor alignment.
func AnchorTxt(txt *text.Text, pos pixel.Vec, anchorX AnchorX, anchorY AnchorY, desc string) {
	txt.Orig = pos
	txt.Dot = pos
	switch anchorX {
	case Left:
		txt.Dot.X -= 0
	case Center:
		txt.Dot.X -= (txt.BoundsOf(desc).W() / 2)
	case Right:
		txt.Dot.X -= txt.BoundsOf(desc).W()
	}
	switch anchorY {
	case Top:
		txt.Dot.Y -= txt.BoundsOf(desc).H()
	case Middle:
		txt.Dot.Y -= (txt.BoundsOf(desc).H() / 2)
	case Bottom:
		txt.Dot.Y -= 0
	}
	txt.Dot.X += 0
	txt.Dot.Y += 0
}

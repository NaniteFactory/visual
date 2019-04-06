package visual

import (
	"github.com/faiface/pixel/text"
	"github.com/golang/freetype/truetype"
	"github.com/nanitefactory/bindat/bindatkuji"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// AtlasASCII36 returns an atlas of font size 36 that's to draw only ASCII characters.
func AtlasASCII36() *text.Atlas {
	return atlas.AtlasASCII36
}

// AtlasASCII18 returns an atlas of font size 18 that's to draw only ASCII characters.
func AtlasASCII18() *text.Atlas {
	return atlas.AtlasASCII18
}

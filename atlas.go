package visual

import (
	"github.com/faiface/pixel/text"
	"github.com/nanitefactory/visual/atlas"
)

// AtlasASCII36 returns an atlas of font size 36 that's to draw only ASCII characters.
func AtlasASCII36() *text.Atlas {
	return atlas.AtlasASCII36
}

// AtlasASCII18 returns an atlas of font size 18 that's to draw only ASCII characters.
func AtlasASCII18() *text.Atlas {
	return atlas.AtlasASCII18
}

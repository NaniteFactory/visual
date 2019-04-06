package atlas

import (
	"github.com/faiface/pixel/text"
	"github.com/golang/freetype/truetype"
	"github.com/nanitefactory/bindat/bindatkuji"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

func newAtlasASCII(size float64) *text.Atlas {
	newTrueTypeFontFaceFromBin := func(bytes []byte, size float64) (font.Face, error) {
		font, err := truetype.Parse(bytes)
		if err != nil {
			return nil, err
		}
		return truetype.NewFace(font, &truetype.Options{
			Size:              size,
			GlyphCacheEntries: 1,
		}), nil
	}
	return text.NewAtlas(func() font.Face {
		if binTTF, err := bindatkuji.Asset("NanumBarunGothic.ttf"); err == nil {
			if retFace, err := newTrueTypeFontFaceFromBin(binTTF, size); err == nil {
				return retFace
			}
		}
		return basicfont.Face7x13
	}(), text.ASCII, nil)
}

// AtlasASCII36 is an atlas of font size 36 that's to draw only ASCII characters.
var AtlasASCII36 = newAtlasASCII(36)

// AtlasASCII18 is an atlas of font size 18 that's to draw only ASCII characters.
var AtlasASCII18 = newAtlasASCII(18)

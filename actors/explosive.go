package actors

import (
	"image/color"

	"github.com/faiface/pixel"
	kuji "github.com/nanitefactory/amidakuji/glossary"
)

// Explosions implements Actor interface.
type Explosions struct {
	*kuji.Explosions
}

// NewExplosions is a constructor.
// The 3rd argument colors can be nil. Then it will use its default value of a color set.
func NewExplosions(width, height float64, colors []color.Color, precision int) *Explosions {
	parent := kuji.NewExplosions(width, height, colors, precision)
	return &Explosions{parent}
}

// Draw implements Drawer interface.
func (e *Explosions) Draw(t pixel.Target) {
	if e.IsExploding() {
		e.Explosions.Draw(t)
	}
}

// Update implements Updater interface.
func (e *Explosions) Update(dt float64) {
	// Only update when there is at least one (animating) explosion.
	if e.IsExploding() {
		e.Explosions.Update(dt)
	}
}

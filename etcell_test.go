// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package tcell_ebiten

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/gdamore/tcell/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type mockFace struct {
	CellSize image.Point
	bounds   image.Rectangle
	metrics  font.Metrics
}

var _ font.Face = (*mockFace)(nil)

func (mf *mockFace) Close() error {
	return nil
}

func (mf *mockFace) update() {
	if mf.bounds.Empty() {
		x := mf.CellSize.X
		y := mf.CellSize.Y
		min_x := -x / 2
		min_y := -y / 2
		max_x := x + min_x
		max_y := y + min_y

		mf.bounds = image.Rect(min_x, min_y, max_x, max_y)

		mf.metrics = font.Metrics{
			Height:     fixed.I(mf.CellSize.Y),
			Ascent:     fixed.I(-min_y),
			Descent:    fixed.I(max_y),
			XHeight:    fixed.I(-min_x),
			CapHeight:  fixed.I(-min_x),
			CaretSlope: image.Point{X: 0, Y: 1},
		}
	}
}

func (mf *mockFace) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	mf.update()

	_, advance, ok = mf.GlyphBounds(r)
	if ok {
		dr = mf.bounds
		mask = &image.Uniform{C: color.Opaque}
		maskp = mf.bounds.Min
	}

	return
}

func (mf *mockFace) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	mf.update()

	advance, ok = mf.GlyphAdvance(r)
	if ok {
		bounds = fixed.R(mf.bounds.Min.X, mf.bounds.Min.Y, mf.bounds.Max.X, mf.bounds.Max.Y)
	}

	return
}

func (mf *mockFace) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	mf.update()

	ok = r != rune(0)
	if ok {
		advance = fixed.I(mf.CellSize.X)
	}

	return
}

func (mf *mockFace) Kern(r0, r1 rune) fixed.Int26_6 {
	return fixed.I(0)
}

func (mf *mockFace) Metrics() font.Metrics {
	mf.update()

	return mf.metrics
}

////// NewScreen() unit tests

func TestNewScreen(t *testing.T) {
	assert := assert.New(t)

	cell_size := image.Point{X: 11, Y: 22}

	face := text.NewGoXFace(&mockFace{CellSize: cell_size})

	gs := NewGameScreen(face)

	assert.NotNil(gs)

	screen, ok := gs.(tcell.Screen)
	assert.True(ok)

	screen.Init()
	defer screen.Fini()

	game, ok := gs.(ebiten.Game)
	assert.True(ok)

	gs.RegisterRuneFallback(rune(0), " ")
	gs.SetContent(0, 0, rune(0), nil, tcell.StyleDefault)
	assert.False(gs.CanDisplay(rune(0), false))
	assert.True(gs.CanDisplay(rune(0), true))

	swidth, sheight := game.Layout(cell_size.X*20+1, cell_size.Y*30+2)
	assert.Equal(cell_size.X*20, swidth)
	assert.Equal(cell_size.Y*30, sheight)
}

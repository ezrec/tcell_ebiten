// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package tcell_ebiten

import (
	"testing"

	"github.com/ezrec/tcell_ebiten/font"

	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"

	"github.com/stretchr/testify/assert"

	"github.com/gdamore/tcell/v2"

	ebiten_text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

////// ETCell() unit tests

func TestETCell(t *testing.T) {
	assert := assert.New(t)

	face_normal, _ := font.NewMonoFontFromTTF(gomono.TTF, 11)
	face_italic, _ := font.NewMonoFontFromTTF(gomonoitalic.TTF, 11)
	face_bold, _ := font.NewMonoFontFromTTF(gomonobold.TTF, 11)
	face_bolditalic, _ := font.NewMonoFontFromTTF(gomonobolditalic.TTF, 11)
	face := &font.FaceWithStyle{
		StyleMap: map[font.FontStyle]font.Face{
			font.FontStyleNormal:     face_normal,
			font.FontStyleItalic:     face_italic,
			font.FontStyleBold:       face_bold,
			font.FontStyleBoldItalic: face_bolditalic,
		},
	}

	cell_size_X, cell_size_Y := face.Size()

	gs := &ETCell{}
	gs.SetFont(face)

	assert.NotNil(gs)

	screen := gs.Screen()

	screen.Init()
	defer screen.Fini()

	game := gs.NewGame()

	const bad_rune = rune(0x1234567)

	screen.RegisterRuneFallback(bad_rune, " ")
	screen.SetContent(0, 0, bad_rune, nil, tcell.StyleDefault)
	assert.False(screen.CanDisplay(bad_rune, false))
	assert.True(screen.CanDisplay(bad_rune, true))

	swidth, sheight := game.Layout(cell_size_X*20+1, cell_size_Y*30+2)
	assert.Equal(cell_size_X*20, swidth)
	assert.Equal(cell_size_Y*30, sheight)
}

func TestETCellResizeAny(t *testing.T) {
	assert := assert.New(t)

	face := &font.CacheFont{
		FontMetrics: ebiten_text.Metrics{HAscent: 2.5, HDescent: 0.5},
		Width:       2,
		Height:      3,
	}

	et := &ETCell{}
	et.SetFont(face)

	game := et.NewGame()
	screen := et.Screen()

	table := [](struct {
		lx, ly int
		gx, gy int
		sx, sy int
	}){
		{lx: 2, ly: 3, gx: 2, gy: 3, sx: 1, sy: 1},
		{lx: 200, ly: 600, gx: 200, gy: 600, sx: 100, sy: 200},
	}

	for _, entry := range table {
		gx, gy := game.Layout(entry.lx, entry.ly)
		assert.Equal(entry.gx, gx)
		assert.Equal(entry.gy, gy)
		sx, sy := screen.Size()
		assert.Equal(entry.sx, sx)
		assert.Equal(entry.sy, sy)
	}
}

// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package etcell

import (
	"testing"

	"github.com/ezrec/etcell/font"

	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"

	"github.com/stretchr/testify/assert"

	"github.com/gdamore/tcell/v2"
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

	game := gs.Game()

	const bad_rune = rune(0x1234567)

	screen.RegisterRuneFallback(bad_rune, " ")
	screen.SetContent(0, 0, bad_rune, nil, tcell.StyleDefault)
	assert.False(screen.CanDisplay(bad_rune, false))
	assert.True(screen.CanDisplay(bad_rune, true))

	swidth, sheight := game.Layout(cell_size_X*20+1, cell_size_Y*30+2)
	assert.Equal(cell_size_X*20, swidth)
	assert.Equal(cell_size_Y*30, sheight)
}

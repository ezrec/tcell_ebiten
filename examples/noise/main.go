package main

import (
	"log"
	"math/rand"

	etcell "github.com/ezrec/tcell_ebiten/v2"
	"github.com/ezrec/tcell_ebiten/v2/font"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
)

type Noise struct {
	etcell.ETCell

	updating bool
}

func (n *Noise) Run() (err error) {
	err = n.ETCell.Run(n.runner)
	return
}

func (n *Noise) runner(screen tcell.Screen) (err error) {
	screen.Init()
	defer screen.Fini()

	var cursor_x, cursor_y int

	n.updating = true

	style := tcell.StyleDefault.Background(color.White).Foreground(color.Black)

	for event := range screen.EventQ() {
		switch ev := event.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEnd:
				return nil
			case tcell.KeyRune:
				// 'any' key
				n.updating = !n.updating
			}
		case *tcell.EventMouse:
			cursor_x, cursor_y = ev.Position()
			screen.ShowCursor(cursor_x, cursor_y)
		case *tcell.EventResize:
			screen.Sync()
		case *tcell.EventInterrupt:
			screen.Sync()
		}

		if !n.updating {
			continue
		}

		width, height := screen.Size()
		for x := range width {
			for y := range height {
				randrune := rune(rand.Int() % (0x7f - 32))
				randattr := rand.Int() & 0x3f
				randfg := tcell.NewRGBColor(
					int32(rand.Int()&0xff),
					int32(rand.Int()&0xff),
					int32(rand.Int()&0xff),
				)
				randbg := tcell.NewRGBColor(
					int32(rand.Int()&0xff),
					int32(rand.Int()&0xff),
					int32(rand.Int()&0xff),
				)
				style = style.Blink(((randattr >> 0) & 1) != 0)
				style = style.Bold(((randattr >> 1) & 1) != 0)
				style = style.Dim(((randattr >> 2) & 1) != 0)
				style = style.Italic(((randattr >> 3) & 1) != 0)
				style = style.Reverse(((randattr >> 4) & 1) != 0)
				style = style.StrikeThrough(((randattr >> 5) & 1) != 0)
				style = style.Foreground(randfg)
				style = style.Background(randbg)
				screen.SetContent(x, y, randrune, nil, style)
			}
		}
		screen.Show()
	}

	return
}

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("etcell noise")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	noise := &Noise{}

	var err error
	font_face := &font.FaceWithStyle{StyleMap: map[font.FontStyle]font.Face{}}
	font_face.StyleMap[font.FontStyleNormal], err = font.NewMonoFontFromTTF(gomono.TTF, 16)
	if err != nil {
		panic(err)
	}
	font_face.StyleMap[font.FontStyleItalic], err = font.NewMonoFontFromTTF(gomonoitalic.TTF, 16)
	if err != nil {
		panic(err)
	}
	font_face.StyleMap[font.FontStyleBold], err = font.NewMonoFontFromTTF(gomonobold.TTF, 16)
	if err != nil {
		panic(err)
	}
	font_face.StyleMap[font.FontStyleBoldItalic], err = font.NewMonoFontFromTTF(gomonobolditalic.TTF, 16)
	if err != nil {
		panic(err)
	}

	noise.SetFont(font_face)

	err = noise.Run()
	if err != nil {
		log.Fatal(err)
	}
}

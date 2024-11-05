// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"log"

	etcell "github.com/ezrec/tcell_ebiten"
	"github.com/ezrec/tcell_ebiten/font"

	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/gofont/gomono"
)

type DemoGame struct {
	text_run    interface{ Run() error }
	game_screen *etcell.ETCell
	text_game   interface {
		ebiten.Game
		LayoutF(x, y float64) (sx, sy float64)
	}
	draw_game interface {
		ebiten.Game
		LayoutF(x, y float64) (sx, sy float64)
	}
}

func NewDemoGame() (dg *DemoGame) {
	font_face, err := font.NewMonoFontFromTTF(gomono.TTF, 16)
	if err != nil {
		panic(err)
	}

	gs := &etcell.ETCell{}
	gs.SetFont(font_face)

	screen := gs.Screen()
	screen.RegisterRuneFallback('╭', "┌")
	screen.RegisterRuneFallback('╯', "┘")
	screen.RegisterRuneFallback('╮', "┐")
	screen.RegisterRuneFallback('╰', "└")

	dg = &DemoGame{
		text_run:    NewTextGame(screen),
		game_screen: gs,
		text_game:   gs.NewGame(),
		draw_game:   &DrawGame{},
	}

	return
}

func (dg *DemoGame) Draw(screen *ebiten.Image) {
	// Draw game first (background)
	dg.draw_game.Draw(screen)
	// Overlay with text (foreground)
	dg.text_game.Draw(screen)
}

func (dg *DemoGame) Update() (err error) {
	err = dg.text_game.Update()
	if err != nil {
		return
	}

	err = dg.draw_game.Update()
	if err != nil {
		return
	}

	return
}

func (dg *DemoGame) LayoutF(x, y float64) (float64, float64) {
	ox, oy := dg.text_game.LayoutF(x, y)

	dg.draw_game.LayoutF(ox, oy)

	return ox, oy
}

func main() {

	dg := NewDemoGame()

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("etcell demo")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	err := dg.game_screen.Run(func(screen tcell.Screen) error {
		screen.Init()
		defer screen.Fini()
		return dg.text_run.Run()
	})

	if err != nil {
		log.Fatal(err)
	}
}

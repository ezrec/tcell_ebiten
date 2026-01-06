// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"log"

	etcell "github.com/ezrec/tcell_ebiten/v2"
	"github.com/ezrec/tcell_ebiten/v2/font"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/gofont/gomono"
)

type DemoGame struct {
	text_run    interface{ Run() error }
	game_screen *etcell.ETCell
	text_game   *etcell.ETCellGame
	draw_game   interface {
		ebiten.Game
		LayoutF(x, y float64) (sx, sy float64)
	}
	y             float64
	monitor_scale float64
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
	err = dg.draw_game.Update()
	if err != nil {
		return
	}

	var geom ebiten.GeoM
	geom.Translate(0, dg.y/2)
	if dg.monitor_scale > 0.0 {
		geom.Scale(dg.monitor_scale, dg.monitor_scale)
	}
	dg.text_game.GeoM = geom
	err = dg.text_game.Update()
	if err != nil {
		return
	}

	return
}

func (dg *DemoGame) Layout(x, y int) (ox, oy int) {
	ox = x
	oy = y
	dg.monitor_scale = 0.0
	dg.y = float64(y)
	return ox, oy
}

func (dg *DemoGame) LayoutF(x, y float64) (ox, oy float64) {
	dg.monitor_scale = ebiten.Monitor().DeviceScaleFactor()
	dg.y = y

	dg.text_game.Layout(int(x), int(y)/2)

	ox = x * dg.monitor_scale
	oy = y * dg.monitor_scale
	dg.draw_game.LayoutF(ox, oy/2)

	return
}

func main() {

	dg := NewDemoGame()

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("etcell demo")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	go func() {
		screen := dg.text_game.Screen()
		screen.Init()
		defer screen.Fini()
		err := dg.text_run.Run()
		if err != nil {
			log.Fatal(err)
		}
	}()

	err := ebiten.RunGame(dg)
	if err != nil {
		log.Fatal(err)
	}
}

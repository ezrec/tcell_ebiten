// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"bytes"
	"context"
	"image"
	"log"

	"github.com/ezrec/tcell_ebiten"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gomono"
)

type DemoGame struct {
	game_screen tcell_ebiten.GameScreen
	text_game   *TextGame
	text_bounds image.Rectangle
	text_image  *ebiten.Image
	draw_game   *DrawGame
	draw_bounds image.Rectangle
	draw_image  *ebiten.Image
}

func NewDemoGame() (dg *DemoGame) {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	if err != nil {
		panic(err)
	}

	font_face := &text.GoTextFace{
		Source: source,
		Size:   16,
	}

	gs := tcell_ebiten.NewGameScreen(font_face)
	dg = &DemoGame{
		game_screen: gs,
		text_game:   NewTextGame(gs),
		draw_game:   &DrawGame{},
	}

	gs.SetHighDPI(true)

	gs.RegisterRuneFallback('╭', "┌")
	gs.RegisterRuneFallback('╯', "┘")
	gs.RegisterRuneFallback('╮', "┐")
	gs.RegisterRuneFallback('╰', "└")

	return
}

func (dg *DemoGame) Draw(screen *ebiten.Image) {
	table := []struct {
		game   ebiten.Game
		bounds image.Rectangle
		image  *ebiten.Image
	}{
		{dg.text_game, dg.text_bounds, dg.text_image},
		{dg.draw_game, dg.draw_bounds, dg.draw_image},
	}

	for _, entry := range table {
		entry.game.Draw(entry.image)
		ops := &ebiten.DrawImageOptions{}
		scale_x := float64(entry.bounds.Dx()) / float64(entry.image.Bounds().Dx())
		scale_y := float64(entry.bounds.Dy()) / float64(entry.image.Bounds().Dy())
		ops.GeoM.Scale(scale_x, scale_y)
		ops.GeoM.Translate(float64(entry.bounds.Min.X), float64(entry.bounds.Min.Y))
		screen.DrawImage(entry.image, ops)
	}
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

func (dg *DemoGame) Layout(x, y int) (int, int) {
	y_split := y / 2

	dg.draw_bounds = image.Rect(0, y_split, x, y)

	// Drawing layout
	draw_x, draw_y := dg.draw_game.Layout(dg.draw_bounds.Dx(), dg.draw_bounds.Dy())
	dg.draw_image = ebiten.NewImage(draw_x, draw_y)
	dg.draw_game.SetInputCapture(dg.draw_bounds)

	dg.text_bounds = image.Rect(0, 0, x, y_split)

	size_x, size_y := dg.text_bounds.Dx(), dg.text_bounds.Dy()
	scale := 2
	text_x, text_y := dg.text_game.LayoutF(float64(size_x)/float64(scale), float64(size_y)/float64(scale))
	dg.text_image = ebiten.NewImage(int(text_x), int(text_y))

	dg.text_game.SetInputCapture(dg.text_bounds, image.Rectangle{})

	return x, y
}

func main() {

	dg := NewDemoGame()

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("tcell_ebiten demo")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	dg.game_screen.Init()
	defer dg.game_screen.Fini()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		dg.text_game.Run(ctx)
		dg.game_screen.Close()
	}()

	err := ebiten.RunGame(dg)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
}

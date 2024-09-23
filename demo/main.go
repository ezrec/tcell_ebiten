// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"context"
	"image"
	"log"

	"github.com/ezrec/tcell_ebiten"

	"github.com/hajimehoshi/bitmapfont/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type DemoGame struct {
	text_game   *TextGame
	text_bounds image.Rectangle
	text_image  *ebiten.Image
	draw_game   *DrawGame
	draw_bounds image.Rectangle
	draw_image  *ebiten.Image
}

func NewDemoGame() (dg *DemoGame) {
	gs := tcell_ebiten.NewGameScreen(text.NewGoXFace(bitmapfont.Face))
	dg = &DemoGame{
		text_game: NewTextGame(gs),
		draw_game: &DrawGame{},
	}

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
	text_x, text_y := dg.text_game.Layout(size_x/scale, size_y/scale)
	dg.text_image = ebiten.NewImage(text_x, text_y)
	// Adjust offset
	offset_x := (size_x - text_x*scale) / 2
	offset_y := (size_y - text_y*scale) / 2
	dg.text_bounds.Add(image.Point{X: offset_x, Y: offset_y})
	dg.text_game.SetInputCapture(dg.text_bounds, image.Rectangle{})

	return x, y
}

func main() {

	dg := NewDemoGame()

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("tcell_ebiten demo")
	ebiten.SetWindowResizable(true)

	dg.game_screen.Init()
	defer dg.game_screen.Fini()

	ctx, cancel := context.WithCancel(context.Background())
	go dg.text_game.Run(ctx)

	err := ebiten.RunGame(dg)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
}

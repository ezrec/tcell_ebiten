// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"image"
	"image/color"

	"golang.org/x/image/draw"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type DrawGame struct {
	cursor   image.Point
	vertices []image.Point

	mouse_capture image.Rectangle
}

func (dg *DrawGame) Update() (err error) {
	x, y := ebiten.CursorPosition()
	cursor := image.Point{X: x, Y: y}

	if dg.mouse_capture.Empty() || cursor.In(dg.mouse_capture) {
		dg.cursor = cursor.Sub(dg.mouse_capture.Min)

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			dg.vertices = append(dg.vertices, dg.cursor)
		}
	}

	return
}

func (dg *DrawGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{192, 128, 192, 255})
	black := ebiten.NewImage(1, 1)
	black.Set(0, 0, color.Black)
	for n, v := range dg.vertices {
		// Draw 3x3 square of vertex
		box := image.Rect(v.X-3, v.Y-3, v.X+3, v.Y+3)
		draw.Draw(screen, box, black, image.Point{}, draw.Src)
		// If n>=1, draw line between box.
		if n > 0 {
			prior := dg.vertices[n-1]
			vector.StrokeLine(screen, float32(prior.X), float32(prior.Y),
				float32(v.X), float32(v.Y), 3.0, color.Black, true)
		}
	}

	// Draw the cursor.
	box := image.Rect(dg.cursor.X-3, dg.cursor.Y-3, dg.cursor.X+3, dg.cursor.Y+3)
	cursor := &image.Uniform{C: color.RGBA{R: 255, G: 128, B: 0, A: 255}}
	draw.Draw(screen, box, cursor, image.Point{}, draw.Src)
}

func (dg *DrawGame) Layout(x, y int) (int, int) {
	return x, y
}

func (dg *DrawGame) SetInputCapture(rect image.Rectangle) {
	dg.mouse_capture = rect
}

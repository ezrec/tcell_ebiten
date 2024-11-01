// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package tcell_ebiten

import (
	"image"

	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
)

// GameScreen combines the interface of tcell.Screen and a
// ebiten.Game handler.
//
// We are exceedingly lucky that tcell.Screen and ebiten.Game
// have no overlapping interfaces!
type GameScreen interface {
	tcell.Screen
	ebiten.Game

	// LayoutF supports High-DPI monitor situations.
	LayoutF(width, height float64) (screen_width, screen_height float64)

	// Close will cause ebiten.Run() to exit on the next Update()
	Close() error

	// SetHighDPI enables high-DPI mode (disable Ebiten automatic upscaling)
	SetHighDPI(enable bool)

	// SetCursorColor changes the color of the text 'hardware' cursor.
	SetCursorColor(color tcell.Color)

	// SetMouseCapture sets the ebiten image area where mouse events are captured.
	// If game is Empty, the entire ebiten game area's mouse events are captured.
	// If screen not Empty, game coordinates are mapped to screen cell coordinates.
	SetMouseCapture(game image.Rectangle, screen image.Rectangle)

	// SetKeyCapture sets the ebiten image area where key events are captured.
	// Default is all key events.
	SetKeyCapture(game image.Rectangle)

	// SetInputCapture set the ebiten image area where all input events are captured.
	// Default is all input events.
	// If game is Empty, the entire ebiten game area's events are captured.
	// If screen not Empty, game coordinates are mapped to screen cell coordinates.
	SetInputCapture(rect image.Rectangle, screen image.Rectangle)
}

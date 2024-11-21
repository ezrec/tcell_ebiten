// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

// Package etcell provides an [github.com/hajimehoshi/ebiten/v2] to [github.com/gdamore/tcell/v2]
// translation layer.
package etcell

import (
	"image"
	"image/color"
	"sync"

	"github.com/ezrec/etcell/font"
	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
)

type cell struct {
	Style     tcell.Style
	Rune      rune
	Combining []rune

	synced    bool
	glyph     *ebiten.Image
	combining [](*ebiten.Image)

	point   image.Point
	fgColor color.Color
	bgColor color.Color
}

// ETCell is the ebiten to tcell manager. An empty ETCell is valid, and ready to use. An ETCell should
// not be copied.
//
// When creating a new [ETCell], use the following patterns to manage how the ETCell display
// resolutions.
//
// 1. Game.Layout() resolution follows outside size; Screen.Size() is layout size divided by font size.
//   - et := ETCell{}
//
// 2. Game.Layout() fixed at 640x480 => Screen.Size() is layout size divided by font size.
//   - et := ETCell{}
//   - et.SetGameLayout(640, 480)
//
// 3. Screen.Size() fixed at 80x25 => Game.Layout() resolution is Screen.Size() times font size.
//   - et := ETCell{}
//   - et.SetScreenSize(80, 25)
//
// 4. Screen.Size() fixed at 80x25 => Game.Layout() resolution fixed at 640x480, drawing offset to 100,200
//   - et := ETCell{}
//   - et.SetScreenSize(80, 25).SetGameLayout(640x480).SetGameDrawOffset(100, 200)
type ETCell struct {
	grid_lock sync.Mutex

	// on_beep is called when the tcell.Screen.Beep() is invoked.
	on_beep func() error

	fixed_layout image.Point
	fixed_size   image.Point
	layout       image.Point

	face      font.Face   // Font face used for this screen.
	grid_size image.Point // Size of the grid, in cells.
	cell_size image.Point // Size of a single cell, in pixels.

	grid []cell // Grid of cells, not yet visible.

	cursor image.Point // Position of cursor, in grid cells

	style_default tcell.Style // Default text style

	cursor_color    tcell.Color       // Color of the cursor
	blink_cursor_ms int64             // Cursor blink _cycle_ duration in ms.
	cursor_style    tcell.CursorStyle // Cursor style

	blink_text_ms int64 // Text blink _cycle_ duration in ms.

	cell_image *ebiten.Image // All-white image of a single cell

	mouse_capture image.Rectangle
	mouse_mapping image.Rectangle
	key_capture   image.Rectangle
	focused       bool
	mouse_flags   tcell.MouseFlags
	enable_focus  bool
	enable_paste  bool

	italic_skew float64

	event_channel chan tcell.Event

	rune_fallback map[rune]string

	suspended   bool  // Input/output is suspended.
	close_error error // Closing error. ebiten.ErrTermination is used for clean shutdown.
}

// init initializes any default fields.
func (et *ETCell) init() {
	if et.face == nil {
		et.setFont(nil)
	}
	if et.blink_text_ms == 0 {
		et.blink_text_ms = 900
	}
	if et.blink_cursor_ms == 0 {
		et.blink_cursor_ms = 750
	}
	if et.rune_fallback == nil {
		et.rune_fallback = make(map[rune]string)
	}
	if et.italic_skew == 0 {
		et.italic_skew = -0.108
	}
	if et.mouse_flags == 0 {
		et.mouse_flags = tcell.MouseButtonEvents
	}
}

// SetScreenCursorColor sets the color of the text 'hardware' cursor.
func (et *ETCell) SetScreenCursorColor(color tcell.Color) *ETCell {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	et.cursor_color = color

	return et
}

// SetScreenSize forces a fixed Screen.Size() text grid layout.
func (et *ETCell) SetScreenSize(cols int, rows int) *ETCell {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	et.fixed_size = image.Point{X: cols, Y: rows}

	return et
}

// SetGameLayout forces a fixed Game.Layout() screen resolution.
func (et *ETCell) SetGameLayout(width int, height int) *ETCell {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	et.fixed_layout = image.Point{X: width, Y: height}

	return et
}

// SetGameDrawOffset sets an offset with which to draw the text grid.
func (et *ETCell) SetGameDrawOffset(x int, y int) *ETCell { return et }

// SetGameDrawScaling scales the text grid (after offsetting).
// Default is x = 1.0, y = 1.0
func (et *ETCell) SetGameDrawScaling(x, y float64) *ETCell { return et }

// SetFont sets the font for the text cells.
func (et *ETCell) SetFont(face font.Face) *ETCell {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	et.setFont(face)

	return et
}

func (et *ETCell) setFont(face font.Face) {
	// Make the layout grid based on the width and height (in pixels) given,
	// based on the font metrics. We use the rune 'O' to determine the nominal
	// bounding box for the character set.
	et.face = face

	width, height := et.face.Size()
	et.cell_size = image.Point{X: width, Y: height}
	et.cell_image = ebiten.NewImage(width, height)
	et.cell_image.Fill(color.White)
}

// SetGameUpdateMouseCapture sets the ebiten screen region to capture mouse events in.
// The default is an empty rectange, which captures all mouse events.
func (et *ETCell) SetGameUpdateMouseCapture(area image.Rectangle) *ETCell {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	et.mouse_capture = area

	return et
}

// SetGameUpdateKeyCapture sets the ebiten screen region to capture key events in.
// The default is an empty rectange, which captures all key events.
func (et *ETCell) SetGameUpdateKeyCapture(area image.Rectangle) *ETCell {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	et.key_capture = area

	return et
}

// SetGameUpdateInputCapture set the ebiten screen region to capture input events in.
func (et *ETCell) SetGameUpdateInputCapture(area image.Rectangle) *ETCell {
	et.SetGameUpdateMouseCapture(area)
	et.SetGameUpdateKeyCapture(area)

	return et
}

// postEvent helper
func (et *ETCell) postEvent(ev tcell.Event) (err error) {
	switch ev.(type) {
	case *tcell.EventFocus:
		if !et.enable_focus {
			return
		}
	case *tcell.EventPaste:
		if !et.enable_paste {
			return
		}
	case *tcell.EventMouse:
		if et.mouse_flags == tcell.MouseFlags(0) {
			return
		}
	default:
	}

	et.event_channel <- ev
	return
}

// Screen returns a struct compliant with the tcell.Screen interface
func (et *ETCell) Screen() *etcellScreen {
	return &etcellScreen{ETCell: et}
}

// Game returns the a struct compliant with ebiten.Game interface
func (et *ETCell) Game() *etcellGame {
	return &etcellGame{ETCell: et}
}

// Run a tcell application
func (et *ETCell) Run(runner func(screen tcell.Screen) error) error {
	go func() {
		err := runner(et.Screen())

		et.grid_lock.Lock()
		defer et.grid_lock.Unlock()

		if err == nil {
			err = ebiten.Termination
		}
		et.close_error = err
	}()

	return ebiten.RunGame(et.Game())
}

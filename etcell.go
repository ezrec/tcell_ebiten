// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

// Package etcell provides an [github.com/hajimehoshi/ebiten/v2] to [github.com/gdamore/tcell/v2]
// translation layer.
package tcell_ebiten

import (
	"image"
	"image/color"
	"sync"

	"github.com/ezrec/tcell_ebiten/font"
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
	fgColor color.RGBA
	bgColor color.RGBA
}

// ETCell is the ebiten to tcell manager. An empty ETCell is valid,
// and ready to use. An ETCell should not be copied.
type ETCell struct {
	grid_lock sync.Mutex

	// on_beep is called when the tcell.Screen.Beep() is invoked.
	on_beep func() error

	layout image.Rectangle

	face      font.Face   // Font face used for this screen.
	grid_size image.Point // Size of the grid, in cells.
	cell_size image.Point // Size of a single cell, in pixels.

	grid      []cell // Grid of cells, not yet visible.
	grid_draw []cell // Grid of cells, currently being drawn.

	cursor image.Point // Position of cursor, in grid cells

	style_default tcell.Style // Default text style

	cursor_color    tcell.Color       // Color of the cursor
	blink_cursor_ms int64             // Cursor blink _cycle_ duration in ms.
	cursor_style    tcell.CursorStyle // Cursor style

	blink_text_ms int64 // Text blink _cycle_ duration in ms.

	cell_image *ebiten.Image // All-white image of a single cell

	focused      bool
	mouse_flags  tcell.MouseFlags
	enable_focus bool
	enable_paste bool

	event_channel chan tcell.Event

	rune_fallback map[rune]string

	suspended   bool  // Input/output is suspended.
	close_error error // Closing error. ebiten.ErrTermination is used for clean shutdown.

	geom ebiten.GeoM
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

func (et *ETCell) setScreenSize(cols int, rows int) *ETCell {
	et.init()

	grid_size := image.Point{
		X: cols,
		Y: rows,
	}

	if grid_size.X <= 0 {
		grid_size.X = 1
	}

	if grid_size.Y <= 0 {
		grid_size.Y = 1
	}

	screenWidth := grid_size.X * et.cell_size.X
	screenHeight := grid_size.Y * et.cell_size.Y
	et.layout = image.Rect(0, 0, screenWidth, screenHeight)

	if !grid_size.Eq(et.grid_size) {
		et.grid_size = grid_size
		et.grid = make([]cell, et.grid_size.X*et.grid_size.Y)

		et.postEvent(tcell.NewEventResize(et.grid_size.X, et.grid_size.Y))
	}

	return et
}

// SetScreenSize resizes the text grid layout.
func (et *ETCell) SetScreenSize(cols int, rows int) *ETCell {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	return et.setScreenSize(cols, rows)
}

// GetGameSize() returns the size of the image to draw (without GeoM scaling)
func (et *ETCell) GetGameSize() (width, height int) {
	width = et.layout.Dx()
	height = et.layout.Dy()
	return
}

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

// SetGameGeoM sets the Ebiten.GeoM for display by Game.Draw() and mouse input mapping by Game.Update().
func (et *ETCell) SetGameGeoM(geom ebiten.GeoM) *ETCell {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	et.geom = geom

	return et
}

// postEvent helper
func (et *ETCell) postEvent(ev tcell.Event) (err error) {
	if et.event_channel == nil {
		return
	}

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
func (et *ETCell) Screen() *ETCellScreen {
	return &ETCellScreen{ETCell: et}
}

// Game returns the a struct compliant with ebiten.Game interface
func (et *ETCell) Game() *ETCellGame {
	return &ETCellGame{ETCell: et}
}

// Run a tcell application
func (et *ETCell) Run(runner func(screen tcell.Screen) error) error {
	go func() {
		err := runner(et.Screen())

		et.Exit(err)
	}()

	return ebiten.RunGame(et.Game())
}

// Exit the tcell application.
func (et *ETCell) Exit(err error) {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	if err == nil {
		err = ebiten.Termination
	}
	et.close_error = err
}

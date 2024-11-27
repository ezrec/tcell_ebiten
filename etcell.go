// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

// Package etcell provides an [github.com/hajimehoshi/ebiten/v2] to [github.com/gdamore/tcell/v2]
// translation layer.
package etcell

import (
	"image"
	"image/color"
	"sync"
	"time"

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
	fgColor color.RGBA
	bgColor color.RGBA
}

// ETCell is the ebiten to tcell manager. An empty ETCell is valid,
// and ready to use. An ETCell should not be copied.
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

	grid      []cell // Grid of cells, not yet visible.
	grid_draw []cell // Grid of cells, currently being drawn.

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
	et.layout = image.Point{
		X: screenWidth,
		Y: screenHeight,
	}

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
func (et *ETCell) Screen() *etcellScreen {
	return &etcellScreen{ETCell: et}
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

// DrawToImage draws to an Ebiten Image.
// Used to implement a custom override for ETCellGame.
// If Screen.Suspend() has been called, does nothing.
func (et *ETCell) DrawToImage(dst *ebiten.Image, geom ebiten.GeoM) {
	et.grid_lock.Lock()
	et.init()

	if et.suspended {
		et.grid_lock.Unlock()
		return
	}

	if cap(et.grid_draw) < len(et.grid) {
		et.grid_draw = make([]cell, len(et.grid))
	}
	et.grid_draw = et.grid_draw[0:len(et.grid)]
	copy(et.grid_draw, et.grid)
	et.grid_lock.Unlock()

	now := time.Now().UnixMilli()
	text_blink_ms := now % et.blink_text_ms
	text_blink_phase := text_blink_ms < (et.blink_text_ms / 2)

	for n := range et.grid_draw {
		cell := &et.grid_draw[n]

		if !cell.synced {
			continue
		}

		x := float64(cell.point.X * et.cell_size.X)
		y := float64(cell.point.Y * et.cell_size.Y)

		var bg_options ebiten.DrawImageOptions
		bg_options.ColorScale.ScaleWithColor(cell.bgColor)
		bg_options.GeoM.Translate(x, y)
		bg_options.GeoM.Concat(geom)

		dst.DrawImage(et.cell_image, &bg_options)

		var fg_options ebiten.DrawImageOptions
		fg_options.ColorScale.ScaleWithColor(cell.fgColor)
		fg_options.GeoM.Translate(x, y)
		fg_options.GeoM.Concat(geom)

		_, _, attr := cell.Style.Decompose()

		// If now blinking, don't draw the text. We _do_ draw underlines and strikethroughs.
		if (attr&tcell.AttrBlink) == 0 || !text_blink_phase {
			if cell.glyph != nil {
				dst.DrawImage(cell.glyph, &fg_options)
			}

			for _, glyph := range cell.combining {
				if glyph != nil {
					dst.DrawImage(glyph, &fg_options)
				}
			}
		}

		// Draw underline, if needed.
		// We define an underline as the top 1/16 of lower 1/8th of the cell.
		if (attr & tcell.AttrUnderline) != 0 {
			var opts ebiten.DrawImageOptions
			opts.ColorScale.ScaleWithColor(cell.fgColor)
			opts.GeoM.Scale(1.0, 1.0/16.0)
			opts.GeoM.Translate(x, y)
			opts.GeoM.Translate(0, float64(et.cell_size.Y)*(1.0-1.0/8.0))
			opts.GeoM.Concat(geom)
			dst.DrawImage(et.cell_image, &opts)
		}

		// Add strike-through
		// We define a strike-through as 1/16 of center of the character cell.
		if (attr & tcell.AttrStrikeThrough) != 0 {
			var opts ebiten.DrawImageOptions
			opts.ColorScale.ScaleWithColor(cell.fgColor)
			opts.GeoM.Scale(1.0, 1.0/16.0)
			opts.GeoM.Translate(x, y)
			opts.GeoM.Translate(0, float64(et.cell_size.Y)/2.0-1.0/32.0)
			opts.GeoM.Concat(geom)
			dst.DrawImage(et.cell_image, &opts)
		}
	}

	cursor_blink_ms := now % et.blink_cursor_ms
	cursor_blink_phase := cursor_blink_ms < (et.blink_cursor_ms / 2)

	// Draw cursor
	opts := ebiten.DrawImageOptions{}
	opts.ColorScale.ScaleWithColor(e_color_of(et.cursor_color))

	metrics := et.face.Metrics()

	switch et.cursor_style {
	case tcell.CursorStyleDefault:
		cursor_blink_phase = false
	case tcell.CursorStyleSteadyUnderline:
		cursor_blink_phase = false
		fallthrough
	case tcell.CursorStyleBlinkingUnderline:
		// Bar is 1/8 of text cell, below baseline.
		opts.GeoM.Scale(1.0, 1.0/8.0)
		opts.GeoM.Translate(0, metrics.HAscent+float64(et.cell_size.Y)*1.0/8.0)
	case tcell.CursorStyleSteadyBlock:
		cursor_blink_phase = false
		fallthrough
	case tcell.CursorStyleBlinkingBlock:
		// Block is entire text cell.
		// c_out = c_src x 1 - c_dst x 1
		// a_out = a_src x 1 + a_dst x 0
		opts.Blend = ebiten.Blend{
			BlendFactorSourceRGB:      ebiten.BlendFactorOne,
			BlendFactorDestinationRGB: ebiten.BlendFactorOne,
			BlendOperationRGB:         ebiten.BlendOperationSubtract,

			BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
			BlendFactorDestinationAlpha: ebiten.BlendFactorZero,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		}
	case tcell.CursorStyleSteadyBar:
		cursor_blink_phase = false
		fallthrough
	case tcell.CursorStyleBlinkingBar:
		// Bar is 1/4 of text cell, above baseline.
		opts.GeoM.Scale(1.0, 1.0/4.0)
		opts.GeoM.Translate(0, metrics.HAscent-float64(et.cell_size.Y)*1.0/4.0)
	}

	if !cursor_blink_phase {
		pos := image.Point{X: et.cursor.X * et.cell_size.X,
			Y: et.cursor.Y * et.cell_size.Y}
		opts.GeoM.Translate(float64(pos.X), float64(pos.Y))
		opts.GeoM.Concat(geom)
		dst.DrawImage(et.cell_image, &opts)
	}
}

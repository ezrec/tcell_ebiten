// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package tcell_ebiten

import (
	"image"
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type cell struct {
	Style     tcell.Style
	Rune      rune
	Combining []rune
	synced    bool
}

type etcell struct {
	// Overridable at any time.

	// OnBeep is called when the tcell.Screen.Beep() is invoked.
	OnBeep func() error

	// Computed, not mutable outside of this package.
	face      text.Face   // Font face used for this screen.
	grid_size image.Point // Size of the grid, in cells.
	cell_size image.Point // Size of a single cell, in pixels.

	grid        []cell        // Grid of cells, not yet visible.
	grid_image  *ebiten.Image // Image of the grid.
	blink_image *ebiten.Image // Blink cells.
	grid_lock   sync.Mutex

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

	suspended bool // Input/output is suspended.
}

// NewGameScreen creates a new GameScreen, suitable for both ebiten and tcell.
func NewGameScreen(font_face text.Face) GameScreen {
	black := tcell.FromImageColor(color.Black)
	white := tcell.FromImageColor(color.White)
	et := &etcell{
		face:            font_face,
		style_default:   tcell.StyleDefault.Background(black).Foreground(white),
		blink_text_ms:   900,
		blink_cursor_ms: 750,
		cursor_color:    tcell.ColorWhite,
		rune_fallback:   make(map[rune]string),
		italic_skew:     -0.108,
	}

	// Make the layout grid based on the width and height (in pixels) given,
	// based on the font metrics. We use the rune 'O' to determine the nominal
	// bounding box for the character set.
	const reference_rune = 'O'

	metrics := et.face.Metrics()
	width, height := text.Measure(string([]rune{reference_rune}), et.face, metrics.HLineGap)
	et.cell_size = image.Point{X: int(width), Y: int(height)}
	et.cell_image = ebiten.NewImage(int(width), int(height))
	et.cell_image.Fill(color.White)

	return et
}

// SetMouseCapture sets the ebiten screen region to capture mouse events in.
// The default is an empty rectange, which captures all mouse events.
func (et *etcell) SetMouseCapture(game, screen image.Rectangle) {
	et.mouse_capture = game
	et.mouse_mapping = screen
}

// SetKeyCapture sets the ebiten screen region to capture key events in.
// The default is an empty rectange, which captures all key events.
func (et *etcell) SetKeyCapture(game image.Rectangle) {
	et.key_capture = game
}

// SetInputCapture set the ebiten screen region to capture input events in.
// The default is an empty rectange, which captures all key and mouse events.
func (et *etcell) SetInputCapture(game, screen image.Rectangle) {
	et.SetMouseCapture(game, screen)
	et.SetKeyCapture(game)
}

// SetCursorColor sets the color of the 'hardware' cursor.
func (et *etcell) SetCursorColor(color tcell.Color) {
	et.cursor_color = color
}

// ////////// ebiten interfaces ////////////////////
var ebiten_button_map = map[ebiten.MouseButton]tcell.ButtonMask{
	ebiten.MouseButtonLeft:   tcell.ButtonPrimary,
	ebiten.MouseButtonMiddle: tcell.ButtonMiddle,
	ebiten.MouseButtonRight:  tcell.ButtonSecondary,
}

var ebiten_mod_map = map[ebiten.Key]tcell.ModMask{
	ebiten.KeyShift:   tcell.ModShift,
	ebiten.KeyControl: tcell.ModCtrl,
	ebiten.KeyAlt:     tcell.ModAlt,
	ebiten.KeyMeta:    tcell.ModMeta,
}

var ebiten_key_map = map[ebiten.Key]tcell.Key{
	ebiten.KeyArrowDown:  tcell.KeyDown,
	ebiten.KeyArrowLeft:  tcell.KeyLeft,
	ebiten.KeyArrowRight: tcell.KeyRight,
	ebiten.KeyArrowUp:    tcell.KeyUp,
	ebiten.KeyBackspace:  tcell.KeyBackspace,
	ebiten.KeyDelete:     tcell.KeyDelete,
	ebiten.KeyEnd:        tcell.KeyEnd,
	ebiten.KeyEnter:      tcell.KeyEnter,
	ebiten.KeyEscape:     tcell.KeyEscape,
	ebiten.KeyF1:         tcell.KeyF1,
	ebiten.KeyF2:         tcell.KeyF2,
	ebiten.KeyF3:         tcell.KeyF3,
	ebiten.KeyF4:         tcell.KeyF4,
	ebiten.KeyF5:         tcell.KeyF5,
	ebiten.KeyF6:         tcell.KeyF6,
	ebiten.KeyF7:         tcell.KeyF7,
	ebiten.KeyF8:         tcell.KeyF8,
	ebiten.KeyF9:         tcell.KeyF9,
	ebiten.KeyF10:        tcell.KeyF10,
	ebiten.KeyF11:        tcell.KeyF11,
	ebiten.KeyF12:        tcell.KeyF12,
	ebiten.KeyHome:       tcell.KeyHome,
	ebiten.KeyInsert:     tcell.KeyInsert,
	ebiten.KeyPageDown:   tcell.KeyPgDn,
	ebiten.KeyPageUp:     tcell.KeyPgUp,
	ebiten.KeyTab:        tcell.KeyTab,
}

var tcell_key_map = map[tcell.Key]ebiten.Key{}

func init() {
	for e_key, t_key := range ebiten_key_map {
		tcell_key_map[t_key] = e_key
	}
}

// modMask gets the tcell.ModMask for the current ebiten key modifiers.
func modMask() (mods tcell.ModMask) {
	for e_mod, t_mod := range ebiten_mod_map {
		if ebiten.IsKeyPressed(e_mod) {
			mods |= t_mod
		}
	}

	return mods
}

// isKeyJustPressedOrRepeating keys simulate repeated keys.
func isKeyJustPressedOrRepeating(key ebiten.Key) bool {
	tps := ebiten.ActualTPS()
	delay_ticks := int(0.500 /*sec*/ * tps)
	interval_ticks := int(0.050 /*sec*/ * tps)

	// If tps is 0 or very small, provide reasonable defaults
	if interval_ticks == 0 {
		delay_ticks = 30
		interval_ticks = 3
	}

	// Down for one tick? Then just pressed.
	d := inpututil.KeyPressDuration(key)
	if d == 1 {
		return true
	}

	// Wait until after the delay to start repeating.
	if d >= delay_ticks {
		if (d-delay_ticks)%interval_ticks == 0 {
			return true
		}
	}

	return false
}

// Update processes ebiten.Game events.
// If Screen.Suspend() has been called, does nothing.
func (et *etcell) Update() (err error) {
	if et.suspended {
		return
	}

	cursor_x, cursor_y := ebiten.CursorPosition()
	cursor := image.Point{X: cursor_x, Y: cursor_y}

	var in_focus bool
	var posted bool

	if et.mouse_capture.Empty() || cursor.In(et.mouse_capture) {
		mouse_mapping := et.mouse_mapping
		if mouse_mapping.Empty() {
			mouse_mapping = image.Rect(0, 0, et.grid_size.X, et.grid_size.Y)
		}

		mouse_capture := et.mouse_capture
		if mouse_capture.Empty() {
			mouse_capture = et.grid_image.Bounds()
		}

		mouse := cursor.Sub(et.mouse_capture.Min)
		if !et.focused {
			et.PostEvent(tcell.NewEventFocus(true))
			et.focused = true
			posted = true
		}
		var buttons tcell.ButtonMask
		for e_button, t_button := range ebiten_button_map {
			if ebiten.IsMouseButtonPressed(e_button) {
				buttons |= t_button
			}
		}

		mouse_x := mouse.X
		mouse_y := mouse.Y
		if !et.mouse_capture.Empty() {
			mouse_x = (mouse_x * mouse_mapping.Dx() / et.mouse_capture.Dx())
			mouse_y = (mouse_y * mouse_mapping.Dy() / et.mouse_capture.Dy())
		}
		mouse_x += et.mouse_mapping.Min.X
		mouse_y += et.mouse_mapping.Min.Y

		et.PostEvent(tcell.NewEventMouse(mouse_x, mouse_y, buttons, modMask()))

		in_focus = true
		posted = true
	}

	if et.key_capture.Empty() || cursor.In(et.key_capture) {
		if !et.focused {
			et.PostEvent(tcell.NewEventFocus(true))
			et.focused = true
		}
		mods := modMask()
		if (mods & tcell.ModCtrl) != 0 {
			for _, e_key := range inpututil.PressedKeys() {
				if !isKeyJustPressedOrRepeating(e_key) {
					continue
				}
				if e_key >= ebiten.KeyA && e_key <= ebiten.KeyZ {
					t_key := tcell.KeyCtrlA + tcell.Key(e_key-ebiten.KeyA)
					ev := tcell.NewEventKey(t_key, rune(0), mods & ^tcell.ModCtrl)
					et.PostEvent(ev)
					posted = true
				}
			}
		} else {
			key_runes := ebiten.AppendInputChars(nil)
			for _, key_rune := range key_runes {
				ev := tcell.NewEventKey(tcell.KeyRune, key_rune, mods & ^tcell.ModShift)
				et.PostEvent(ev)
				posted = true
			}
		}

		key_codes := inpututil.AppendPressedKeys(nil)
		for _, e_key := range key_codes {
			if !isKeyJustPressedOrRepeating(e_key) {
				continue
			}
			t_key, ok := ebiten_key_map[e_key]
			if ok {
				ev := tcell.NewEventKey(t_key, rune(0), mods)
				et.PostEvent(ev)
				posted = true
			}
		}

		in_focus = true
	}

	if !in_focus {
		if et.focused {
			et.PostEvent(tcell.NewEventFocus(false))
			et.focused = false
			posted = true
		}
	}

	// Always post a time event, if no other event was fired.
	if !posted {
		ev := &tcell.EventTime{}
		ev.SetEventNow()
		et.PostEvent(ev)
	}

	return
}

// Draw in ebiten.Game context.
// If Screen.Suspend() has been called, does nothing.
func (et *etcell) Draw(screen *ebiten.Image) {
	if et.suspended {
		return
	}

	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	if et.grid_image == nil {
		return
	}

	screen.DrawImage(et.grid_image, nil)

	now := time.Now().UnixMilli()
	text_blink_ms := now % et.blink_text_ms
	text_blink_phase := text_blink_ms < (et.blink_text_ms / 2)
	if text_blink_phase {
		screen.DrawImage(et.blink_image, nil)
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
		screen.DrawImage(et.cell_image, &opts)
	}
}

func (et *etcell) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {

	grid_size := image.Point{
		X: outsideWidth / et.cell_size.X,
		Y: outsideHeight / et.cell_size.Y,
	}

	screenWidth = grid_size.X * et.cell_size.X
	screenHeight = grid_size.Y * et.cell_size.Y

	if !grid_size.Eq(et.grid_size) {
		et.grid_lock.Lock()
		et.grid_size = grid_size
		et.grid = make([]cell, et.grid_size.X*et.grid_size.Y)
		et.grid_image = ebiten.NewImage(screenWidth, screenHeight)
		et.grid_image.Fill(color.Transparent)
		et.blink_image = ebiten.NewImage(screenWidth, screenHeight)
		et.blink_image.Fill(color.Transparent)
		et.grid_lock.Unlock()

		et.PostEvent(tcell.NewEventResize(et.grid_size.X, et.grid_size.Y))
	}

	return
}

//////////// tcell interfaces ////////////////////

// Init initializes the screen for use.
func (et *etcell) Init() (err error) {
	et.event_channel = make(chan tcell.Event, 128)

	et.Clear()

	return
}

// Fini finalizes the screen also releasing resources.
func (et *etcell) Fini() {
	close(et.event_channel)
}

// Clear logically erases the screen.
// This is effectively a short-cut for Fill(' ', StyleDefault).
func (et *etcell) Clear() {
	et.Fill(' ', tcell.StyleDefault)
}

// Fill fills the screen with the given character and style.
// The effect of filling the screen is not visible until Show
// is called (or Sync).
func (et *etcell) Fill(r rune, style tcell.Style) {
	for n := 0; n < len(et.grid); n++ {
		et.grid[n] = cell{
			Style: style,
			Rune:  r,
		}
	}
}

// SetCell is an older API, and will be removed.  Please use
// SetContent instead; SetCell is implemented in terms of SetContent.
func (et *etcell) SetCell(x int, y int, style tcell.Style, ch ...rune) {
	if len(ch) == 0 {
		ch = []rune{' '}
	}
	et.SetContent(x, y, ch[0], ch[1:], style)
}

// GetContent returns the contents at the given location.  If the
// coordinates are out of range, then the values will be 0, nil,
// StyleDefault.  Note that the contents returned are logical contents
// and may not actually be what is displayed, but rather are what will
// be displayed if Show() or Sync() is called.  The width is the width
// in screen cells; most often this will be 1, but some East Asian
// characters and emoji require two cells.
func (et *etcell) GetContent(x, y int) (primary rune, combining []rune, style tcell.Style, width int) {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	if x >= et.grid_size.X {
		return
	}
	if y >= et.grid_size.Y {
		return
	}

	n := y*et.grid_size.X + x
	cell := et.grid[n]

	primary = cell.Rune
	combining = cell.Combining
	style = cell.Style
	width = 1

	return
}

// SetContent sets the contents of the given cell location.  If
// the coordinates are out of range, then the operation is ignored.
//
// The first rune is the primary non-zero width rune.  The array
// that follows is a possible list of combining characters to append,
// and will usually be nil (no combining characters.)
//
// The results are not displayed until Show() or Sync() is called.
//
// Note that wide (East Asian full width and emoji) runes occupy two cells,
// and attempts to place character at next cell to the right will have
// undefined effects.  Wide runes that are printed in the
// last column will be replaced with a single width space on output.
func (et *etcell) SetContent(x int, y int, primary rune, combining []rune, style tcell.Style) {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	if x >= et.grid_size.X {
		return
	}
	if y >= et.grid_size.Y {
		return
	}

	n := y*et.grid_size.X + x

	et.grid[n] = cell{
		Rune:      primary,
		Combining: combining,
		Style:     style,
	}
}

// SetStyle sets the default style to use when clearing the screen
// or when StyleDefault is specified.  If it is also StyleDefault,
// then whatever system/terminal default is relevant will be used.
func (et *etcell) SetStyle(style tcell.Style) {
	et.style_default = style
}

// ShowCursor is used to display the cursor at a given location.
// If the coordinates -1, -1 are given or are otherwise outside the
// dimensions of the screen, the cursor will be hidden.
func (et *etcell) ShowCursor(x int, y int) {
	et.cursor = image.Point{X: x, Y: y}
}

// HideCursor is used to hide the cursor.  It's an alias for
// ShowCursor(-1, -1).sim
func (et *etcell) HideCursor() {
	et.ShowCursor(-1, -1)
}

// SetCursorStyle is used to set the cursor style.  If the style
// is not supported (or cursor styles are not supported at all),
// then this will have no effect.
func (et *etcell) SetCursorStyle(cs tcell.CursorStyle) {
	et.cursor_style = cs
}

// Size returns the screen size as width, height.  This changes in
// response to a call to Clear or Flush.
func (et *etcell) Size() (width, height int) {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	width = et.grid_size.X
	height = et.grid_size.Y

	return
}

// ChannelEvents is an infinite loop that waits for an event and
// channels it into the user provided channel ch.  Closing the
// quit channel and calling the Fini method are cancellation
// signals.  When a cancellation signal is received the method
// returns after closing ch.
//
// This method should be used as a goroutine.
//
// NOTE: PollEvent should not be called while this method is running.
func (et *etcell) ChannelEvents(ch chan<- tcell.Event, quit <-chan struct{}) {
	go func() {
		for {
			select {
			case ev := <-et.event_channel:
				ch <- ev
			case _ = <-quit:
				close(ch)
				return
			}
		}
	}()
}

// PollEvent waits for events to arrive.  Main application loops
// must spin on this to prevent the application from stalling.
// Furthermore, this will return nil if the Screen is finalized.
func (et *etcell) PollEvent() (ev tcell.Event) {
	ev = <-et.event_channel
	return ev
}

// HasPendingEvent returns true if PollEvent would return an event
// without blocking.  If the screen is stopped and PollEvent would
// return nil, then the return value from this function is unspecified.
// The purpose of this function is to allow multiple events to be collected
// at once, to minimize screen redraws.
func (et *etcell) HasPendingEvent() (has bool) {
	return len(et.event_channel) != 0
}

// PostEvent tries to post an event into the event stream.  This
// can fail if the event queue is full.  In that case, the event
// is dropped, and ErrEventQFull is returned.
func (et *etcell) PostEvent(ev tcell.Event) (err error) {
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

// Deprecated: PostEventWait is unsafe, and will be removed
// in the future.
//
// PostEventWait is like PostEvent, but if the queue is full, it
// blocks until there is space in the queue, making delivery
// reliable.  However, it is VERY important that this function
// never be called from within whatever event loop is polling
// with PollEvent(), otherwise a deadlock may arise.
//
// For this reason, when using this function, the use of a
// Goroutine is recommended to ensure no deadlock can occur.
func (et *etcell) PostEventWait(ev tcell.Event) {
	et.PostEvent(ev)
}

// EnableMouse enables the mouse.  (If your terminal supports it.)
// If no flags are specified, then all events are reported, if the
// terminal supports them.
func (et *etcell) EnableMouse(flags ...tcell.MouseFlags) {
	for _, flag := range flags {
		et.mouse_flags |= flag
	}
}

// DisableMouse disables the mouse.
func (et *etcell) DisableMouse() {
	et.mouse_flags = 0
}

// EnablePaste enables bracketed paste mode, if supported.
func (et *etcell) EnablePaste() {
	et.enable_paste = true
}

// DisablePaste disables bracketed paste mode.
func (et *etcell) DisablePaste() {
	et.enable_paste = false
}

// EnableFocus enables reporting of focus events, if your terminal supports it.
func (et *etcell) EnableFocus() {
	et.enable_focus = true
}

// DisableFocus disables reporting of focus events.
func (et *etcell) DisableFocus() {
	et.enable_focus = false
}

// HasMouse returns true if the terminal (apparently) supports a
// mouse.  Note that the return value of true doesn't guarantee that
// a mouse/pointing device is present; a false return definitely
// indicates no mouse support is available.
func (et *etcell) HasMouse() (has bool) {
	has = true
	return
}

// Colors returns the number of colors.  All colors are assumed to
// use the ANSI color map.  If a terminal is monochrome, it will
// return 0.
func (et *etcell) Colors() (ncolors int) {
	ncolors = 16
	return
}

func e_color_of(c tcell.Color) color.Color {
	r, g, b := c.TrueColor().RGB()
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// Show makes all the content changes made using SetContent() visible
// on the display.
//
// It does so in the most efficient and least visually disruptive
// manner possible.
func (et *etcell) Show() {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	dst := et.grid_image

	pt := image.Point{}
	n := 0
	pt.Y = 0
	for y := 0; y < et.grid_size.Y; y++ {
		pt.Y = y * et.cell_size.Y
		pt.X = 0
		for x := 0; x < et.grid_size.X; x++ {
			pt.X = x * et.cell_size.X
			cell := &et.grid[n]
			n++

			style := cell.Style
			if style == tcell.StyleDefault {
				style = et.style_default
			}
			fg, bg, attr := style.Decompose()

			if cell.synced {
				continue
			}
			cell.synced = true

			if (attr & tcell.AttrInvalid) != 0 {
				// Ignore all attributes.
				attr = tcell.AttrNone
			}

			if fg == tcell.ColorDefault {
				fg = tcell.ColorWhite
			}

			if bg == tcell.ColorDefault {
				bg = tcell.ColorBlack
			}

			// Reverse fg & bg if asked to.
			if (attr & tcell.AttrReverse) != 0 {
				fg, bg = bg, fg
			}

			// For Bold, intensify the color.
			if (attr & tcell.AttrBold) != 0 {
				r, g, b := fg.TrueColor().RGB()
				fg = tcell.NewRGBColor(
					min(255, int32(float32(r)*2)),
					min(255, int32(float32(g)*2)),
					min(255, int32(float32(b)*2)),
				)
			}

			// For Dim, de-intensify the color.
			if (attr & tcell.AttrDim) != 0 {
				r, g, b := fg.TrueColor().RGB()
				fg = tcell.NewRGBColor(
					min(255, int32(float32(r)/2)),
					min(255, int32(float32(g)/2)),
					min(255, int32(float32(b)/2)),
				)
			}

			fg_color := e_color_of(fg)
			bg_color := e_color_of(bg)

			// Draw background first
			opts := ebiten.DrawImageOptions{}
			opts.ColorScale.ScaleWithColor(bg_color)
			opts.GeoM.Translate(float64(pt.X), float64(pt.Y))
			dst.DrawImage(et.cell_image, &opts)

			if (attr & tcell.AttrBlink) != 0 {
				// Add to the blink image.
				et.blink_image.DrawImage(et.cell_image, &opts)
			} else {
				// Clear from the blink image
				opts.ColorScale.ScaleWithColor(color.Transparent)
				et.blink_image.DrawImage(et.cell_image, &opts)
			}

			// Is this a rune that can be displayed?
			str := string([]rune{cell.Rune})
			if text.Advance(str, et.face) == 0.0 {
				var ok bool
				str, ok = et.rune_fallback[cell.Rune]
				if !ok {
					str = " "
				}
			}

			// Draw the first rune
			text_opts := text.DrawOptions{}
			text_opts.ColorScale.ScaleWithColor(fg_color)
			if (attr & tcell.AttrItalic) != 0 {
				text_opts.GeoM.Skew(et.italic_skew, 0.0)
				text_opts.GeoM.Translate(-float64(et.cell_size.X)*math.Sin(et.italic_skew), 0)
			}
			text_opts.GeoM.Translate(float64(pt.X), float64(pt.Y))

			text.Draw(dst, str, et.face, &text_opts)

			// Draw the combining runes
			if len(cell.Combining) > 0 {
				text.Draw(dst, string(cell.Combining), et.face, &text_opts)
			}

			// Draw underline, if needed.
			// We define an underline as the top 1/16 of lower 1/8th of the cell.
			if (attr & tcell.AttrUnderline) != 0 {
				opts := ebiten.DrawImageOptions{}
				opts.ColorScale.ScaleWithColor(fg_color)
				opts.GeoM.Scale(1.0, 1.0/16.0)
				opts.GeoM.Translate(0, float64(et.cell_size.Y)*(1.0-1.0/8.0))
				opts.GeoM.Translate(float64(pt.X), float64(pt.Y))
				dst.DrawImage(et.cell_image, &opts)
			}

			// Add strike-through
			// We define a strike-through as 1/16 of center of the character cell.
			if (attr & tcell.AttrStrikeThrough) != 0 {
				opts := ebiten.DrawImageOptions{}
				opts.ColorScale.ScaleWithColor(fg_color)
				opts.GeoM.Scale(1.0, 1.0/16.0)
				opts.GeoM.Translate(0, float64(et.cell_size.Y)/2.0-1.0/32.0)
				opts.GeoM.Translate(float64(pt.X), float64(pt.Y))
				dst.DrawImage(et.cell_image, &opts)
			}
		}
	}
}

// Sync works like Show(), but it updates every visible cell on the
// physical display, assuming that it is not synchronized with any
// internal model.  This may be both expensive and visually jarring,
// so it should only be used when believed to actually be necessary.
//
// Typically, this is called as a result of a user-requested redraw
// (e.g. to clear up on-screen corruption caused by some other program),
// or during a resize event.
func (et *etcell) Sync() {
	et.grid_lock.Lock()
	for n := 0; n < len(et.grid); n++ {
		et.grid[n].synced = false
	}
	et.grid_lock.Unlock()

	et.Show()
}

// CharacterSet returns information about the character set.
// This isn't the full locale, but it does give us the input/output
// character set.  Note that this is just for diagnostic purposes,
// we normally translate input/output to/from UTF-8, regardless of
// what the user's environment is.
func (et *etcell) CharacterSet() (charset string) {
	charset = "UTF-8"
	return
}

// RegisterRuneFallback adds a fallback for runes that are not
// part of the character set -- for example one could register
// o as a fallback for ø.  This should be done cautiously for
// characters that might be displayed ordinarily in language
// specific text -- characters that could change the meaning of
// written text would be dangerous.  The intention here is to
// facilitate fallback characters in pseudo-graphical applications.
//
// If the terminal has fallbacks already in place via an alternate
// character set, those are used in preference.  Also, standard
// fallbacks for graphical characters in the alternate character set
// terminfo string are registered implicitly.
//
// The display string should be the same width as original rune.
// This makes it possible to register two character replacements
// for full width East Asian characters, for example.
//
// It is recommended that replacement strings consist only of
// 7-bit ASCII, since other characters may not display everywhere.
func (et *etcell) RegisterRuneFallback(r rune, subst string) {
	et.rune_fallback[r] = subst
}

// UnregisterRuneFallback unmaps a replacement.  It will unmap
// the implicit ASCII replacements for alternate characters as well.
// When an unmapped char needs to be displayed, but no suitable
// glyph is available, '?' is emitted instead.  It is not possible
// to "disable" the use of alternate characters that are supported
// by your terminal except by changing the terminal database.
func (et *etcell) UnregisterRuneFallback(r rune) {
	delete(et.rune_fallback, r)
}

// CanDisplay returns true if the given rune can be displayed on
// this screen.  Note that this is a best-guess effort -- whether
// your fonts support the character or not may be questionable.
// Mostly this is for folks who work outside of Unicode.
//
// If checkFallbacks is true, then if any (possibly imperfect)
// fallbacks are registered, this will return true.  This will
// also return true if the terminal can replace the glyph with
// one that is visually indistinguishable from the one requested.
func (et *etcell) CanDisplay(r rune, checkFallbacks bool) (can bool) {
	str := string([]rune{r})

	can = text.Advance(str, et.face) > 0.0

	if !can && checkFallbacks {
		_, can = et.rune_fallback[r]
	}

	return
}

// Resize does nothing, since it's generally not possible to
// ask a screen to resize, but it allows the Screen to implement
// the View interface.
func (et *etcell) Resize(int, int, int, int) {
	// Not implemented.
}

// HasKey returns true if the keyboard is believed to have the
// key.  In some cases a keyboard may have keys with this name
// but no support for them, while in others a key may be reported
// as supported but not actually be usable (such as some emulators
// that hijack certain keys).  Its best not to depend to strictly
// on this function, but it can be used for hinting when building
// menus, displayed hot-keys, etc.  Note that KeyRune (literal
// runes) is always true.
func (et *etcell) HasKey(key tcell.Key) (has bool) {

	switch {
	case key == tcell.KeyRune:
		has = true
	case key >= tcell.KeyCtrlA && key <= tcell.KeyCtrlZ:
		has = true
	default:
		_, has = tcell_key_map[key]
	}

	return
}

// Suspend pauses input and output processing.  It also restores the
// terminal settings to what they were when the application started.
// This can be used to, for example, run a sub-shell.
func (et *etcell) Suspend() (err error) {
	et.suspended = true
	return
}

// Resume resumes after Suspend().
func (et *etcell) Resume() (err error) {
	et.suspended = false
	return
}

// Beep attempts to sound an OS-dependent audible alert and returns an error
// when unsuccessful.
func (et *etcell) Beep() (err error) {
	if et.OnBeep != nil {
		err = et.OnBeep()
	}
	return
}

// SetSize attempts to resize the window.  It also invalidates the cells and
// calls the resize function.  Note that if the window size is changed, it will
// not be restored upon application exit.
//
// Many terminals cannot support this.  Perversely, the "modern" Windows Terminal
// does not support application-initiated resizing, whereas the legacy terminal does.
// Also, some emulators can support this but may have it disabled by default.
func (et *etcell) SetSize(int, int) {
	// Not implemented.
}

// LockRegion sets or unsets a lock on a region of cells. A lock on a
// cell prevents the cell from being redrawn.
func (et *etcell) LockRegion(x, y, width, height int, lock bool) {
	// Not implemented.
}

// Tty returns the underlying Tty. If the screen is not a terminal, the
// returned bool will be false
func (et *etcell) Tty() (tty tcell.Tty, is_tty bool) {
	// Not implemented
	tty = nil
	is_tty = false
	return
}

// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

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

type ETCellScreen struct {
	grid_lock sync.Mutex

	// on_beep is called when the tcell.Screen.Beep() is invoked.
	on_beep func() error

	layout image.Rectangle

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

	focused      bool
	mouse_flags  tcell.MouseFlags
	enable_focus bool
	enable_paste bool

	event_channel chan tcell.Event

	rune_fallback map[rune]string

	suspended   bool  // Input/output is suspended.
	close_error error // Closing error. ebiten.ErrTermination is used for clean shutdown.
}

// Validate interface compliance
var _ tcell.Screen = (*ETCellScreen)(nil)

// Init initializes the screen for use.
func (et *ETCellScreen) Init() (err error) {
	et.event_channel = make(chan tcell.Event, 128)

	et.Clear()

	return
}

// Fini finalizes the screen also releasing resources.
func (et *ETCellScreen) Fini() {
	close(et.event_channel)
	et.event_channel = nil
}

// Clear logically erases the screen.
// This is effectively a short-cut for Fill(' ', StyleDefault).
func (et *ETCellScreen) Clear() {
	et.Fill(' ', tcell.StyleDefault)
}

// Fill fills the screen with the given character and style.
// The effect of filling the screen is not visible until Show
// is called (or Sync).
func (et *ETCellScreen) Fill(r rune, style tcell.Style) {
	for n := 0; n < len(et.grid); n++ {
		et.grid[n] = cell{
			Style: style,
			Rune:  r,
		}
	}
}

// SetCell is an older API, and will be removed.  Please use
// SetContent instead; SetCell is implemented in terms of SetContent.
func (et *ETCellScreen) SetCell(x int, y int, style tcell.Style, ch ...rune) {
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
func (et *ETCellScreen) GetContent(x, y int) (primary rune, combining []rune, style tcell.Style, width int) {
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
func (et *ETCellScreen) SetContent(x int, y int, primary rune, combining []rune, style tcell.Style) {
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
func (et *ETCellScreen) SetStyle(style tcell.Style) {
	et.style_default = style
}

// ShowCursor is used to display the cursor at a given location.
// If the coordinates -1, -1 are given or are otherwise outside the
// dimensions of the screen, the cursor will be hidden.
func (et *ETCellScreen) ShowCursor(x int, y int) {
	et.cursor = image.Point{X: x, Y: y}
}

// HideCursor is used to hide the cursor.  It's an alias for
// ShowCursor(-1, -1).sim
func (et *ETCellScreen) HideCursor() {
	et.ShowCursor(-1, -1)
}

// SetCursorStyle is used to set the cursor style.  If the style
// is not supported (or cursor styles are not supported at all),
// then this will have no effect.
func (et *ETCellScreen) SetCursorStyle(cs tcell.CursorStyle, colors ...tcell.Color) {
	et.cursor_style = cs
}

// SetTitle sets the title bar of the window.
// Not implemented.
func (et *ETCellScreen) SetTitle(title string) {
	// not implemented
}

// Size returns the screen size as width, height.  This changes in
// response to a call to Clear or Flush.
func (et *ETCellScreen) Size() (width, height int) {
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
func (et *ETCellScreen) ChannelEvents(ch chan<- tcell.Event, quit <-chan struct{}) {
	go func() {
		for {
			select {
			case ev := <-et.event_channel:
				ch <- ev
			case <-quit:
				close(ch)
				return
			}
		}
	}()
}

// PollEvent waits for events to arrive.  Main application loops
// must spin on this to prevent the application from stalling.
// Furthermore, this will return nil if the Screen is finalized.
func (et *ETCellScreen) PollEvent() (ev tcell.Event) {
	ev = <-et.event_channel
	return ev
}

// HasPendingEvent returns true if PollEvent would return an event
// without blocking.  If the screen is stopped and PollEvent would
// return nil, then the return value from this function is unspecified.
// The purpose of this function is to allow multiple events to be collected
// at once, to minimize screen redraws.
func (et *ETCellScreen) HasPendingEvent() (has bool) {
	return len(et.event_channel) != 0
}

// PostEvent tries to post an event into the event stream.  This
// can fail if the event queue is full.  In that case, the event
// is dropped, and ErrEventQFull is returned.
func (et *ETCellScreen) PostEvent(ev tcell.Event) (err error) {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	err = et.postEvent(ev)

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
func (et *ETCellScreen) PostEventWait(ev tcell.Event) {
	et.PostEvent(ev)
}

// EnableMouse enables the mouse.  (If your terminal supports it.)
// If no flags are specified, then all events are reported, if the
// terminal supports them.
func (et *ETCellScreen) EnableMouse(flags ...tcell.MouseFlags) {
	for _, flag := range flags {
		et.mouse_flags |= flag
	}
}

// DisableMouse disables the mouse.
func (et *ETCellScreen) DisableMouse() {
	et.mouse_flags = 0
}

// EnablePaste enables bracketed paste mode, if supported.
func (et *ETCellScreen) EnablePaste() {
	et.enable_paste = true
}

// DisablePaste disables bracketed paste mode.
func (et *ETCellScreen) DisablePaste() {
	et.enable_paste = false
}

// EnableFocus enables reporting of focus events, if your terminal supports it.
func (et *ETCellScreen) EnableFocus() {
	et.enable_focus = true
}

// DisableFocus disables reporting of focus events.
func (et *ETCellScreen) DisableFocus() {
	et.enable_focus = false
}

// HasMouse returns true if the terminal (apparently) supports a
// mouse.  Note that the return value of true doesn't guarantee that
// a mouse/pointing device is present; a false return definitely
// indicates no mouse support is available.
func (et *ETCellScreen) HasMouse() (has bool) {
	has = true
	return
}

// Colors returns the number of colors.  All colors are assumed to
// use the ANSI color map.  If a terminal is monochrome, it will
// return 0.
func (et *ETCellScreen) Colors() (ncolors int) {
	ncolors = 16
	return
}

func e_color_of(c tcell.Color) color.RGBA {
	r, g, b := c.TrueColor().RGB()
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// Show makes all the content changes made using SetContent() visible
// on the display.
//
// It does so in the most efficient and least visually disruptive
// manner possible.
func (et *ETCellScreen) Show() {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	pt := image.Point{}
	n := 0
	pt.Y = 0
	for y := 0; y < et.grid_size.Y; y++ {
		pt.Y = y
		pt.X = 0
		for x := 0; x < et.grid_size.X; x++ {
			pt.X = x
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

			cell.point = pt
			cell.bgColor = e_color_of(bg)
			cell.fgColor = e_color_of(fg)

			// Is this a rune that can be displayed?
			runes := append([]rune{cell.Rune}, cell.Combining...)
			if !et.CanDisplay(runes[0], false) {
				str, ok := et.rune_fallback[cell.Rune]
				if !ok {
					runes[0] = ' '
				} else {
					runes = []rune(str)
				}
			}

			font_style := font.FontStyleNormal
			if (attr & (tcell.AttrItalic | tcell.AttrBold)) == (tcell.AttrItalic | tcell.AttrBold) {
				font_style = font.FontStyleBoldItalic
			} else if (attr & tcell.AttrItalic) != 0 {
				font_style = font.FontStyleItalic
			} else if (attr & tcell.AttrBold) != 0 {
				font_style = font.FontStyleBold
			}

			cell.glyph, _ = et.face.Glyph(runes[0], font_style)

			if len(runes) > 1 {
				// Draw the combining runes
				cell.combining = make([](*ebiten.Image), len(runes[1:]))
				for n, char := range runes[1:] {
					glyph, _ := et.face.Glyph(char, font_style)
					cell.combining[n] = glyph
				}
			} else {
				cell.combining = nil
			}

			cell.synced = true
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
func (et *ETCellScreen) Sync() {
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
func (et *ETCellScreen) CharacterSet() (charset string) {
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
func (et *ETCellScreen) RegisterRuneFallback(r rune, subst string) {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	if et.rune_fallback == nil {
		et.rune_fallback = make(map[rune]string, 16)
	}
	et.rune_fallback[r] = subst
}

// UnregisterRuneFallback unmaps a replacement.  It will unmap
// the implicit ASCII replacements for alternate characters as well.
// When an unmapped char needs to be displayed, but no suitable
// glyph is available, '?' is emitted instead.  It is not possible
// to "disable" the use of alternate characters that are supported
// by your terminal except by changing the terminal database.
func (et *ETCellScreen) UnregisterRuneFallback(r rune) {
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
func (et *ETCellScreen) CanDisplay(r rune, checkFallbacks bool) (can bool) {
	_, is_empty := et.face.Glyph(r, font.FontStyleNormal)

	can = !is_empty

	if !can && checkFallbacks {
		_, can = et.rune_fallback[r]
	}

	return
}

// Resize does nothing, since it's generally not possible to
// ask a screen to resize, but it allows the Screen to implement
// the View interface.
func (et *ETCellScreen) Resize(int, int, int, int) {
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
func (et *ETCellScreen) HasKey(key tcell.Key) (has bool) {

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
func (et *ETCellScreen) Suspend() (err error) {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	et.suspended = true
	return
}

// Resume resumes after Suspend().
func (et *ETCellScreen) Resume() (err error) {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	et.suspended = false
	return
}

// Beep attempts to sound an OS-dependent audible alert and returns an error
// when unsuccessful.
func (et *ETCellScreen) Beep() (err error) {
	if et.on_beep != nil {
		err = et.on_beep()
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
func (et *ETCellScreen) SetSize(int, int) {
	// Not implemented.
}

// LockRegion sets or unsets a lock on a region of cells. A lock on a
// cell prevents the cell from being redrawn.
func (et *ETCellScreen) LockRegion(x, y, width, height int, lock bool) {
	// Not implemented.
}

// Tty returns the underlying Tty. If the screen is not a terminal, the
// returned bool will be false
func (et *ETCellScreen) Tty() (tty tcell.Tty, is_tty bool) {
	// Not implemented
	tty = nil
	is_tty = false
	return
}

// GetClipboard triggers an EventPaste with the clipboard as the Data()
// Not implemented.
func (et *ETCellScreen) GetClipboard() {
	// Not implemented by tcell_ebiten.
}

// SetClipboard sets the content of the system clipboard to the bytes given.
// Not implemented.
func (et *ETCellScreen) SetClipboard(content []byte) {
	// Not implemented by tcell_ebiten.
}

// postEvent helper
func (et *ETCellScreen) postEvent(ev tcell.Event) (err error) {
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

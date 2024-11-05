// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package tcell_ebiten

import (
	"image"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type ETCellGame struct {
	*ETCell

	GeoM ebiten.GeoM // This should only be set initially, or modified in Draw(), Update(), or Layout() overrides.

	grid_draw []cell // Grid of cells, currently being drawn.
}

// Validate interface compliance
var _ ebiten.Game = (*ETCellGame)(nil)
var _ interface {
	LayoutF(w, h float64) (sw, sh float64)
} = (*ETCellGame)(nil)

// Update processes ebiten.Game events.
// If Screen.Suspend() has been called, does nothing.
func (et *ETCellGame) Update() (err error) {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	if et.close_error != nil {
		err = et.close_error
		et.close_error = nil
		return
	}

	if et.suspended {
		return
	}

	cursor_x, cursor_y := ebiten.CursorPosition()

	mapping := et.GeoM
	mapping.Invert()
	mouse_x, mouse_y := mapping.Apply(float64(cursor_x), float64(cursor_y))
	mouse := image.Point{X: int(mouse_x), Y: int(mouse_y)}

	var in_focus bool
	var posted bool

	// Mouse buttons
	if mouse.In(et.layout) {
		if !et.focused {
			et.postEvent(tcell.NewEventFocus(true))
			et.focused = true
			posted = true
		}
		var buttons tcell.ButtonMask
		for e_button, t_button := range ebiten_button_map {
			if ebiten.IsMouseButtonPressed(e_button) {
				buttons |= t_button
			}
		}

		// Translate from absolute mouse position to cell position.
		mouse_x := mouse.X / et.cell_size.X
		mouse_y := mouse.Y / et.cell_size.Y

		// Mouse wheel movement.
		xoff, yoff := ebiten.Wheel()
		if xoff < 0 {
			buttons |= tcell.WheelLeft
		}
		if xoff > 0 {
			buttons |= tcell.WheelRight
		}
		if yoff < 0 {
			buttons |= tcell.WheelDown
		}
		if yoff > 0 {
			buttons |= tcell.WheelUp
		}

		et.postEvent(tcell.NewEventMouse(mouse_x, mouse_y, buttons, modMask()))

		in_focus = true
		posted = true
	}

	if mouse.In(et.layout) {
		if !et.focused {
			et.postEvent(tcell.NewEventFocus(true))
			et.focused = true
		}
		mods := modMask()
		if (mods & tcell.ModCtrl) != 0 {
			keys := make([]ebiten.Key, 0, 16)
			for _, e_key := range inpututil.AppendPressedKeys(keys) {
				if !isKeyJustPressedOrRepeating(e_key) {
					continue
				}
				if e_key >= ebiten.KeyA && e_key <= ebiten.KeyZ {
					t_key := tcell.KeyCtrlA + tcell.Key(e_key-ebiten.KeyA)
					ev := tcell.NewEventKey(t_key, rune(0), mods & ^tcell.ModCtrl)
					et.postEvent(ev)
					posted = true
				}
			}
		} else {
			key_runes := ebiten.AppendInputChars(nil)
			for _, key_rune := range key_runes {
				ev := tcell.NewEventKey(tcell.KeyRune, key_rune, mods & ^tcell.ModShift)
				et.postEvent(ev)
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
				et.postEvent(ev)
				posted = true
			}
		}

		in_focus = true
	}

	if !in_focus {
		if et.focused {
			et.postEvent(tcell.NewEventFocus(false))
			et.focused = false
			posted = true
		}
	}

	// Always post a time event, if no other event was fired.
	if !posted {
		ev := &tcell.EventTime{}
		ev.SetEventNow()
		et.postEvent(ev)
	}

	return
}

// Draw handles drawing in the game context.
// Used to implement a custom override for ETCellGame.
func (et *ETCellGame) Draw(dst *ebiten.Image) {
	et.grid_lock.Lock()
	et.init()

	if cap(et.grid_draw) < len(et.grid) {
		et.grid_draw = make([]cell, len(et.grid))
	}
	et.grid_draw = et.grid_draw[0:len(et.grid)]
	copy(et.grid_draw, et.grid)
	geom := et.GeoM
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

// LayoutF returns the floating point layout.
func (et *ETCellGame) LayoutF(outsideWidth, outsideHeight float64) (screenWidth, screenHeight float64) {
	monitor_scale := ebiten.Monitor().DeviceScaleFactor()
	ow := int(float64(outsideWidth) * monitor_scale)
	oh := int(float64(outsideHeight) * monitor_scale)
	sw, sh := et.Layout(ow, oh)
	screenWidth = float64(sw)
	screenHeight = float64(sh)
	return
}

// Layout returns the integer layout.
func (et *ETCellGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	et.grid_lock.Lock()
	defer et.grid_lock.Unlock()

	screen_rows := outsideWidth / et.cell_size.X
	screen_cols := outsideHeight / et.cell_size.Y

	et.setScreenSize(screen_rows, screen_cols)

	screenWidth = et.layout.Dx()
	screenHeight = et.layout.Dy()

	return
}

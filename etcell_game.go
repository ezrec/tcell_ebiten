// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package etcell

import (
	"image"

	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type ETCellGame struct {
	*ETCell
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
	cursor := image.Point{X: cursor_x, Y: cursor_y}

	var in_focus bool
	var posted bool

	// Mouse buttons
	if et.mouse_capture.Empty() || cursor.In(et.mouse_capture) {
		mouse_mapping := et.mouse_mapping
		if mouse_mapping.Empty() {
			mouse_mapping = image.Rect(0, 0, et.grid_size.X, et.grid_size.Y)
		}

		mouse_capture := et.mouse_capture
		if mouse_capture.Empty() {
			mouse_capture = image.Rect(0, 0, et.layout.X, et.layout.Y)
		}

		if mouse_capture.Dx() == 0 || mouse_capture.Dy() == 0 {
			return
		}

		mouse := cursor.Sub(et.mouse_capture.Min)
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
		mouse_x := mouse.X
		mouse_y := mouse.Y

		mouse_x = (mouse_x * mouse_mapping.Dx() / mouse_capture.Dx())
		mouse_y = (mouse_y * mouse_mapping.Dy() / mouse_capture.Dy())
		mouse_x += et.mouse_mapping.Min.X
		mouse_y += et.mouse_mapping.Min.Y

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

	if et.key_capture.Empty() || cursor.In(et.key_capture) {
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

// Draw in ebiten.Game context.
// If Screen.Suspend() has been called, does nothing.
func (et *ETCellGame) Draw(dst *ebiten.Image) {
	var geom ebiten.GeoM
	et.DrawToImage(dst, geom)
}

// LayoutF returns the floating point layout.
func (et *ETCellGame) LayoutF(outsideWidth, outsideHeight float64) (screenWidth, screenHeight float64) {
	monitor_scale := float64(1.0)
	monitor_scale = ebiten.Monitor().DeviceScaleFactor()
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

	screenWidth = et.layout.X
	screenHeight = et.layout.Y

	return
}

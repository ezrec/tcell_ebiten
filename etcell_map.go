// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package tcell_ebiten

import (
	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// //////// ebiten interfaces ////////////////////
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

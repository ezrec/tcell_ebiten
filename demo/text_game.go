// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"context"

	"github.com/gdamore/tcell/v2"

	"github.com/ezrec/tcell_ebiten"
)

type TextGame struct {
	tcell_ebiten.GameScreen
	seen_key  map[tcell.Key]int
	seen_rune map[rune]int
}

func NewTextGame(screen tcell_ebiten.GameScreen) (tg *TextGame) {
	tg = &TextGame{
		GameScreen: screen,
		seen_key:   map[tcell.Key]int{},
		seen_rune:  map[rune]int{},
	}

	tg.SetStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorBlack))

	tg.SetCursorColor(tcell.ColorRed)

	return
}

const color_span = int(tcell.ColorWhite-tcell.ColorBlack) + 1

func (tg *TextGame) draw_key(k tcell.Key, v int) {
	// Determine postion
	x := 0
	y := 0
	switch {
	case k >= tcell.KeyCtrlSpace && k <= tcell.KeyCtrlUnderscore:
		x = int(k - tcell.KeyCtrlSpace)
		y = 0
	case k >= tcell.KeyRune:
		x = int(k - tcell.KeyRune)
		y = 1
	default:
		x = int(k) % 64
		y = 2
	}

	color := tcell.ColorBlack + tcell.Color(v%color_span)
	style := tcell.StyleDefault.Background(color)
	tg.SetContent(x, y, 'x', nil, style)
}

func (tg *TextGame) draw_rune(r rune, v int) {
	x := int(r) % 64
	y := (int(r) / 64) + 4

	color := tcell.ColorBlack + tcell.Color(v%color_span)
	style := tcell.StyleDefault.Background(color)
	tg.SetContent(x, y, r, nil, style)
}

func (tg *TextGame) redraw() {
	tg.Clear()

	for k, v := range tg.seen_key {
		tg.draw_key(k, v)
	}

	for r, v := range tg.seen_rune {
		tg.draw_rune(r, v)
	}

	tg.Show()
}

func (tg *TextGame) Run(ctx context.Context) {
	do_rune := func(r rune) {
		v, ok := tg.seen_rune[r]
		if !ok {
			tg.seen_rune[r] = 0
		} else {
			v++
			tg.seen_rune[r] = v
		}
		tg.draw_rune(r, v)
	}

	for ctx.Err() == nil {
		event := tg.PollEvent()
		switch ev := event.(type) {
		case *tcell.EventPaste:
			// tcell.EventPaste is not supported. Manually catch the
			// mouse or control key for pasting, and use a library such
			// as golang.design/x/clipboard to read the system clipboard.
		case *tcell.EventFocus:
			if ev.Focused {
				tg.SetCursorStyle(tcell.CursorStyleBlinkingBar)
			} else {
				tg.SetCursorStyle(tcell.CursorStyleSteadyBlock)
			}
		case *tcell.EventMouse:
			x, y := ev.Position()
			tg.ShowCursor(x, y)
			switch ev.Buttons() {
			case tcell.ButtonPrimary:
				do_rune(0x80)
			case tcell.ButtonMiddle:
				do_rune(0x81)
			case tcell.ButtonSecondary:
				do_rune(0x82)
			case tcell.Button4:
				do_rune(0x83)
			case tcell.Button5:
				do_rune(0x84)
			case tcell.Button6:
				do_rune(0x85)
			case tcell.Button7:
				do_rune(0x86)
			case tcell.Button8:
				do_rune(0x87)
			case tcell.WheelUp:
				do_rune(0x88)
			case tcell.WheelDown:
				do_rune(0x89)
			case tcell.WheelLeft:
				do_rune(0x8a)
			case tcell.WheelRight:
				do_rune(0x8b)
			}
			tg.Show()
		case *tcell.EventResize:
			tg.redraw()
			tg.Sync()
		case *tcell.EventInterrupt:
			tg.redraw()
			tg.Sync()
		case *tcell.EventKey:
			key := ev.Key()
			v, ok := tg.seen_key[key]
			if !ok {
				tg.seen_key[key] = 0
			} else {
				v++
				tg.seen_key[key] = v
			}
			tg.draw_key(key, v)
			if key == tcell.KeyRune {
				r := ev.Rune()
				do_rune(r)
			}
			tg.Show()
		}
	}
}

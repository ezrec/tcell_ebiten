// Copyright 2024, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"github.com/gdamore/tcell/v2"
)

type TextGame struct {
	tcell.Screen
	seen_key  map[tcell.Key]int
	seen_rune map[rune]int
}

func NewTextGame(screen tcell.Screen) (tg *TextGame) {
	tg = &TextGame{
		Screen:    screen,
		seen_key:  map[tcell.Key]int{},
		seen_rune: map[rune]int{},
	}

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
	tg.SetContent(1+x, 1+y, 'x', nil, style)
}

func (tg *TextGame) draw_rune(r rune, v int) {
	x := int(r) % 64
	y := (int(r) / 64) + 4

	color := tcell.ColorBlack + tcell.Color(v%color_span)
	style := tcell.StyleDefault.Background(color)
	tg.SetContent(1+x, 1+y, r, nil, style)
}

func (tg *TextGame) redraw() {
	tg.Clear()

	for k, v := range tg.seen_key {
		tg.draw_key(k, v)
	}

	for r, v := range tg.seen_rune {
		tg.draw_rune(r, v)
	}

	// Draw border
	max_x, max_y := tg.Size()

	style := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	for x := 1; x < (max_x - 1); x++ {
		for _, y := range []int{0, max_y - 1} {
			tg.SetContent(x, y, '─', nil, style)
		}
	}
	for y := 1; y < (max_y - 1); y++ {
		for _, x := range []int{0, max_x - 1} {
			tg.SetContent(x, y, '│', nil, style)
		}
	}
	tg.SetContent(0, 0, '╭', nil, style)
	tg.SetContent(max_x-1, 0, '╮', nil, style)
	tg.SetContent(0, max_y-1, '╰', nil, style)
	tg.SetContent(max_x-1, max_y-1, '╯', nil, style)

	// Show
	tg.Show()
}

func (tg *TextGame) Run() error {
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

	for {
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
			if key == tcell.KeyEnd {
				return nil
			}
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

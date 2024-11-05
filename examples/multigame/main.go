package main

import (
	"log"
	"math"
	"math/rand"
	"time"

	etcell "github.com/ezrec/tcell_ebiten"
	"github.com/ezrec/tcell_ebiten/font"

	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/gofont/gomono"
)

type Spinner struct {
	*etcell.ETCellGame

	w int
	h int
}

func (sp *Spinner) Layout(x, y int) (sx, sy int) {
	sx, sy = sp.ETCellGame.Layout(x, y)

	sp.w = sx
	sp.h = sy

	return
}

func (sp *Spinner) Update() (err error) {
	now := float64(time.Now().UnixMilli()) / 1000.0

	rotation_cycle_s := 5.0
	theta := math.Mod(now, rotation_cycle_s) / rotation_cycle_s * math.Pi * 2
	var geom ebiten.GeoM
	w_2 := float64(sp.w) / 2
	h_2 := float64(sp.h) / 2
	geom.Translate(-w_2, -h_2)
	geom.Rotate(theta)
	geom.Scale(0.5, 0.5)
	geom.Translate(w_2/2, h_2/2)

	sp.ETCellGame.GeoM = geom

	return sp.ETCellGame.Update()
}

type Zoomer struct {
	*etcell.ETCellGame
	w int
	h int
}

func (zm *Zoomer) Layout(x, y int) (sx, sy int) {
	sx, sy = zm.ETCellGame.Layout(x, y)

	zm.w = sx
	zm.h = sy

	return
}

func (zm *Zoomer) Update() (err error) {
	now := float64(time.Now().UnixMilli()) / 1000.0

	zoom_cycle_s := 6.0
	theta := math.Sin(math.Mod(now, zoom_cycle_s) / zoom_cycle_s * math.Pi * 2)

	scale := math.Abs(theta) / 2
	var geom ebiten.GeoM
	geom.Scale(scale, scale)
	w_2 := float64(zm.w) / 2.0
	h_2 := float64(zm.h) / 2.0
	geom.Translate(w_2, h_2)

	zm.ETCellGame.GeoM = geom

	return zm.ETCellGame.Update()
}

type Multiscreen struct {
	etcell.ETCell

	spin Spinner
	zoom Zoomer

	updating bool
}

func (m *Multiscreen) Update() (err error) {
	err = m.spin.Update()
	if err != nil {
		return
	}
	err = m.zoom.Update()
	if err != nil {
		return
	}

	return
}

func (m *Multiscreen) Layout(x, y int) (sx, sy int) {
	m.spin.Layout(x, y)
	m.zoom.Layout(x, y)
	return x, y
}

func (m *Multiscreen) LayoutF(x, y float64) (sx, sy float64) {
	monitor_scale := ebiten.Monitor().DeviceScaleFactor()
	ix := int(float64(x) * monitor_scale)
	iy := int(float64(y) * monitor_scale)
	m.spin.Layout(ix, iy)
	m.zoom.Layout(ix, iy)
	return x, y
}

func (m *Multiscreen) Draw(img *ebiten.Image) {
	m.spin.Draw(img)
	m.zoom.Draw(img)
}

func (n *Multiscreen) runner(screen tcell.Screen) (err error) {
	var cursor_x, cursor_y int

	n.updating = true

	style := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)

	for {
		event := screen.PollEvent()
		if event == nil {
			return
		}
		switch ev := event.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEnd:
				return nil
			case tcell.KeyRune:
				// 'any' key
				n.updating = !n.updating
			}
		case *tcell.EventMouse:
			cursor_x, cursor_y = ev.Position()
			screen.ShowCursor(cursor_x, cursor_y)
		case *tcell.EventResize:
			screen.Sync()
		case *tcell.EventInterrupt:
			screen.Sync()
		}

		if !n.updating {
			continue
		}

		width, height := screen.Size()
		for x := range width {
			for y := range height {
				randrune := rune(rand.Int() % (0x7f - 32))
				randattr := tcell.AttrMask(rand.Int() & 0xff)
				randfg := tcell.NewRGBColor(
					int32(rand.Int()&0xff),
					int32(rand.Int()&0xff),
					int32(rand.Int()&0xff),
				)
				randbg := tcell.NewRGBColor(
					int32(rand.Int()&0xff),
					int32(rand.Int()&0xff),
					int32(rand.Int()&0xff),
				)
				style = style.Attributes(randattr)
				style = style.Foreground(randfg)
				style = style.Background(randbg)
				screen.SetContent(x, y, randrune, nil, style)
			}
		}
		screen.Show()
	}
}

func (m *Multiscreen) Run() (err error) {
	go func() {
		screen := m.Screen()
		screen.Init()
		defer screen.Fini()
		err := m.runner(screen)

		m.Exit(err)
	}()

	return ebiten.RunGame(m)
}

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("etcell ms")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	ms := &Multiscreen{}

	ms.spin.ETCellGame = ms.NewGame()
	ms.zoom.ETCellGame = ms.NewGame()

	font, err := font.NewMonoFontFromTTF(gomono.TTF, 16)
	if err != nil {
		panic(err)
	}
	ms.SetFont(font)

	err = ms.Run()
	if err != nil {
		log.Fatal(err)
	}
}

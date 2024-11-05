# tcell Screen wrapper for ebiten

Easily add text components to your Ebiten game, leveraging the
`tcell` library and its add-ons.

This library provides a ebiten/v2.Game interface that can be
used as the entirety of the game loop for ebiten, or a sub-panel.

## Usage

### Convert `tcell` Application to `ebiten` Application

In your `tcell` program, add the following imports:

```
import (
    "github.com/ezrec/tcell_ebiten"
    "github.com/hajimehoshi/ebiten/v2"
)

|  // Define a nominal window size.
|  const window_width = 640
|  const window_height = 480

func main() {
    app := &views.Application{}
    window = ... // your existing tcell program here..
    app.SetRootWidget(window)

|  // Define our ebiten window size, and enable resizes.
|  ebiten.SetWindowSize(window_width, window_height)
|  ebiten.SetWindowTitle("hbox demo")
|  ebiten.SetWindowResizeable(true)
|
|  // Select an ebiten/v2/text/v2 font to use.
|  bm_face := text.NewGoXFace(bitmapfont.Face)
|  font_face, _ := font.NewMonoFont(bm_face)
|
|  // Create a proxy between the ebiten and tcell engines.
|  et := &etcell.ETCell{}
|  et.SetFont(font)
|
|  err := et.Run(func (screen tcell.Screen) error {
|      app.SetScreen(screen)
       err := app.Run()
       return err;
|  })
   if err != nil {
       panic(err)
   }
}
```

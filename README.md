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
    "github.com/hajimehoshi/bitmapfont/v3"
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/text/v2"
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
|  // Select an ebiten/v2/text/v2 font to use. The letter 'O'
|  // from the font face is used to determine the nominal font cell.
|
|  var font_face text.Face
|  font_face = text.NewGoXFace(bitmapfont.Face)
|
|  // Create a proxy between the ebiten and tcell engines.
|  gs := tcell_ebiten.NewGameScreen(font_face)
|  gs.Init()
|  defer gs.Fini()
|
|  go ebiten.RunGame(gs)

   app.SetScreen(gs)

   if e := app.Run(); e != nil {
        fmt.Fprintln(os.Stderr, e.Error())
        os.Exit(1)
    }
}
```

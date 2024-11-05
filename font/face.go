// Package font supplies font face wrappers for monospaced fonts,
// for use with [github.com/ezrec/ebiten_tcell].
package font

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	typesetting_font "github.com/go-text/typesetting/font"
	"github.com/hajimehoshi/ebiten/v2"
	ebiten_text "github.com/hajimehoshi/ebiten/v2/text/v2"
	image_font "golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
)

var ErrFontType = errors.New("unknown font source type")

// FontStyle selects which font style with which to render a rune.
type FontStyle int

const (
	FontStyleNormal     = FontStyle(iota) // Normal font style.
	FontStyleBold                         // Bold font style.
	FontStyleItalic                       // Italic font style.
	FontStyleBoldItalic                   // Bold & Italic font style.
)

// Face provides an interace to the font properties.
type Face interface {
	Metrics() (metrics ebiten_text.Metrics)
	Size() (width, height int) // Character cell size, in pixels.
	Glyph(character rune, style FontStyle) (glyph *ebiten.Image, is_empty bool)
	Empty() (empty_glyph *ebiten.Image)
}

// Implements Face
type CacheFont struct {
	FontMetrics ebiten_text.Metrics
	Cache       map[rune](*ebiten.Image)
	Width       int // Nominal cell width.
	Height      int // Nominal cell height.

	empty *ebiten.Image
}

// Assert interface compliance.
var _ Face = (*CacheFont)(nil)

// SetGlyph() sets a glyph into the cache.
func (mf *CacheFont) SetGlyph(character rune, glyph *ebiten.Image) {
	if mf.Cache == nil {
		mf.Cache = map[rune](*ebiten.Image){}
	}

	if glyph != nil {
		size := glyph.Bounds().Size()
		if size.X != mf.Width || size.Y != mf.Height {
			panic(fmt.Sprintf("invalid glyph size %vx%v for %vx%v font", size.X, size.Y, mf.Width, mf.Height))
		}
	}

	mf.Cache[character] = glyph
}

// Empty() returns the empty image.
func (mf *CacheFont) Empty() *ebiten.Image {
	if mf.empty == nil {
		mf.empty = ebiten.NewImage(mf.Width, mf.Height)
	}

	return mf.empty
}

// Metrics returns the font metrics.
func (mf *CacheFont) Metrics() ebiten_text.Metrics {
	return mf.FontMetrics
}

// Size of a cache-font cell.
func (mf *CacheFont) Size() (width, height int) {
	width = mf.Width
	height = mf.Height
	return
}

// HasGlyph returns true if a rune is in the font, in the specified style.
func (mf *CacheFont) HasGlyph(character rune, style FontStyle) (has bool) {
	glyph, ok := mf.Cache[character]

	// Short-circuit for cached glyphs.
	if ok && glyph != nil {
		has = true
		return
	}

	return
}

// Glyph returns a glyph for a rune. Rune glyphs are cached on their first access.
func (mf *CacheFont) Glyph(character rune, style FontStyle) (glyph *ebiten.Image, is_empty bool) {
	if mf.Cache == nil {
		mf.Cache = map[rune](*ebiten.Image){}
	}

	if mf.empty == nil {
		mf.empty = ebiten.NewImage(mf.Width, mf.Height)
	}

	glyph, ok := mf.Cache[character]
	if !ok {
		glyph = nil
		mf.Cache[character] = glyph
	}

	if glyph == nil {
		glyph = mf.empty
		is_empty = true
	}

	return
}

// Implements Face
type MonoFont struct {
	CacheFont
	Face ebiten_text.Face

	drawOptions ebiten_text.DrawOptions
}

// Assert interface compliance.
var _ Face = (*MonoFont)(nil)

// NewMonoFont() creates a new monospaced font face.
// Takes any of the following types:
//   - image_font.Face
//   - ebiten_text.Face
//   - nil (assumes GoMono TTF, size 11)
func NewMonoFont(source any) (mf *MonoFont, err error) {
	if source == nil {
		mf, err = NewMonoFontFromTTF(nil, 0)
		return
	}
	var face ebiten_text.Face
	switch source := source.(type) {
	case image_font.Face:
		face = ebiten_text.NewGoXFace(source)
	case ebiten_text.Face:
		face = source
	default:
		err = ErrFontType
		return
	}

	// We use the rune FULL_BLOCK to determine the nominal bounding box for the character set.
	const reference_rune = '█'

	metrics := face.Metrics()
	width_f, height_f := ebiten_text.Measure(string([]rune{reference_rune}), face, 0)
	width := int(width_f)
	height := int(height_f)

	// FIXME: For some reason we need to assume the font is
	//        smaller than the metric.
	scale_w := float64(width) / width_f
	scale_h := float64(height) / (height_f * 0.97)

	mf = &MonoFont{
		CacheFont: CacheFont{
			Width:       width,
			Height:      height,
			FontMetrics: metrics,
		},
		Face: face,
	}

	mf.drawOptions.GeoM.Scale(scale_w, scale_h)

	return
}

// NewMonoFontFromTTF creates a new monospaced font face from a TTF font.
// Takes any of the following types:
// - io.Reader (to a TTF source)
// - []byte (of a TTF blob)
// - nil (assumes GoMono TTF)
func NewMonoFontFromTTF(source any, size float64) (mf *MonoFont, err error) {
	if source == nil {
		source = gomono.TTF
	}

	if size == 0 {
		size = 11.0
	}

	var face ebiten_text.Face
	switch source := source.(type) {
	case []byte:
		return NewMonoFontFromTTF(bytes.NewReader(source), size)
	case io.Reader:
		var face_source *ebiten_text.GoTextFaceSource
		face_source, err = ebiten_text.NewGoTextFaceSource(source)
		if err != nil {
			return
		}
		face = &ebiten_text.GoTextFace{
			Source: face_source,
			Size:   size,
		}
	default:
		err = ErrFontType
		return
	}

	return NewMonoFont(face)
}

// HasGlyph returns true if a rune is in the font, in the specified style.
func (mf *MonoFont) HasGlyph(character rune, style FontStyle) (has bool) {
	// Short-circuit for cached glyphs.
	glyph, ok := mf.CacheFont.Cache[character]
	if ok {
		return glyph != nil
	}

	// Ugly, since text.Face does not export hasGlyph().
	switch ebiten_face := mf.Face.(type) {
	case (*ebiten_text.GoXFace):
		face := ebiten_face.UnsafeInternal()
		_, has = face.GlyphAdvance(character)
	case (*ebiten_text.GoTextFace):
		face := ebiten_face.Source.UnsafeInternal().(*typesetting_font.Face)
		_, has = face.NominalGlyph(character)
	default:
		// Some future internal ebiten/v2/text/v2 font face type.
		has = ebiten_text.Advance(string([]rune{character}), mf.Face) > 0.0
	}

	return
}

// Glyph returns a glyph for a rune. Rune glyphs are cached on their first access.
func (mf *MonoFont) Glyph(character rune, style FontStyle) (glyph *ebiten.Image, is_empty bool) {
	glyph, ok := mf.CacheFont.Cache[character]
	if !ok {
		if !mf.HasGlyph(character, style) {
			// Empty glyph.
			glyph = nil
		} else {
			// Generate new glyph for this rune.
			glyph = ebiten.NewImage(mf.Width, mf.Height)
			ebiten_text.Draw(glyph, string([]rune{character}), mf.Face, &mf.drawOptions)
		}

		mf.CacheFont.SetGlyph(character, glyph)
	}

	if glyph == nil {
		glyph = mf.CacheFont.Empty()
		is_empty = true
	}

	return
}

// FaceWithOnlyRunes limits the font to only the specified runes.
type FaceWithOnlyRunes struct {
	Face
	Runes []rune

	runemap map[rune](struct{})
}

// Assert interface compliance.
var _ Face = (*FaceWithOnlyRunes)(nil)

// Glyph returns the image for the rune, so long as it is in the mapping.
func (fm *FaceWithOnlyRunes) Glyph(character rune, style FontStyle) (glyph *ebiten.Image, is_empty bool) {
	if len(fm.runemap) != len(fm.Runes) {
		fm.runemap = make(map[rune](struct{}), len(fm.Runes))
		for _, r := range fm.Runes {
			fm.runemap[r] = struct{}{}
		}
	}

	_, ok := fm.runemap[character]
	if !ok {
		glyph = fm.Face.Empty()
		is_empty = true
	} else {
		glyph, is_empty = fm.Face.Glyph(character, style)
	}

	return
}

// FaceWithRuneMapping applies a rune mapping to a font.
// Implements [Face]
type FaceWithRuneMapping struct {
	Face
	RuneMapping map[rune]rune
}

// Assert interface compliance.
var _ Face = (*FaceWithRuneMapping)(nil)

// Glyph returns the image for the rune, mapped by the rune-to-rune mapping.
func (fm *FaceWithRuneMapping) Glyph(character rune, style FontStyle) (glyph *ebiten.Image, is_empty bool) {
	replacement, ok := fm.RuneMapping[character]
	if ok {
		character = replacement
	}

	return fm.Face.Glyph(character, style)
}

// FaceWithBackup allows a font be the 'backup' for another font, if the primary font doesn't have the right runes.
// Implements [Face]
type FaceWithBackup struct {
	Face
	Backup Face
}

// Glyph returns the image for the rune, using the backup font if needed.
func (fm *FaceWithBackup) Glyph(character rune, style FontStyle) (glyph *ebiten.Image, is_empty bool) {
	glyph, is_empty = fm.Face.Glyph(character, style)
	if !is_empty {
		return
	}

	glyph, is_empty = fm.Backup.Glyph(character, style)
	return
}

// FaceWithStyle has alternate fonts for bold or italic styles.
//
// FontStyleNormal must be mapped to a valid face.
// Implements [Face]
type FaceWithStyle struct {
	StyleMap map[FontStyle]Face
}

// Assert interface compliance.
var _ Face = (*FaceWithStyle)(nil)

func (fm *FaceWithStyle) forStyle(style FontStyle) (face Face) {
	var ok bool
	switch style {
	case FontStyleNormal:
		face, ok = fm.StyleMap[FontStyleNormal]
	case FontStyleItalic:
		face, ok = fm.StyleMap[FontStyleItalic]
		if !ok {
			face, ok = fm.StyleMap[FontStyleNormal]
		}
	case FontStyleBoldItalic:
		face, ok = fm.StyleMap[FontStyleBoldItalic]
		if ok {
			break
		}
		face, ok = fm.StyleMap[FontStyleItalic]
		if !ok {
			face, ok = fm.StyleMap[FontStyleBold]
			if !ok {
				face, ok = fm.StyleMap[FontStyleNormal]
			}
		}
	case FontStyleBold:
		face, ok = fm.StyleMap[FontStyleBold]
		if !ok {
			face, ok = fm.StyleMap[FontStyleNormal]
		}
	}

	if !ok {
		panic("FaceWithStyle.StyleMap[FontStyleNormal] does not exist")
	}

	return
}

// Metrics returns the font metrics.
func (fm *FaceWithStyle) Metrics() ebiten_text.Metrics {
	return fm.forStyle(FontStyleNormal).Metrics()
}

// Size returns the font size.
func (fm *FaceWithStyle) Size() (width, height int) {
	return fm.forStyle(FontStyleNormal).Size()
}

// Empty returns the empty glyph
func (fm *FaceWithStyle) Empty() (glyph *ebiten.Image) {
	return fm.forStyle(FontStyleNormal).Empty()
}

// Glyph returns the image for the rune, using the appropriate style font.
// FontStyleBoldItalic falls back to FontStyleBold
// FontStyleItalic falls back to FontStyleNormal
// FontStyleBold falls back to FontStyleNormal
//
// FontStyleNormal must be mapped.
//
// Style hints are passed unchanged to the underlying font.
func (fm *FaceWithStyle) Glyph(character rune, style FontStyle) (glyph *ebiten.Image, is_empty bool) {
	return fm.forStyle(style).Glyph(character, style)
}

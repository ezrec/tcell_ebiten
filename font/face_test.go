package font

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hajimehoshi/ebiten/v2"
)

const full_block = rune('█')
const bad_rune = rune(0x1234567)

func TestCacheFont(t *testing.T) {
	assert := assert.New(t)

	mf := &CacheFont{
		Width:  10,
		Height: 16,
	}

	w, h := mf.Size()
	assert.Equal(w, 10)
	assert.Equal(h, 16)

	block := ebiten.NewImage(w, h)
	block.Fill(color.White)
	mf.SetGlyph(full_block, block)

	for _, style := range []FontStyle{FontStyleNormal, FontStyleBold, FontStyleItalic, FontStyleBoldItalic} {
		assert.True(mf.HasGlyph(full_block, style))

		assert.False(mf.HasGlyph(bad_rune, style))

		// Verify we get a valid glyph.
		glyph, is_empty := mf.Glyph(full_block, style)
		assert.False(is_empty)
		size := glyph.Bounds().Size()
		assert.Equal(size.X, 10)
		assert.Equal(size.Y, 16)
		assert.NotSame(glyph, mf.Empty())

		// Verify we get the same cached glyph.
		g_cached, is_empty := mf.Glyph(full_block, style)
		assert.False(is_empty)
		assert.Same(glyph, g_cached)

		// Verify we get the empty glyph.
		glyph, is_empty = mf.Glyph(bad_rune, style)
		assert.True(is_empty)
		size = glyph.Bounds().Size()
		assert.Equal(size.X, 10)
		assert.Equal(size.Y, 16)
		assert.Same(glyph, mf.Empty())
	}

}

func TestMonoFont(t *testing.T) {
	assert := assert.New(t)

	_, err := NewMonoFont("a string")
	assert.Equal(ErrFontType, err)

	mf, err := NewMonoFont(nil)
	assert.Nil(err)

	w, h := mf.Size()
	assert.Equal(w, 6)
	assert.Equal(h, 12)

	for _, style := range []FontStyle{FontStyleNormal, FontStyleBold, FontStyleItalic, FontStyleBoldItalic} {
		assert.True(mf.HasGlyph(full_block, style))

		assert.False(mf.HasGlyph(bad_rune, style))

		// Verify we get a valid glyph.
		glyph, is_empty := mf.Glyph(full_block, style)
		assert.False(is_empty)
		size := glyph.Bounds().Size()
		assert.Equal(size.X, 6)
		assert.Equal(size.Y, 12)
		assert.NotSame(glyph, mf.Empty())

		// Verify we get the same cached glyph.
		g_cached, is_empty := mf.Glyph(full_block, style)
		assert.False(is_empty)
		assert.Same(glyph, g_cached)

		// Verify we get the empty glyph.
		glyph, is_empty = mf.Glyph(bad_rune, style)
		assert.True(is_empty)
		size = glyph.Bounds().Size()
		assert.Equal(size.X, 6)
		assert.Equal(size.Y, 12)
		assert.Same(glyph, mf.Empty())
	}

}

func TestFaceWithRuneMapping(t *testing.T) {
	assert := assert.New(t)

	cf := &CacheFont{
		Width:  7,
		Height: 13,
	}
	w, h := cf.Size()
	block := ebiten.NewImage(w, h)
	block.Fill(color.White)
	cf.SetGlyph('?', block)

	mf := &FaceWithRuneMapping{
		Face:        cf,
		RuneMapping: map[rune]rune{full_block: '?'},
	}

	for _, style := range []FontStyle{FontStyleNormal, FontStyleBold, FontStyleItalic, FontStyleBoldItalic} {
		// Verify we get a valid glyph.
		glyph, is_empty := mf.Glyph(full_block, style)
		assert.False(is_empty)
		size := glyph.Bounds().Size()
		assert.Equal(size.X, 7)
		assert.Equal(size.Y, 13)

		// Verify we get the same cached glyph.
		g_cached, is_empty := mf.Glyph(full_block, style)
		assert.False(is_empty)
		assert.Same(glyph, g_cached)

		// Verify we get the empty glyph.
		glyph, is_empty = mf.Glyph(bad_rune, style)
		assert.True(is_empty)
		size = glyph.Bounds().Size()
		assert.Equal(size.X, 7)
		assert.Equal(size.Y, 13)
	}
}

func TestFaceWithBackup(t *testing.T) {
	assert := assert.New(t)

	cf := &CacheFont{
		Width:  7,
		Height: 13,
	}
	w, h := cf.Size()
	block := ebiten.NewImage(w, h)
	block.Fill(color.White)
	cf.SetGlyph('?', block)

	bf := &CacheFont{
		Width:  7,
		Height: 13,
	}
	w, h = cf.Size()
	block = ebiten.NewImage(w, h)
	block.Fill(color.White)
	cf.SetGlyph(full_block, block)

	mf := &FaceWithBackup{
		Face:   cf,
		Backup: bf,
	}

	for _, style := range []FontStyle{FontStyleNormal, FontStyleBold, FontStyleItalic, FontStyleBoldItalic} {
		// Verify we get a valid glyph.
		glyph, is_empty := mf.Glyph(full_block, style)
		assert.False(is_empty)
		size := glyph.Bounds().Size()
		assert.Equal(size.X, 7)
		assert.Equal(size.Y, 13)

		// Verify we get the same cached glyph.
		g_cached, is_empty := mf.Glyph(full_block, style)
		assert.False(is_empty)
		assert.Same(glyph, g_cached)

		// Verify we get the empty glyph.
		glyph, is_empty = mf.Glyph(bad_rune, style)
		assert.True(is_empty)
		size = glyph.Bounds().Size()
		assert.Equal(size.X, 7)
		assert.Equal(size.Y, 13)
	}
}

func TestFaceStyle(t *testing.T) {
	assert := assert.New(t)

	type styledFont struct {
		Face  Face
		Glyph *ebiten.Image
	}

	style_glyph := map[FontStyle]styledFont{}

	styles := []FontStyle{FontStyleNormal, FontStyleBold, FontStyleItalic, FontStyleBoldItalic}
	for n, style := range styles {
		w := 7
		h := 13

		block := ebiten.NewImage(w, h)
		block.Fill(color.Gray{Y: uint8(n + 1)})

		cf := &CacheFont{
			Width:  w,
			Height: h,
		}
		cf.SetGlyph(full_block, block)

		style_glyph[style] = styledFont{
			Glyph: block,
			Face:  cf,
		}
	}

	table := [](struct {
		combo   []FontStyle
		mapping map[FontStyle]FontStyle
	}){
		// Only normal is set.
		{[]FontStyle{FontStyleNormal},
			map[FontStyle]FontStyle{
				FontStyleNormal:     FontStyleNormal,
				FontStyleItalic:     FontStyleNormal,
				FontStyleBoldItalic: FontStyleNormal,
				FontStyleBold:       FontStyleNormal,
			},
		},
		// Normal and Bold
		{[]FontStyle{FontStyleNormal, FontStyleBold},
			map[FontStyle]FontStyle{
				FontStyleNormal:     FontStyleNormal,
				FontStyleItalic:     FontStyleNormal,
				FontStyleBoldItalic: FontStyleBold,
				FontStyleBold:       FontStyleBold,
			},
		},
		// Normal and BoldItalic
		{[]FontStyle{FontStyleNormal, FontStyleBoldItalic},
			map[FontStyle]FontStyle{
				FontStyleNormal:     FontStyleNormal,
				FontStyleItalic:     FontStyleNormal,
				FontStyleBoldItalic: FontStyleBoldItalic,
				FontStyleBold:       FontStyleNormal,
			},
		},
		// Normal and Italic
		{[]FontStyle{FontStyleNormal, FontStyleItalic},
			map[FontStyle]FontStyle{
				FontStyleNormal:     FontStyleNormal,
				FontStyleItalic:     FontStyleItalic,
				FontStyleBoldItalic: FontStyleItalic,
				FontStyleBold:       FontStyleNormal,
			},
		},
		// Normal and Italic and Bold
		{[]FontStyle{FontStyleNormal, FontStyleItalic, FontStyleBold},
			map[FontStyle]FontStyle{
				FontStyleNormal:     FontStyleNormal,
				FontStyleItalic:     FontStyleItalic,
				FontStyleBoldItalic: FontStyleItalic, // Italic before bold.
				FontStyleBold:       FontStyleBold,
			},
		},
	}

	for _, entry := range table {
		mf := &FaceWithStyle{StyleMap: map[FontStyle]Face{}}
		for _, style := range entry.combo {
			mf.StyleMap[style] = style_glyph[style].Face
		}

		for style, expect := range entry.mapping {
			// Verify we get a valid glyph.
			glyph, is_empty := mf.Glyph(full_block, style)
			assert.False(is_empty)
			assert.Same(style_glyph[expect].Glyph, glyph)
		}
	}
}

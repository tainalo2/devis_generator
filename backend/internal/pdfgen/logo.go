package pdfgen

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"sync"
)

// logoPNG est le pictogramme du produit (même dessin que le favicon), rasterisé
// pour l’embarquer dans le PDF. Fond blanc : sur la page blanche, seuls les
// coins arrondis se voient.
func logoPNG() ([]byte, error) {
	logoOnce.Do(func() {
		logoBytes, logoErr = renderLogo(256)
	})
	return logoBytes, logoErr
}

var (
	logoOnce  sync.Once
	logoBytes []byte
	logoErr   error
)

func renderLogo(n int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	s := float64(n) / 64

	px := func(v float64) int { return int(math.Round(v * s)) }
	ink := color.RGBA{27, 58, 75, 255}
	paper := color.RGBA{232, 244, 242, 255}
	fold := color.RGBA{47, 111, 106, 255}
	accent := color.RGBA{47, 158, 148, 255}

	fillRoundRect(img, 0, 0, n, n, px(14), ink)
	fillRoundRect(img, px(16), px(16), px(28), px(34), px(3), paper)
	// Pli du coin supérieur droit de la feuille.
	strokeLine(img, px(40), px(16), px(40), px(24), maxInt(px(1.6), 2), fold)
	strokeLine(img, px(40), px(24), px(48), px(24), maxInt(px(1.6), 2), fold)
	sw := maxInt(px(2), 2)
	strokeLine(img, px(22), px(28), px(38), px(28), sw, ink)
	strokeLine(img, px(22), px(34), px(36), px(34), sw, ink)
	strokeLine(img, px(22), px(40), px(32), px(40), sw, ink)
	fillCircle(img, px(46), px(46), px(10), accent)
	strokeLine(img, px(42.5), px(46.2), px(45.1), px(48.8), sw, paper)
	strokeLine(img, px(45.1), px(48.8), px(49.9), px(43.6), sw, paper)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func fillRoundRect(img *image.RGBA, x, y, w, h, rad int, c color.RGBA) {
	if rad < 0 {
		rad = 0
	}
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if inRoundRect(xx, yy, x, y, w, h, rad) {
				img.SetRGBA(xx, yy, c)
			}
		}
	}
}

func inRoundRect(px, py, x, y, w, h, r int) bool {
	if px < x || py < y || px >= x+w || py >= y+h {
		return false
	}
	cx, cy := 0, 0
	switch {
	case px < x+r && py < y+r:
		cx, cy = x+r, y+r
	case px >= x+w-r && py < y+r:
		cx, cy = x+w-r-1, y+r
	case px >= x+w-r && py >= y+h-r:
		cx, cy = x+w-r-1, y+h-r-1
	case px < x+r && py >= y+h-r:
		cx, cy = x+r, y+h-r-1
	default:
		return true
	}
	dx, dy := px-cx, py-cy
	return dx*dx+dy*dy <= r*r
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	r2 := r * r
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 {
				if image.Pt(x, y).In(img.Bounds()) {
					img.SetRGBA(x, y, c)
				}
			}
		}
	}
}

func strokeLine(img *image.RGBA, x0, y0, x1, y1, sw int, c color.RGBA) {
	if sw < 1 {
		sw = 1
	}
	rad := sw / 2
	dx, dy := float64(x1-x0), float64(y1-y0)
	steps := int(math.Hypot(dx, dy))
	if steps < 1 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		fillCircle(img, int(math.Round(float64(x0)+dx*t)), int(math.Round(float64(y0)+dy*t)), rad, c)
	}
}

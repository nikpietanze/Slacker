// Command genart regenerates the addon's textures:
//
//	go run ./cmd/genart ..
//	go run ./cmd/genart -style flat ..   # or glass
//
// Shapes are drawn from signed distance functions and the "S" from an
// embedded outline, so edges are anti-aliased and the output is reproducible
// with no binary art sources or font files to keep in sync.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tc-hib/winres"
	"golang.org/x/image/vector"

	"slacker/companion/internal/bridge"
)

// Outline of "S" from Noto Sans Bold (SIL Open Font License 1.1), in font
// units (1000/em), y pointing down from the baseline.
const glyphS = `M511,-198 Q511,-103 442,-46 Q374,10 248,10 Q135,10 46,-33 L46,-174 Q97,-152 151,-133
Q206,-115 260,-115 Q316,-115 339,-136 Q363,-158 363,-191 Q363,-218 344,-237 Q326,-256 295,-272
Q264,-289 224,-308 Q199,-320 170,-336 Q141,-353 114,-377 Q88,-402 71,-437 Q54,-472 54,-521
Q54,-617 119,-670 Q184,-724 296,-724 Q352,-724 402,-711 Q453,-698 508,-674 L459,-556
Q410,-576 371,-587 Q332,-598 291,-598 Q248,-598 225,-578 Q202,-558 202,-526 Q202,-488 236,-466
Q270,-444 337,-412 Q392,-386 430,-358 Q469,-330 490,-292 Q511,-254 511,-198`

const size = 64 // texture size; shapes are drawn in these units

var (
	maskSS = 8.0 // glyph mask resolution per unit
	px     = 1.0 // units per output pixel (smaller for the large logo)
)

type rgba struct{ r, g, b, a float64 }

func hex(s string, a float64) rgba {
	v, _ := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 32)
	return rgba{float64(v>>16&255) / 255, float64(v>>8&255) / 255, float64(v&255) / 255, a}
}

func over(dst, src rgba) rgba {
	a := src.a + dst.a*(1-src.a)
	if a == 0 {
		return rgba{}
	}
	mix := func(s, d float64) float64 { return (s*src.a + d*dst.a*(1-src.a)) / a }
	return rgba{mix(src.r, dst.r), mix(src.g, dst.g), mix(src.b, dst.b), a}
}

func smooth(e0, e1, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-e0)/(e1-e0)))
	return t * t * (3 - 2*t)
}

// fill is the coverage of a shape at signed distance d, anti-aliased over ~1px.
func fill(d float64) float64 { return 1 - smooth(-px/2, px/2, d) }

func sdRoundRect(px, py, cx, cy, hw, hh, r float64) float64 {
	qx := math.Abs(px-cx) - hw + r
	qy := math.Abs(py-cy) - hh + r
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

func sdCircle(px, py, cx, cy, r float64) float64 { return math.Hypot(px-cx, py-cy) - r }

// sdTriangle: exact signed distance to a triangle (negative inside).
func sdTriangle(px, py float64, p [3][2]float64) float64 {
	d := math.Inf(1)
	s := 1.0
	for i := 0; i < 3; i++ {
		ax, ay := p[i][0], p[i][1]
		bx, by := p[(i+1)%3][0], p[(i+1)%3][1]
		ex, ey := bx-ax, by-ay
		wx, wy := px-ax, py-ay
		t := math.Max(0, math.Min(1, (wx*ex+wy*ey)/(ex*ex+ey*ey)))
		dx, dy := wx-ex*t, wy-ey*t
		d = math.Min(d, dx*dx+dy*dy)
		c1, c2, c3 := ay <= py, by > py, ex*wy > ey*wx
		if (c1 && c2 && c3) || (!c1 && !c2 && !c3) {
			s = -s
		}
	}
	return s * math.Sqrt(d)
}

// smin blends two distance fields so the tail flows into the body.
func smin(a, b, k float64) float64 {
	h := math.Max(k-math.Abs(a-b), 0) / k
	return math.Min(a, b) - h*h*k*0.25
}

// style describes one icon look.
type style struct {
	shape   func(x, y float64) float64
	glyphCX float64 // centre of the S
	glyphCY float64
	glyphH  float64 // cap height in px
	fillTop rgba
	fillBot rgba
	stroke  rgba    // inner stroke colour (alpha 0 = none)
	strokeW float64 // px
	edge    rgba    // 1px outer contrast edge
	glyph   rgba
	shadow  float64 // drop shadow strength
	bodyTop float64 // gradient range
	bodyBot float64
}

var styles = map[string]style{
	// Flat rounded bubble with a soft-merged tail.
	"flat": {
		shape: func(x, y float64) float64 {
			body := sdRoundRect(x, y, 32, 28.5, 26.5, 20.5, 15)
			tail := sdTriangle(x, y, [3][2]float64{{13, 38}, {27, 45}, {8.5, 57}})
			return smin(body, tail, 5)
		},
		glyphCX: 32, glyphCY: 28.5, glyphH: 21,
		fillTop: hex("#7468F2", 1), fillBot: hex("#5B4FE0", 1),
		edge:  hex("#120E2E", 0.55),
		glyph: hex("#FFFFFF", 1), shadow: 0.35, bodyTop: 8, bodyBot: 49,
	},
	// Round chat bubble.
	"round": {
		shape: func(x, y float64) float64 {
			body := sdCircle(x, y, 33, 29.5, 24.5)
			tail := sdTriangle(x, y, [3][2]float64{{13, 38}, {25, 49}, {7, 58}})
			return smin(body, tail, 6)
		},
		glyphCX: 33, glyphCY: 29.5, glyphH: 22,
		fillTop: hex("#7468F2", 1), fillBot: hex("#5B4FE0", 1),
		edge:  hex("#120E2E", 0.55),
		glyph: hex("#FFFFFF", 1), shadow: 0.35, bodyTop: 5, bodyBot: 54,
	},
	// Dark glass HUD bubble with a light outline.
	"glass": {
		shape: func(x, y float64) float64 {
			body := sdRoundRect(x, y, 32, 28.5, 26.5, 20.5, 15)
			tail := sdTriangle(x, y, [3][2]float64{{13, 38}, {27, 45}, {8.5, 57}})
			return smin(body, tail, 5)
		},
		glyphCX: 32, glyphCY: 28.5, glyphH: 21,
		fillTop: hex("#23232C", 0.92), fillBot: hex("#16161C", 0.92),
		stroke: hex("#FFFFFF", 0.85), strokeW: 2.2,
		edge:  hex("#000000", 0.5),
		glyph: hex("#FFFFFF", 1), shadow: 0.3, bodyTop: 8, bodyBot: 49,
	},
}

// glyphMask rasterizes the S at maskSS x texture resolution.
func glyphMask(st style) *image.Alpha {
	n := int(size * maskSS)
	r := vector.NewRasterizer(n, n)
	const capTop, capBot, left, right = -724.0, 10.0, 46.0, 511.0
	scale := st.glyphH / (capBot - capTop) * maskSS
	ox := st.glyphCX*maskSS - (left+right)/2*scale
	oy := st.glyphCY*maskSS - (capTop+capBot)/2*scale
	pt := func(s string) (float32, float32) {
		xy := strings.Split(s, ",")
		x, _ := strconv.ParseFloat(xy[0], 64)
		y, _ := strconv.ParseFloat(xy[1], 64)
		return float32(ox + x*scale), float32(oy + y*scale)
	}
	fields := strings.Fields(glyphS)
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		switch f[0] {
		case 'M':
			r.MoveTo(pt(f[1:]))
		case 'L':
			r.LineTo(pt(f[1:]))
		case 'Q':
			cx, cy := pt(f[1:])
			i++
			x, y := pt(fields[i])
			r.QuadTo(cx, cy, x, y)
		}
	}
	r.ClosePath()
	mask := image.NewAlpha(image.Rect(0, 0, n, n))
	r.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	return mask
}

func icon(st style) func(x, y float64) rgba {
	mask := glyphMask(st)
	return func(x, y float64) rgba {
		var c rgba
		if st.shadow > 0 {
			sd := st.shape(x, y-1.6)
			c = over(c, rgba{0, 0, 0, st.shadow * (1 - smooth(-1.5, 2.5, sd))})
		}
		d := st.shape(x, y)
		if d > 1.5 {
			return c
		}
		// 1px dark edge just outside the body for contrast on bright scenery
		c = over(c, rgba{st.edge.r, st.edge.g, st.edge.b, st.edge.a * fill(d-1.5*px)})
		t := math.Max(0, math.Min(1, (y-st.bodyTop)/(st.bodyBot-st.bodyTop)))
		body := rgba{
			st.fillTop.r + (st.fillBot.r-st.fillTop.r)*t,
			st.fillTop.g + (st.fillBot.g-st.fillTop.g)*t,
			st.fillTop.b + (st.fillBot.b-st.fillTop.b)*t,
			(st.fillTop.a + (st.fillBot.a-st.fillTop.a)*t) * fill(d),
		}
		c = over(c, body)
		if st.stroke.a > 0 {
			ring := fill(d) * smooth(-st.strokeW-0.5, -st.strokeW+0.5, d)
			c = over(c, rgba{st.stroke.r, st.stroke.g, st.stroke.b, st.stroke.a * ring})
		}
		mx, my := int(x*maskSS), int(y*maskSS)
		if g := float64(mask.AlphaAt(mx, my).A) / 255; g > 0 {
			c = over(c, rgba{st.glyph.r, st.glyph.g, st.glyph.b, st.glyph.a * g * fill(d)})
		}
		return c
	}
}

// Badge: flat red disc with a thin dark contrast ring. The addon stretches
// its middle column into a pill for two or three digits.
func badge(x, y float64) rgba {
	d := sdCircle(x, y, 32, 32, 30.5)
	if d > 1 {
		return rgba{}
	}
	c := rgba{0.05, 0.02, 0.04, 0.5 * fill(d)}
	c = over(c, rgba{0.937, 0.231, 0.239, fill(d + 3)}) // #EF3B3D
	return c
}

func glow(x, y float64) rgba {
	r := math.Hypot(x-32, y-32) / 32
	return rgba{1, 1, 1, math.Pow(math.Max(0, 1-r), 2)}
}

// render draws an n x n image of the 64-unit design.
func render(n int, shade func(x, y float64) rgba) []byte {
	out := make([]byte, n*n*4)
	const ss = 4
	u := float64(size) / float64(n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var acc rgba
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					c := shade((float64(x)+(float64(sx)+0.5)/ss)*u, (float64(y)+(float64(sy)+0.5)/ss)*u)
					acc.r += c.r * c.a
					acc.g += c.g * c.a
					acc.b += c.b * c.a
					acc.a += c.a
				}
			}
			i := (y*n + x) * 4
			if acc.a > 0 {
				out[i] = byte(math.Round(255 * acc.r / acc.a))
				out[i+1] = byte(math.Round(255 * acc.g / acc.a))
				out[i+2] = byte(math.Round(255 * acc.b / acc.a))
			}
			out[i+3] = byte(math.Round(255 * acc.a / (ss * ss)))
		}
	}
	return out
}

func write(path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", path)
}

// at sets the resolution globals for an n x n render.
func at(n int) {
	px = float64(size) / float64(n)
	maskSS = math.Max(8, 2*float64(n)/size)
}

func toImage(n int, pix []byte) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	copy(img.Pix, pix)
	return img
}

func pngBytes(img image.Image) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// unreadDot adds a small red dot to the top right of an icon.
func unreadDot(shade func(x, y float64) rgba) func(x, y float64) rgba {
	return func(x, y float64) rgba {
		c := shade(x, y)
		d := sdCircle(x, y, 50, 14, 12)
		c = over(c, rgba{0.07, 0.05, 0.12, 0.9 * fill(d)})
		return over(c, rgba{0.937, 0.231, 0.239, fill(d + 2.5)})
	}
}

// template is the macOS menu bar version: a black bubble with the S cut out.
func template(st style) func(x, y float64) rgba {
	mask := glyphMask(st)
	return func(x, y float64) rgba {
		a := fill(st.shape(x, y))
		g := float64(mask.AlphaAt(int(x*maskSS), int(y*maskSS)).A) / 255
		return rgba{0, 0, 0, a * (1 - g)}
	}
}

// ico packs PNG images into a Windows .ico (PNG entries, Vista and later).
func ico(images map[int][]byte, sizes []int) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, []uint16{0, 1, uint16(len(sizes))})
	offset := 6 + 16*len(sizes)
	for _, n := range sizes {
		dim := byte(n)
		if n >= 256 {
			dim = 0
		}
		b.Write([]byte{dim, dim, 0, 0})
		binary.Write(&b, binary.LittleEndian, []uint16{1, 32})
		binary.Write(&b, binary.LittleEndian, []uint32{uint32(len(images[n])), uint32(offset)})
		offset += len(images[n])
	}
	for _, n := range sizes {
		b.Write(images[n])
	}
	return b.Bytes()
}

// companionArt writes the tray icons, the macOS app icon set and the Windows
// .exe icon resources into the companion folder.
func companionArt(st style, companionDir string) {
	flat := st
	flat.shadow = 0 // shadows turn to mud at tray sizes
	assets := filepath.Join(companionDir, "assets")

	sizes := []int{16, 20, 24, 32, 40, 48, 64}
	normal, unread := map[int][]byte{}, map[int][]byte{}
	for _, n := range sizes {
		at(n)
		normal[n] = pngBytes(toImage(n, render(n, icon(flat))))
		at(n)
		unread[n] = pngBytes(toImage(n, render(n, unreadDot(icon(flat)))))
	}
	write(filepath.Join(assets, "tray.ico"), ico(normal, sizes))
	write(filepath.Join(assets, "tray-unread.ico"), ico(unread, sizes))
	write(filepath.Join(assets, "tray.png"), normal[64])
	write(filepath.Join(assets, "tray-unread.png"), unread[64])
	at(36)
	write(filepath.Join(assets, "tray-template.png"), pngBytes(toImage(36, render(36, template(st)))))

	// macOS app icon: icon_NxN.png and @2x variants for iconutil.
	iconset := filepath.Join(assets, "Slacker.iconset")
	for _, n := range []int{16, 32, 128, 256, 512} {
		for _, scale := range []int{1, 2} {
			name := fmt.Sprintf("icon_%dx%d.png", n, n)
			if scale == 2 {
				name = fmt.Sprintf("icon_%dx%d@2x.png", n, n)
			}
			at(n * scale)
			write(filepath.Join(iconset, name), pngBytes(toImage(n*scale, render(n*scale, icon(st)))))
		}
	}

	// Windows .exe icon and manifest, linked automatically by "go build".
	var imgs []image.Image
	for _, n := range []int{16, 24, 32, 48, 64, 128, 256} {
		at(n)
		imgs = append(imgs, toImage(n, render(n, icon(st))))
	}
	appIcon, err := winres.NewIconFromImages(imgs)
	if err != nil {
		panic(err)
	}
	rs := &winres.ResourceSet{}
	rs.SetIcon(winres.Name("APPICON"), appIcon)
	rs.SetManifest(winres.AppManifest{
		Description:         "Slacker companion",
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
	})
	for arch, a := range map[string]winres.Arch{"amd64": winres.ArchAMD64, "arm64": winres.ArchARM64} {
		var b bytes.Buffer
		if err := rs.WriteObject(&b, a); err != nil {
			panic(err)
		}
		write(filepath.Join(companionDir, "rsrc_windows_"+arch+".syso"), b.Bytes())
	}
	at(size)
}

func main() {
	styleName := flag.String("style", "round", "icon style: round, flat or glass")
	logo := flag.String("logo", "", "also write a large PNG of the icon to this path")
	companion := flag.String("companion", "", "also write the companion's tray and app icons into this folder")
	flag.Parse()
	st, ok := styles[*styleName]
	if flag.NArg() != 1 || !ok {
		fmt.Fprintln(os.Stderr, "usage: genart [-style round|flat|glass] [-logo file.png] [-companion dir] <addon folder (the repo root)>")
		os.Exit(2)
	}
	dir := flag.Arg(0)
	write(filepath.Join(dir, "Media", "Icon.tga"), bridge.RGBATGA(size, size, render(size, icon(st))))
	write(filepath.Join(dir, "Media", "Badge.tga"), bridge.RGBATGA(size, size, render(size, badge)))
	write(filepath.Join(dir, "Media", "Glow.tga"), bridge.RGBATGA(size, size, render(size, glow)))
	w, h := bridge.Dimensions(bridge.CodeOffline)
	write(filepath.Join(dir, "Bridge", "count.tga"), bridge.BlankTGA(w, h))
	if *logo != "" {
		at(512)
		write(*logo, pngBytes(toImage(512, render(512, icon(st)))))
		at(size)
	}
	if *companion != "" {
		companionArt(st, *companion)
	}
}

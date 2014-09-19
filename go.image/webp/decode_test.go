// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webp

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// hex is like fmt.Sprintf("% x", x) but also inserts dots every 16 bytes, to
// delineate VP8 macroblock boundaries.
func hex(x []byte) string {
	buf := new(bytes.Buffer)
	for len(x) > 0 {
		n := len(x)
		if n > 16 {
			n = 16
		}
		fmt.Fprintf(buf, " . % x", x[:n])
		x = x[n:]
	}
	return buf.String()
}

func TestDecodeVP8(t *testing.T) {
	testCases := []string{
		"blue-purple-pink",
		"blue-purple-pink-large.no-filter",
		"blue-purple-pink-large.simple-filter",
		"blue-purple-pink-large.normal-filter",
		"video-001",
		"yellow_rose",
	}

	for _, tc := range testCases {
		f0, err := os.Open("../testdata/" + tc + ".lossy.webp")
		if err != nil {
			t.Errorf("%s: Open WEBP: %v", tc, err)
			continue
		}
		defer f0.Close()
		img0, err := Decode(f0)
		if err != nil {
			t.Errorf("%s: Decode WEBP: %v", tc, err)
			continue
		}

		m0, ok := img0.(*image.YCbCr)
		if !ok || m0.SubsampleRatio != image.YCbCrSubsampleRatio420 {
			t.Errorf("%s: decoded WEBP image is not a 4:2:0 YCbCr", tc)
			continue
		}
		// w2 and h2 are the half-width and half-height, rounded up.
		w, h := m0.Bounds().Dx(), m0.Bounds().Dy()
		w2, h2 := int((w+1)/2), int((h+1)/2)

		f1, err := os.Open("../testdata/" + tc + ".lossy.webp.ycbcr.png")
		if err != nil {
			t.Errorf("%s: Open PNG: %v", tc, err)
			continue
		}
		defer f1.Close()
		img1, err := png.Decode(f1)
		if err != nil {
			t.Errorf("%s: Open PNG: %v", tc, err)
			continue
		}

		// The split-into-YCbCr-planes golden image is a 2*w2 wide and h+h2 high
		// gray image arranged in IMC4 format:
		//   YYYY
		//   YYYY
		//   BBRR
		// See http://www.fourcc.org/yuv.php#IMC4
		if got, want := img1.Bounds(), image.Rect(0, 0, 2*w2, h+h2); got != want {
			t.Errorf("%s: bounds0: got %v, want %v", tc, got, want)
			continue
		}
		m1, ok := img1.(*image.Gray)
		if !ok {
			t.Errorf("%s: decoded PNG image is not a Gray", tc)
			continue
		}

		planes := []struct {
			name     string
			m0Pix    []uint8
			m0Stride int
			m1Rect   image.Rectangle
		}{
			{"Y", m0.Y, m0.YStride, image.Rect(0, 0, w, h)},
			{"Cb", m0.Cb, m0.CStride, image.Rect(0*w2, h, 1*w2, h+h2)},
			{"Cr", m0.Cr, m0.CStride, image.Rect(1*w2, h, 2*w2, h+h2)},
		}
		for _, plane := range planes {
			dx := plane.m1Rect.Dx()
			nDiff, diff := 0, make([]byte, dx)
			for j, y := 0, plane.m1Rect.Min.Y; y < plane.m1Rect.Max.Y; j, y = j+1, y+1 {
				got := plane.m0Pix[j*plane.m0Stride:][:dx]
				want := m1.Pix[y*m1.Stride+plane.m1Rect.Min.X:][:dx]
				if bytes.Equal(got, want) {
					continue
				}
				nDiff++
				if nDiff > 10 {
					t.Errorf("%s: %s plane: more rows differ", tc, plane.name)
					break
				}
				for i := range got {
					diff[i] = got[i] - want[i]
				}
				t.Errorf("%s: %s plane: m0 row %d, m1 row %d\ngot %s\nwant%s\ndiff%s",
					tc, plane.name, j, y, hex(got), hex(want), hex(diff))
			}
		}
	}
}

func TestDecodeVP8L(t *testing.T) {
	testCases := []string{
		"blue-purple-pink",
		"blue-purple-pink-large",
		"gopher-doc.1bpp",
		"gopher-doc.2bpp",
		"gopher-doc.4bpp",
		"gopher-doc.8bpp",
		"tux",
		"yellow_rose",
	}

loop:
	for _, tc := range testCases {
		f0, err := os.Open("../testdata/" + tc + ".lossless.webp")
		if err != nil {
			t.Errorf("%s: Open WEBP: %v", tc, err)
			continue
		}
		defer f0.Close()
		img0, err := Decode(f0)
		if err != nil {
			t.Errorf("%s: Decode WEBP: %v", tc, err)
			continue
		}
		m0, ok := img0.(*image.NRGBA)
		if !ok {
			t.Errorf("%s: WEBP image is %T, want *image.NRGBA", tc, img0)
			continue
		}

		f1, err := os.Open("../testdata/" + tc + ".png")
		if err != nil {
			t.Errorf("%s: Open PNG: %v", tc, err)
			continue
		}
		defer f1.Close()
		img1, err := png.Decode(f1)
		if err != nil {
			t.Errorf("%s: Decode PNG: %v", tc, err)
			continue
		}
		m1, ok := img1.(*image.NRGBA)
		if !ok {
			rgba1, ok := img1.(*image.RGBA)
			if !ok {
				t.Fatalf("%s: PNG image is %T, want *image.NRGBA", tc, img1)
				continue
			}
			if !rgba1.Opaque() {
				t.Fatalf("%s: PNG image is non-opaque *image.RGBA, want *image.NRGBA", tc)
				continue
			}
			// The image is fully opaque, so we can re-interpret the RGBA pixels
			// as NRGBA pixels.
			m1 = &image.NRGBA{
				Pix:    rgba1.Pix,
				Stride: rgba1.Stride,
				Rect:   rgba1.Rect,
			}
		}

		b0, b1 := m0.Bounds(), m1.Bounds()
		if b0 != b1 {
			t.Errorf("%s: bounds: got %v, want %v", tc, b0, b1)
			continue
		}
		for i := range m0.Pix {
			if m0.Pix[i] != m1.Pix[i] {
				y := i / m0.Stride
				x := (i - y*m0.Stride) / 4
				i = 4 * (y*m0.Stride + x)
				t.Errorf("%s: at (%d, %d):\ngot  %02x %02x %02x %02x\nwant %02x %02x %02x %02x",
					tc, x, y,
					m0.Pix[i+0], m0.Pix[i+1], m0.Pix[i+2], m0.Pix[i+3],
					m1.Pix[i+0], m1.Pix[i+1], m1.Pix[i+2], m1.Pix[i+3],
				)
				continue loop
			}
		}
	}
}

func benchmarkDecode(b *testing.B, filename string) {
	data, err := ioutil.ReadFile("../testdata/blue-purple-pink-large." + filename + ".webp")
	if err != nil {
		b.Fatal(err)
	}
	s := string(data)
	cfg, err := DecodeConfig(strings.NewReader(s))
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(cfg.Width * cfg.Height * 4))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(strings.NewReader(s))
	}
}

func BenchmarkDecodeVP8NoFilter(b *testing.B)     { benchmarkDecode(b, "no-filter.lossy") }
func BenchmarkDecodeVP8SimpleFilter(b *testing.B) { benchmarkDecode(b, "simple-filter.lossy") }
func BenchmarkDecodeVP8NormalFilter(b *testing.B) { benchmarkDecode(b, "normal-filter.lossy") }
func BenchmarkDecodeVP8L(b *testing.B)            { benchmarkDecode(b, "lossless") }

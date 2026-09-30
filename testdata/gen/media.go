package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
)

// svgBadge is the vector image embedded by the images fixture.
const svgBadge = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
  <title>Sample vector badge</title>
  <rect width="64" height="64" fill="#1f6feb"/>
  <text x="32" y="40" font-family="sans-serif" font-size="20" fill="#ffffff" text-anchor="middle">EN</text>
</svg>
`

// sampleImage draws a small two-tone bitmap shared by the raster fixtures.
func sampleImage() *image.RGBA {
	const size = 16
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	blue := &image.Uniform{C: color.RGBA{R: 0x1f, G: 0x6f, B: 0xeb, A: 0xff}}
	white := &image.Uniform{C: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}}
	draw.Draw(img, img.Bounds(), blue, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, size/2, size/2), white, image.Point{}, draw.Src)
	return img
}

func pngBytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, sampleImage()); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
}

func jpegBytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, sampleImage(), &jpeg.Options{Quality: 80}); err != nil {
		return nil, fmt.Errorf("encoding JPEG: %w", err)
	}
	return buf.Bytes(), nil
}

func gifBytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := gif.Encode(&buf, sampleImage(), &gif.Options{NumColors: 16}); err != nil {
		return nil, fmt.Errorf("encoding GIF: %w", err)
	}
	return buf.Bytes(), nil
}

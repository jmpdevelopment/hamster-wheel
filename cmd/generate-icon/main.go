package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

func main() {
	// Create a 22x22 template icon (black on transparent for macOS)
	size := 22
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	// Draw a simple hamster wheel (circle with spokes)
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 2
	innerR := r - 3

	// Outer ring
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= r && dist >= innerR {
				img.Set(x, y, color.Black)
			}
		}
	}

	// Center dot
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= 2 {
				img.Set(x, y, color.Black)
			}
		}
	}

	// Draw 4 spokes from center to edge
	for i := 0; i < 4; i++ {
		angle := float64(i) * math.Pi / 2
		for t := 3.0; t < innerR; t += 0.5 {
			x := int(cx + t*math.Cos(angle))
			y := int(cy + t*math.Sin(angle))
			if x >= 0 && x < size && y >= 0 && y < size {
				img.Set(x, y, color.Black)
			}
		}
	}

	// Save as iconTemplate.png (Template suffix for macOS)
	f, err := os.Create("assets/iconTemplate.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}

	log.Println("Generated assets/iconTemplate.png")
}

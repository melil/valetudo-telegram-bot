package valetudo

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

// RenderMapPNG renders a ValetudoMap struct to PNG bytes.
func RenderMapPNG(m *ValetudoMap) ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("map is nil")
	}
	if m.PixelSize <= 0 {
		m.PixelSize = 5
	}

	minX, maxX := math.MaxInt32, math.MinInt32
	minY, maxY := math.MaxInt32, math.MinInt32

	updateBounds := func(x, y int) {
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}

	// Unpack layers
	type unpackedLayer struct {
		layerType string
		segID     string
		pixels    [][2]int
	}
	var unpacked []unpackedLayer

	for _, l := range m.Layers {
		var pts [][2]int
		if len(l.CompressedPixels) > 0 {
			for i := 0; i+2 < len(l.CompressedPixels); i += 3 {
				xStart := l.CompressedPixels[i]
				y := l.CompressedPixels[i+1]
				count := l.CompressedPixels[i+2]
				for j := 0; j < count; j++ {
					px, py := xStart+j, y
					pts = append(pts, [2]int{px, py})
					updateBounds(px, py)
				}
			}
		} else if len(l.Pixels) > 0 {
			for i := 0; i+1 < len(l.Pixels); i += 2 {
				px, py := l.Pixels[i], l.Pixels[i+1]
				pts = append(pts, [2]int{px, py})
				updateBounds(px, py)
			}
		}
		unpacked = append(unpacked, unpackedLayer{
			layerType: l.Type,
			segID:     l.MetaData.SegmentID,
			pixels:    pts,
		})
	}

	if minX > maxX || minY > maxY {
		return nil, fmt.Errorf("map contains no layer pixels")
	}

	pad := 8
	mapW := (maxX - minX + 1) + 2*pad
	mapH := (maxY - minY + 1) + 2*pad

	scale := 5
	if mapW > 0 && mapH > 0 {
		maxDim := mapW
		if mapH > maxDim {
			maxDim = mapH
		}
		scale = 800 / maxDim
		if scale < 2 {
			scale = 2
		}
		if scale > 8 {
			scale = 8
		}
	}

	imgW := mapW * scale
	imgH := mapH * scale

	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	bgColor := color.RGBA{R: 26, G: 27, B: 38, A: 255} // #1a1b26 (Tokyo Night dark)
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	segmentPalette := []color.RGBA{
		{R: 59, G: 82, B: 139, A: 255},  // Blue
		{R: 38, G: 120, B: 120, A: 255}, // Teal
		{R: 72, G: 122, B: 84, A: 255},  // Green
		{R: 128, G: 80, B: 138, A: 255}, // Violet
		{R: 138, G: 104, B: 58, A: 255}, // Amber
		{R: 56, G: 102, B: 138, A: 255}, // Steel
		{R: 130, G: 68, B: 78, A: 255},  // Rose
		{R: 70, G: 110, B: 100, A: 255}, // Seafoam
	}

	segColorMap := make(map[string]color.RGBA)
	colorIdx := 0

	drawBlock := func(px, py int, c color.RGBA) {
		cx := (px - minX + pad) * scale
		cy := (py - minY + pad) * scale
		for dy := 0; dy < scale; dy++ {
			for dx := 0; dx < scale; dx++ {
				img.SetRGBA(cx+dx, cy+dy, c)
			}
		}
	}

	// 1. Draw floor and segments
	for _, ul := range unpacked {
		if ul.layerType == "wall" {
			continue
		}
		var col color.RGBA
		if ul.layerType == "floor" {
			col = color.RGBA{R: 44, G: 50, B: 68, A: 255}
		} else {
			if _, exists := segColorMap[ul.segID]; !exists {
				segColorMap[ul.segID] = segmentPalette[colorIdx%len(segmentPalette)]
				colorIdx++
			}
			col = segColorMap[ul.segID]
		}
		for _, pt := range ul.pixels {
			drawBlock(pt[0], pt[1], col)
		}
	}

	// 2. Draw walls
	wallColor := color.RGBA{R: 169, G: 177, B: 214, A: 255} // #a9b1d6 crisp border
	for _, ul := range unpacked {
		if ul.layerType == "wall" {
			for _, pt := range ul.pixels {
				drawBlock(pt[0], pt[1], wallColor)
			}
		}
	}

	toCanvas := func(wx, wy int) (int, int) {
		fx := float64(wx) / float64(m.PixelSize)
		fy := float64(wy) / float64(m.PixelSize)
		cx := int((fx - float64(minX) + float64(pad)) * float64(scale))
		cy := int((fy - float64(minY) + float64(pad)) * float64(scale))
		return cx, cy
	}

	drawLine := func(x0, y0, x1, y1 int, col color.RGBA, width int) {
		dx := math.Abs(float64(x1 - x0))
		dy := math.Abs(float64(y1 - y0))
		sx := -1
		if x0 < x1 {
			sx = 1
		}
		sy := -1
		if y0 < y1 {
			sy = 1
		}
		errVal := dx - dy

		setThickPixel := func(cx, cy int) {
			r := width / 2
			for dy := -r; dy <= r; dy++ {
				for dx := -r; dx <= r; dx++ {
					if dx*dx+dy*dy <= r*r+1 {
						nx, ny := cx+dx, cy+dy
						if nx >= 0 && nx < imgW && ny >= 0 && ny < imgH {
							img.SetRGBA(nx, ny, col)
						}
					}
				}
			}
		}

		for {
			setThickPixel(x0, y0)
			if x0 == x1 && y0 == y1 {
				break
			}
			e2 := 2 * errVal
			if e2 > -dy {
				errVal -= dy
				x0 += sx
			}
			if e2 < dx {
				errVal += dx
				y0 += sy
			}
		}
	}

	drawCircle := func(cx, cy, r int, fillCol, borderCol color.RGBA) {
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				d2 := x*x + y*y
				if d2 <= r*r {
					nx, ny := cx+x, cy+y
					if nx >= 0 && nx < imgW && ny >= 0 && ny < imgH {
						if d2 >= (r-2)*(r-2) {
							img.SetRGBA(nx, ny, borderCol)
						} else {
							img.SetRGBA(nx, ny, fillCol)
						}
					}
				}
			}
		}
	}

	// 3. Draw Carpets (if any)
	carpetBorder := color.RGBA{R: 210, G: 195, B: 90, A: 255} // Light gold accent border
	for _, ent := range m.Entities {
		if ent.Type == "carpet" && len(ent.Points) >= 4 {
			numPts := len(ent.Points) / 2
			for i := 0; i < numPts; i++ {
				j := (i + 1) % numPts
				x0, y0 := toCanvas(ent.Points[2*i], ent.Points[2*i+1])
				x1, y1 := toCanvas(ent.Points[2*j], ent.Points[2*j+1])
				drawLine(x0, y0, x1, y1, carpetBorder, 1)
			}
		}
	}

	// 4. Draw Paths (cleaned route)
	pathColor := color.RGBA{R: 255, G: 255, B: 255, A: 255} // Pure crisp white
	lineWidth := scale / 2
	if lineWidth < 2 {
		lineWidth = 2
	}
	if lineWidth > 4 {
		lineWidth = 4
	}

	for _, ent := range m.Entities {
		if (ent.Type == "path" || ent.Type == "predicted_path") && len(ent.Points) >= 4 {
			col := pathColor
			if ent.Type == "predicted_path" {
				col = color.RGBA{R: 147, G: 197, B: 253, A: 180} // Light sky blue
			}
			for i := 0; i+3 < len(ent.Points); i += 2 {
				x0, y0 := toCanvas(ent.Points[i], ent.Points[i+1])
				x1, y1 := toCanvas(ent.Points[i+2], ent.Points[i+3])
				drawLine(x0, y0, x1, y1, col, lineWidth)
			}
		}
	}

	// 5. Draw Charger Location
	dockFill := color.RGBA{R: 16, G: 185, B: 129, A: 255}   // Emerald Green
	dockBorder := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	dockSize := scale + 3
	for _, ent := range m.Entities {
		if ent.Type == "charger_location" && len(ent.Points) >= 2 {
			cx, cy := toCanvas(ent.Points[0], ent.Points[1])
			drawCircle(cx, cy, dockSize, dockFill, dockBorder)
		}
	}

	// 6. Draw Robot Position
	robotFill := color.RGBA{R: 245, G: 158, B: 11, A: 255}   // Amber
	robotBorder := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	robotSize := scale + 2
	for _, ent := range m.Entities {
		if ent.Type == "robot_position" && len(ent.Points) >= 2 {
			cx, cy := toCanvas(ent.Points[0], ent.Points[1])
			drawCircle(cx, cy, robotSize, robotFill, robotBorder)
		}
	}

	// 7. Draw Obstacles
	obsFill := color.RGBA{R: 239, G: 68, B: 68, A: 255}     // Red alert
	obsBorder := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	obsSize := scale - 1
	if obsSize < 3 {
		obsSize = 3
	}
	for _, ent := range m.Entities {
		if ent.Type == "obstacle" && len(ent.Points) >= 2 {
			cx, cy := toCanvas(ent.Points[0], ent.Points[1])
			drawCircle(cx, cy, obsSize, obsFill, obsBorder)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

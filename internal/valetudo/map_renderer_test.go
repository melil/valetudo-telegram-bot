package valetudo

import (
	"bytes"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderMapPNG_ValidMap(t *testing.T) {
	// Sample minimalist Valetudo map
	m := &ValetudoMap{
		PixelSize: 5,
		Layers: []MapLayer{
			{
				Type: "segment",
				MetaData: MapLayerMeta{
					SegmentID: "1",
					Name:      "Kitchen",
				},
				CompressedPixels: []int{
					10, 10, 5, // 10..14, y=10
					10, 11, 5, // 10..14, y=11
					10, 12, 5, // 10..14, y=12
				},
			},
			{
				Type: "wall",
				CompressedPixels: []int{
					9, 9, 7,
					9, 13, 7,
				},
			},
		},
		Entities: []MapEntity{
			{
				Type:   "charger_location",
				Points: []int{55, 55}, // 55 / 5 = 11, in pixel space
			},
			{
				Type:   "robot_position",
				Points: []int{60, 60}, // 60 / 5 = 12, in pixel space
			},
			{
				Type:   "path",
				Points: []int{55, 55, 60, 60},
			},
			{
				Type:   "obstacle",
				Points: []int{65, 65},
			},
			{
				Type:   "carpet",
				Points: []int{50, 50, 60, 50, 60, 60, 50, 60},
			},
		},
	}

	pngBytes, err := RenderMapPNG(m)
	if err != nil {
		t.Fatalf("RenderMapPNG failed: %v", err)
	}

	if len(pngBytes) == 0 {
		t.Fatal("RenderMapPNG returned 0 bytes")
	}

	// Verify that the output is a valid PNG
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("png.Decode failed on rendered bytes: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		t.Fatalf("invalid image dimensions: %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestRenderMapPNG_Errors(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		_, err := RenderMapPNG(nil)
		if err == nil {
			t.Error("expected error for nil map, got nil")
		}
	})

	t.Run("empty layers", func(t *testing.T) {
		m := &ValetudoMap{
			PixelSize: 5,
			Layers:    []MapLayer{},
		}
		_, err := RenderMapPNG(m)
		if err == nil {
			t.Error("expected error for empty layers, got nil")
		}
	})
}

func TestRenderMapPNG_RealMap(t *testing.T) {
	// If real_map.json exists in root, test rendering it
	data, err := os.ReadFile(filepath.Join("..", "..", "real_map.json"))
	if err != nil {
		// Try test_map.json
		data, err = os.ReadFile(filepath.Join("..", "..", "test_map.json"))
		if err != nil {
			t.Skip("no sample map json file found, skipping real map test")
		}
	}

	var m ValetudoMap
	trimmed := bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	trimmed = bytes.TrimPrefix(trimmed, []byte("\xff\xfe"))
	if len(trimmed) == 0 {
		t.Skip("sample file is empty, skipping")
	}

	if err := json.Unmarshal(trimmed, &m); err != nil {
		t.Skipf("cannot unmarshal json: %v", err)
	}

	pngBytes, err := RenderMapPNG(&m)
	if err != nil {
		t.Fatalf("failed to render real map: %v", err)
	}
	if len(pngBytes) == 0 {
		t.Fatal("rendered real map is empty")
	}
}

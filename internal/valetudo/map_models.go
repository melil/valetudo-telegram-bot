package valetudo

// ValetudoMap represents the vendor-agnostic map structure from Valetudo API (/api/v2/robot/state/map).
type ValetudoMap struct {
	Class     string      `json:"__class"`
	MetaData  MapMetaData `json:"metaData"`
	Size      MapSize     `json:"size"`
	PixelSize int         `json:"pixelSize"`
	Layers    []MapLayer  `json:"layers"`
	Entities  []MapEntity `json:"entities"`
}

type MapMetaData struct {
	VendorMapID    int    `json:"vendorMapId,omitempty"`
	Version        int    `json:"version,omitempty"`
	Nonce          string `json:"nonce,omitempty"`
	TotalLayerArea int    `json:"totalLayerArea,omitempty"`
}

type MapSize struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type MapLayer struct {
	Class            string       `json:"__class,omitempty"`
	Type             string       `json:"type"`
	MetaData         MapLayerMeta `json:"metaData"`
	Pixels           []int        `json:"pixels"`
	CompressedPixels []int        `json:"compressedPixels"`
}

type MapLayerMeta struct {
	SegmentID string `json:"segmentId,omitempty"`
	Name      string `json:"name,omitempty"`
	Material  string `json:"material,omitempty"`
	Active    bool   `json:"active,omitempty"`
	Area      int    `json:"area,omitempty"`
}

type MapEntity struct {
	Class    string         `json:"__class,omitempty"`
	Type     string         `json:"type"`
	MetaData map[string]any `json:"metaData,omitempty"`
	Points   []int          `json:"points"`
}

package valetudo

// --- Valetudo Models ---

type CleanSegmentPayload struct {
	Action     string   `json:"action"`
	SegmentIDs []string `json:"segment_ids"`
	Iterations int      `json:"iterations"`
}

type MapSegment struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type PresetActionPayload struct {
	Name string `json:"name"`
}

type GenericAttribute struct {
	Class string `json:"__class"`
	Type  string `json:"type,omitempty"`
	Value any    `json:"value,omitempty"`
	Level int    `json:"level,omitempty"`
	Flag  string `json:"flag,omitempty"`
}

type ConsumableRemaining struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

type ConsumableItem struct {
	Type      string              `json:"type"`
	SubType   string              `json:"subType"`
	Remaining ConsumableRemaining `json:"remaining"`
}

type AvailableConsumableProperty struct {
	Type     string `json:"type"`
	SubType  string `json:"subType"`
	Unit     string `json:"unit"`
	MaxValue int    `json:"maxValue"`
}

type ConsumableProperties struct {
	AvailableConsumables []AvailableConsumableProperty `json:"availableConsumables"`
}

type DataPoint struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

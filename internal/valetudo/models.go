package valetudo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

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

type RobotErrorSeverity struct {
	Kind  string `json:"kind,omitempty"`
	Level string `json:"level,omitempty"`
}

type RobotError struct {
	Severity        *RobotErrorSeverity `json:"severity,omitempty"`
	Subsystem       string              `json:"subsystem,omitempty"`
	Message         string              `json:"message,omitempty"`
	VendorErrorCode any                 `json:"vendorErrorCode,omitempty"`
}

func (e *RobotError) GetVendorErrorCode() string {
	if e == nil || e.VendorErrorCode == nil {
		return ""
	}
	switch v := e.VendorErrorCode.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.Itoa(int(v))
	case int:
		return strconv.Itoa(v)
	default:
		s := fmt.Sprintf("%v", v)
		return strings.TrimSpace(s)
	}
}

func (e *RobotError) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		e.Message = str
		return nil
	}
	type Alias RobotError
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*e = RobotError(aux)
	return nil
}

type GenericAttribute struct {
	Class string      `json:"__class"`
	Type  string      `json:"type,omitempty"`
	Value any         `json:"value,omitempty"`
	Level int         `json:"level,omitempty"`
	Flag  string      `json:"flag,omitempty"`
	Error *RobotError `json:"error,omitempty"`
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

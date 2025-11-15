package models

import (
	"encoding/json"
	"time"
)

type Couple struct {
	AnchorTimeMs uint32
	SongID       uint32
}

type RecordData struct {
	Audio      string   `json:"audio"`
	Duration   float64  `json:"duration"`
	Channels   int      `json:"channels"`
	SampleRate int      `json:"sampleRate"`
	SampleSize int      `json:"sampleSize"`
	Latitude   *float64 `json:"latitude,omitempty"`
	Longitude  *float64 `json:"longitude,omitempty"`
}

// Detection represents a stored drone detection with location and metadata
type Detection struct {
	ID              int64                  `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	Latitude        *float64               `json:"latitude,omitempty"`
	Longitude       *float64               `json:"longitude,omitempty"`
	IsDrone         bool                   `json:"isDrone"`
	PrimaryType     string                 `json:"primaryType,omitempty"`
	PrimaryLabel    string                 `json:"primaryLabel,omitempty"`
	PrimaryCategory string                 `json:"primaryCategory,omitempty"`
	Confidence      float64                `json:"confidence"`
	SNRDb           float64                `json:"snrDb,omitempty"`
	LatencyMs       float64                `json:"latencyMs"`
	Predictions     json.RawMessage        `json:"predictions"` // Store as JSON
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CountryOfOrigin string                 `json:"countryOfOrigin,omitempty"`
	RecordingPath   string                 `json:"recordingPath,omitempty"`
}

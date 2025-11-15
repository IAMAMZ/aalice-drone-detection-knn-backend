package drone

// Prototype represents a single embedding vector describing a labelled audio asset.
type Prototype struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Category    string            `json:"category"`
	Description string            `json:"description,omitempty"`
	Source      string            `json:"source,omitempty"`
	Features    []float64         `json:"features"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PrototypeScore captures the similarity between the analysed audio and a stored prototype.
type PrototypeScore struct {
	ID       string  `json:"id"`
	Distance float64 `json:"distance"`
	Weight   float64 `json:"weight"`
	Source   string  `json:"source,omitempty"`
}

// Prediction summarises the per-class aggregation across nearest prototypes.
type Prediction struct {
	Label            string            `json:"label"`
	Category         string            `json:"category"`
	Type             string            `json:"type"`
	Description      string            `json:"description,omitempty"`
	Confidence       float64           `json:"confidence"`
	AverageDist      float64           `json:"averageDistance"`
	Support          int               `json:"support"`
	TopPrototypes    []PrototypeScore  `json:"topPrototypes"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	ThreatAssessment *ThreatAssessment `json:"threatAssessment,omitempty"` // Defense-focused intelligence
}

// WindowPrediction captures predictions for a specific temporal window.
type WindowPrediction struct {
	Index       int          `json:"index"`
	Start       float64      `json:"start"`       // seconds
	End         float64      `json:"end"`         // seconds
	Predictions []Prediction `json:"predictions"` // sorted by confidence
}

// ModelStats exposes metadata about the loaded prototype collection.
type ModelStats struct {
	PrototypeCount int              `json:"prototypeCount"`
	LabelCount     int              `json:"labelCount"`
	Labels         []ModelLabelStat `json:"labels"`
	UsingExample   bool             `json:"usingExample"`
}

// ModelLabelStat summarises prototype density per label.
type ModelLabelStat struct {
	Label      string `json:"label"`
	Category   string `json:"category"`
	Prototypes int    `json:"prototypes"`
}

// ClassificationSummary packages the raw predictions together with auxiliary telemetry.
type ClassificationSummary struct {
	Predictions       []Prediction       `json:"predictions"`
	IsDrone           bool               `json:"isDrone"`
	LatencyMs         float64            `json:"latencyMs"`
	FeatureVector     []float64          `json:"featureVector"`
	PrimaryType       string             `json:"primaryType,omitempty"`
	SNRDb             float64            `json:"snrDb,omitempty"`             // Signal-to-noise ratio in dB
	AdjustedThreshold float64            `json:"adjustedThreshold,omitempty"` // Threshold used after SNR adjustment
	Windows           []WindowPrediction `json:"windows,omitempty"`
	Latitude          *float64           `json:"latitude,omitempty"`
	Longitude         *float64           `json:"longitude,omitempty"`
	RecordingPath     string             `json:"recordingPath,omitempty"`
	TemplatePreds     []Prediction       `json:"templatePredictions,omitempty"`
}

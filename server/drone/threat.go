package drone

import "fmt"

// ThreatAssessment provides defense-focused intelligence about a detected drone
type ThreatAssessment struct {
	ThreatLevel                   string  `json:"threatLevel,omitempty"`
	RiskCategory                  string  `json:"riskCategory,omitempty"`
	PayloadCapacityKg             float64 `json:"payloadCapacityKg,omitempty"`
	MaxRangeKm                    float64 `json:"maxRangeKm,omitempty"`
	MaxSpeedMs                    float64 `json:"maxSpeedMs,omitempty"`
	FlightTimeMinutes             int     `json:"flightTimeMinutes,omitempty"`
	JammingSusceptible            bool    `json:"jammingSusceptible,omitempty"`
	CountermeasureRecommendations string  `json:"countermeasureRecommendations,omitempty"`
	DetectionRangeM               float64 `json:"detectionRangeM,omitempty"`
	OperatorType                  string  `json:"operatorType,omitempty"`
	IsMilitaryGrade               bool    `json:"isMilitaryGrade,omitempty"`
}

// ExtractThreatAssessment extracts defense-relevant information from prediction metadata
func ExtractThreatAssessment(prediction Prediction) ThreatAssessment {
	ta := ThreatAssessment{}

	if prediction.Metadata == nil {
		return ta
	}

	// Threat assessment fields
	if val, ok := prediction.Metadata["threat_level"]; ok {
		ta.ThreatLevel = val
	}
	if val, ok := prediction.Metadata["risk_category"]; ok {
		ta.RiskCategory = val
	}

	// Technical specifications
	if val, ok := prediction.Metadata["payload_capacity_kg"]; ok {
		if f, err := parseFloat(val); err == nil {
			ta.PayloadCapacityKg = f
		}
	}
	if val, ok := prediction.Metadata["max_range_km"]; ok {
		if f, err := parseFloat(val); err == nil {
			ta.MaxRangeKm = f
		}
	}
	if val, ok := prediction.Metadata["max_speed_ms"]; ok {
		if f, err := parseFloat(val); err == nil {
			ta.MaxSpeedMs = f
		}
	}
	if val, ok := prediction.Metadata["flight_time_minutes"]; ok {
		if i, err := parseInt(val); err == nil {
			ta.FlightTimeMinutes = i
		}
	}
	if val, ok := prediction.Metadata["detection_range_m"]; ok {
		if f, err := parseFloat(val); err == nil {
			ta.DetectionRangeM = f
		}
	}

	// Countermeasure information
	if val, ok := prediction.Metadata["jamming_susceptible"]; ok {
		ta.JammingSusceptible = (val == "true" || val == "yes" || val == "1")
	}
	if val, ok := prediction.Metadata["countermeasure_recommendations"]; ok {
		ta.CountermeasureRecommendations = val
	}

	// Operational intelligence
	if val, ok := prediction.Metadata["operator_type"]; ok {
		ta.OperatorType = val
	}
	if val, ok := prediction.Metadata["is_military_grade"]; ok {
		ta.IsMilitaryGrade = (val == "true" || val == "yes" || val == "1")
	}

	return ta
}

// Helper functions for parsing
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

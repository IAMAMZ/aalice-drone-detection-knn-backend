package drone

// Feature Scaling and Normalization
//
// This package implements feature scaling to ensure all feature dimensions contribute
// meaningfully to distance calculations. Without proper scaling, features with large
// magnitudes (like spectral crest factor ≈0.997) completely dominate the feature vector
// after L2 normalization, making all prototypes indistinguishable.
//
// The solution is to scale each feature dimension to a similar range BEFORE applying
// L2 normalization. This ensures that spectral, temporal, and harmonic features all
// contribute to the final distance metric.

import (
	"errors"
	"math"
)

// FeatureScaler standardizes features across a dataset using z-score normalization.
// Each feature dimension is transformed to have mean=0 and std=1.
type FeatureScaler struct {
	Mean   []float64 `json:"mean"`
	Stddev []float64 `json:"stddev"`
}

// NewFeatureScalerFromPrototypes computes scaling parameters from a set of prototypes
func NewFeatureScalerFromPrototypes(prototypes []Prototype) (*FeatureScaler, error) {
	if len(prototypes) == 0 {
		return nil, errors.New("no prototypes provided")
	}

	featureCount := len(prototypes[0].Features)
	if featureCount == 0 {
		return nil, errors.New("prototypes have no features")
	}

	// Calculate mean for each feature dimension
	mean := make([]float64, featureCount)
	for _, proto := range prototypes {
		if len(proto.Features) != featureCount {
			return nil, errors.New("inconsistent feature dimensions")
		}
		for i, val := range proto.Features {
			mean[i] += val
		}
	}
	for i := range mean {
		mean[i] /= float64(len(prototypes))
	}

	// Calculate standard deviation for each feature dimension
	stddev := make([]float64, featureCount)
	for _, proto := range prototypes {
		for i, val := range proto.Features {
			diff := val - mean[i]
			stddev[i] += diff * diff
		}
	}
	for i := range stddev {
		stddev[i] = math.Sqrt(stddev[i] / float64(len(prototypes)))
		// Prevent division by zero for constant features
		if stddev[i] < 1e-10 {
			stddev[i] = 1.0
		}
	}

	return &FeatureScaler{
		Mean:   mean,
		Stddev: stddev,
	}, nil
}

// Transform applies z-score standardization to a feature vector
func (fs *FeatureScaler) Transform(features []float64) []float64 {
	if len(features) != len(fs.Mean) {
		return features // Return unchanged if dimensions don't match
	}

	scaled := make([]float64, len(features))
	for i, val := range features {
		scaled[i] = (val - fs.Mean[i]) / fs.Stddev[i]
	}

	return scaled
}

// TransformAndNormalize applies scaling followed by L2 normalization
func (fs *FeatureScaler) TransformAndNormalize(features []float64) []float64 {
	scaled := fs.Transform(features)
	NormaliseVectorInPlace(scaled)
	return scaled
}

// MinMaxScaler scales features to [0, 1] range based on min/max values
type MinMaxScaler struct {
	Min   []float64 `json:"min"`
	Range []float64 `json:"range"` // max - min
}

// NewMinMaxScalerFromPrototypes computes min-max scaling parameters
func NewMinMaxScalerFromPrototypes(prototypes []Prototype) (*MinMaxScaler, error) {
	if len(prototypes) == 0 {
		return nil, errors.New("no prototypes provided")
	}

	featureCount := len(prototypes[0].Features)
	if featureCount == 0 {
		return nil, errors.New("prototypes have no features")
	}

	// Initialize with first prototype
	min := make([]float64, featureCount)
	max := make([]float64, featureCount)
	copy(min, prototypes[0].Features)
	copy(max, prototypes[0].Features)

	// Find min and max for each dimension
	for _, proto := range prototypes[1:] {
		if len(proto.Features) != featureCount {
			return nil, errors.New("inconsistent feature dimensions")
		}
		for i, val := range proto.Features {
			if val < min[i] {
				min[i] = val
			}
			if val > max[i] {
				max[i] = val
			}
		}
	}

	// Calculate range
	featureRange := make([]float64, featureCount)
	for i := range featureRange {
		featureRange[i] = max[i] - min[i]
		// Prevent division by zero for constant features
		if featureRange[i] < 1e-10 {
			featureRange[i] = 1.0
		}
	}

	return &MinMaxScaler{
		Min:   min,
		Range: featureRange,
	}, nil
}

// Transform applies min-max scaling to a feature vector
func (mms *MinMaxScaler) Transform(features []float64) []float64 {
	if len(features) != len(mms.Min) {
		return features // Return unchanged if dimensions don't match
	}

	scaled := make([]float64, len(features))
	for i, val := range features {
		scaled[i] = (val - mms.Min[i]) / mms.Range[i]
		// Clamp to [0, 1] range to handle out-of-range values
		if scaled[i] < 0 {
			scaled[i] = 0
		}
		if scaled[i] > 1 {
			scaled[i] = 1
		}
	}

	return scaled
}

// TransformAndNormalize applies scaling followed by L2 normalization
func (mms *MinMaxScaler) TransformAndNormalize(features []float64) []float64 {
	scaled := mms.Transform(features)
	NormaliseVectorInPlace(scaled)
	return scaled
}

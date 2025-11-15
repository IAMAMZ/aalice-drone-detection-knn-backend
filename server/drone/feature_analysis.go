package drone

import (
	"fmt"
	"math"
)

// FeatureScaleAnalysis analyzes the scales of features before normalization
// to identify potential scale mismatches that could affect classification.
type FeatureScaleAnalysis struct {
	FeatureNames []string
	MinValues    []float64
	MaxValues    []float64
	MeanValues   []float64
	StdValues    []float64
}

// AnalyzeFeatureScales examines a set of feature vectors to understand their scales
func AnalyzeFeatureScales(prototypes []Prototype) FeatureScaleAnalysis {
	if len(prototypes) == 0 {
		return FeatureScaleAnalysis{}
	}

	featureCount := len(prototypes[0].Features)
	analysis := FeatureScaleAnalysis{
		FeatureNames: getFeatureNames(),
		MinValues:    make([]float64, featureCount),
		MaxValues:    make([]float64, featureCount),
		MeanValues:   make([]float64, featureCount),
		StdValues:    make([]float64, featureCount),
	}

	// Initialize min/max with extreme values
	for i := range analysis.MinValues {
		analysis.MinValues[i] = math.MaxFloat64
		analysis.MaxValues[i] = math.SmallestNonzeroFloat64
	}

	// Collect all values for each feature
	values := make([][]float64, featureCount)
	for i := range values {
		values[i] = make([]float64, 0, len(prototypes))
	}

	for _, proto := range prototypes {
		// Use unnormalized features if available, otherwise denormalize
		// For now, we'll analyze normalized features to see their distribution
		for i, val := range proto.Features {
			if i < featureCount {
				values[i] = append(values[i], val)
				if val < analysis.MinValues[i] {
					analysis.MinValues[i] = val
				}
				if val > analysis.MaxValues[i] {
					analysis.MaxValues[i] = val
				}
			}
		}
	}

	// Calculate mean and std for each feature
	for i := range values {
		if len(values[i]) == 0 {
			continue
		}
		var sum float64
		for _, val := range values[i] {
			sum += val
		}
		analysis.MeanValues[i] = sum / float64(len(values[i]))

		var variance float64
		for _, val := range values[i] {
			diff := val - analysis.MeanValues[i]
			variance += diff * diff
		}
		analysis.StdValues[i] = math.Sqrt(variance / float64(len(values[i])))
	}

	return analysis
}

// PrintFeatureScaleReport prints a detailed report of feature scales
func (f *FeatureScaleAnalysis) PrintFeatureScaleReport() {
	fmt.Println("\n=== Feature Scale Analysis ===")
	fmt.Printf("%-25s %12s %12s %12s %12s %12s\n", "Feature", "Min", "Max", "Mean", "Std", "Range")
	fmt.Println("--------------------------------------------------------------------------------")

	for i, name := range f.FeatureNames {
		if i >= len(f.MinValues) {
			break
		}
		rangeVal := f.MaxValues[i] - f.MinValues[i]
		fmt.Printf("%-25s %12.6f %12.6f %12.6f %12.6f %12.6f\n",
			name, f.MinValues[i], f.MaxValues[i], f.MeanValues[i], f.StdValues[i], rangeVal)
	}
	fmt.Println()
}

// CheckScaleIssues identifies potential scale mismatches
func (f *FeatureScaleAnalysis) CheckScaleIssues() []string {
	issues := []string{}

	// Check for features with very different ranges
	maxRange := 0.0
	minRange := math.MaxFloat64

	for i := range f.FeatureNames {
		if i >= len(f.MinValues) {
			break
		}
		rangeVal := f.MaxValues[i] - f.MinValues[i]
		if rangeVal > maxRange {
			maxRange = rangeVal
		}
		if rangeVal < minRange {
			minRange = rangeVal
		}
	}

	// Check for features with very large standard deviations relative to mean
	for i, name := range f.FeatureNames {
		if i >= len(f.MeanValues) {
			break
		}
		if math.Abs(f.MeanValues[i]) > 1e-9 {
			coeffVar := f.StdValues[i] / math.Abs(f.MeanValues[i])
			if coeffVar > 2.0 {
				issues = append(issues, fmt.Sprintf(
					"Feature '%s' has high coefficient of variation (%.2f), indicating high variability",
					name, coeffVar))
			}
		}
	}

	// Check for features that might dominate after normalization
	// Features with larger magnitudes will have more influence in L2 normalization
	totalSquaredMean := 0.0
	for i := range f.MeanValues {
		totalSquaredMean += f.MeanValues[i] * f.MeanValues[i]
	}

	for i, name := range f.FeatureNames {
		if i >= len(f.MeanValues) {
			break
		}
		contribution := (f.MeanValues[i] * f.MeanValues[i]) / totalSquaredMean
		if contribution > 0.2 {
			issues = append(issues, fmt.Sprintf(
				"Feature '%s' contributes %.1f%% of normalized vector magnitude, may dominate classification",
				name, contribution*100))
		}
	}

	return issues
}

func getFeatureNames() []string {
	return []string{
		"Energy (RMS)",
		"Zero Crossing Rate",
		"Spectral Centroid",
		"Spectral Bandwidth",
		"Spectral Rolloff",
		"Spectral Flatness",
		"Dominant Frequency",
		"Spectral Crest Factor",
		"Spectral Entropy",
		"Variance",
		"Temporal Centroid",
		"Onset Rate",
		"Amplitude Modulation Depth",
		"Spectral Skewness",
		"Spectral Kurtosis",
		"Peak Prominence",
		"Harmonic Ratio",
		"Harmonic Count",
		"Harmonic Strength",
	}
}

// ConfidenceAnalysis explains why confidence might not be 100% even for identical audio
func ExplainConfidenceCalculation() string {
	return `
Confidence Score Calculation Explanation:

The confidence score is calculated using k-nearest neighbors (k-NN) classification:

1. Distance Calculation:
   - Cosine similarity is computed between input features and all prototypes
   - Distance = 1 - cosine_similarity (ranges from 0 to 2)
   - Lower distance = more similar

2. Weight Calculation:
   - Weight = 1 / (distance + epsilon)
   - Closer neighbors get exponentially higher weights

3. Confidence Calculation:
   - For each label: sum of weights for that label
   - Total: sum of weights for all k neighbors
   - Confidence = label_weight_sum / total_weight_sum

Why confidence might not be 100% even for identical audio:

1. K-Neighbors Effect:
   - Even if the top match is perfect (distance ≈ 0), k-NN considers k neighbors
   - If k=5 and other neighbors have different labels, confidence is diluted
   - Example: If top match has weight=1000, but 4 other neighbors have weights=10 each,
     confidence = 1000 / (1000 + 40) = 96.15%

2. Prototype Diversity:
   - If there are similar prototypes from different labels in the top-k,
     their weights reduce the confidence for the correct label

3. Feature Variability:
   - Small variations in audio (noise, recording conditions) cause small distance differences
   - This affects which prototypes are in the top-k

Recommendations:

1. Increase k value if you have many prototypes per label
2. Ensure prototypes are well-separated in feature space
3. Consider using distance threshold instead of fixed k
4. Normalize features to similar scales before extraction (see feature scale analysis)
`
}

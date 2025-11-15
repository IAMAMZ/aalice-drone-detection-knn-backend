package drone

// K-Nearest Neighbors Classifier for Drone Detection
//
// This package implements a prototype-based classifier for identifying drone acoustic signatures.
// The system uses a k-nearest neighbors (KNN) approach with normalized feature vectors.
//
// How It Works:
//
// 1. Prototype Storage:
//    - Each prototype is a labeled audio sample with extracted features
//    - Features are normalized to unit length (L2 normalization)
//    - Prototypes are stored with metadata (label, category, description, etc.)
//
// 2. Classification Process:
//    - Input audio is processed to extract the same 11 features as prototypes
//    - Feature vector is normalized to unit length
//    - Euclidean distance is computed between input and all prototypes
//    - K nearest prototypes are selected (default k=5)
//
// 3. Prediction Aggregation:
//    - For each label, aggregate weights from matching prototypes
//    - Weight = 1 / (distance + epsilon) - closer prototypes have higher weight
//    - Confidence = sum of weights for label / total weight of all k neighbors
//    - Average distance and support count are also computed per label
//
// 4. Drone Detection:
//    - DetermineDroneLikely() checks if top prediction:
//      * Has confidence >= threshold (default 0.55)
//      * Is not categorized as "noise"
//
// The classifier supports dynamic prototype addition, allowing the system to learn new
// drone types without retraining. Prototypes can be uploaded via the web interface.

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"song-recognition/utils"
)

const harmonicFeatureCount = 3

// Feature weights for PANNS embeddings (2048 dimensions)
// All set to 1.0 for equal weighting across all learned features
var featureWeights []float64

func init() {
	// Initialize 2048 feature weights for PANNS embeddings
	featureWeights = make([]float64, 2048)
	for i := range featureWeights {
		featureWeights[i] = 1.0
	}
}

// Classifier performs k-nearest prototype lookups in the feature space.
type Classifier struct {
	mu            sync.RWMutex
	prototypes    []Prototype
	k             int
	usingExample  bool
	modelPath     string
	labelCategory map[string]string
	labelMetadata map[string]map[string]string
	featureScaler *FeatureScaler // Standardizes features before distance calculation
}

type distancePair struct {
	index    int
	distance float64
}

// NewClassifierFromFile loads prototype embeddings from the supplied path.
func NewClassifierFromFile(path string, k int) (*Classifier, error) {
	if k <= 0 {
		return nil, fmt.Errorf("invalid neighbour count: %d", k)
	}

	resolvedPath := filepath.Clean(path)
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		// if the primary file is missing, attempt to fallback to `.example.json`
		// e.g., "prototypes.json" -> "prototypes.example.json"
		ext := filepath.Ext(resolvedPath)
		base := strings.TrimSuffix(resolvedPath, ext)
		fallbackPath := base + ".example" + ext
		data, err = os.ReadFile(fallbackPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load prototypes (%s): %w", resolvedPath, err)
		}
		rcLogger := utils.GetLogger()
		rcLogger.Warn("falling back to example prototypes", "path", fallbackPath)
		resolvedPath = fallbackPath
	}

	var prototypes []Prototype
	if err := json.Unmarshal(data, &prototypes); err != nil {
		return nil, fmt.Errorf("unable to parse prototypes: %w", err)
	}
	labelCategory := make(map[string]string)
	labelMetadata := make(map[string]map[string]string)
	expectedFeatureCount := len(featureWeights)
	rcLogger := utils.GetLogger()
	zeroHarmonicCount := 0

	if len(prototypes) == 0 {
		rcLogger.Warn("no prototypes loaded; classifier will start empty", "path", resolvedPath)
	} else {
		for idx := range prototypes {
			proto := prototypes[idx]
			if len(proto.Features) == 0 {
				return nil, fmt.Errorf("prototype %s has no features", proto.ID)
			}
			if proto.Label == "" {
				return nil, fmt.Errorf("prototype %s missing label", proto.ID)
			}

			// Require the current feature dimension
			if len(proto.Features) != expectedFeatureCount {
				return nil, fmt.Errorf("prototype %s has %d features, expected %d (prototypes must be regenerated)",
					proto.ID, len(proto.Features), expectedFeatureCount)
			}

			// Check if harmonic features (last harmonicFeatureCount) are zeros
			if len(proto.Features) < harmonicFeatureCount {
				return nil, fmt.Errorf("prototype %s has insufficient harmonic features", proto.ID)
			}
			harmonicStart := len(proto.Features) - harmonicFeatureCount
			harmonicSlice := proto.Features[harmonicStart:]
			allZero := true
			for _, value := range harmonicSlice {
				if value != 0 {
					allZero = false
					break
				}
			}
			if allZero {
				zeroHarmonicCount++
				rcLogger.Warn("prototype has zero harmonic features (needs regeneration)",
					"id", proto.ID,
					"label", proto.Label)
			}

			// Don't normalize yet - we need to compute the scaler first
			prototypes[idx].Features = proto.Features
			if _, ok := labelCategory[proto.Label]; !ok {
				labelCategory[proto.Label] = proto.Category
			}
			if _, ok := labelMetadata[proto.Label]; !ok {
				labelMetadata[proto.Label] = map[string]string{}
			}
			for k, v := range proto.Metadata {
				labelMetadata[proto.Label][k] = v
			}
		}
	}

	// CRITICAL FIX: Compute feature scaler from raw (unscaled) prototypes
	// This prevents one feature dimension (like spectral crest factor) from dominating
	// However, skip scaling for PANNS embeddings (2048 dims) - they're already properly scaled
	var featureScaler *FeatureScaler
	if len(prototypes) > 0 {
		isPANNS := len(prototypes[0].Features) == 2048

		if isPANNS {
			rcLogger.Info("detected PANNS embeddings, skipping feature scaling",
				"prototype_count", len(prototypes),
				"feature_dimensions", len(prototypes[0].Features))
		} else {
			var err error
			featureScaler, err = NewFeatureScalerFromPrototypes(prototypes)
			if err != nil {
				rcLogger.Warn("failed to create feature scaler, using raw features", "error", err)
			} else {
				// Apply scaling and normalization to all prototypes
				for idx := range prototypes {
					scaled := featureScaler.Transform(prototypes[idx].Features)
					NormaliseVectorInPlace(scaled)
					prototypes[idx].Features = scaled
				}
				rcLogger.Info("feature scaler initialized successfully",
					"prototype_count", len(prototypes),
					"feature_dimensions", len(featureScaler.Mean))
			}
		}
	}

	usingExample := strings.HasSuffix(resolvedPath, ".example")

	// Store the actual model path (not the example fallback)
	modelPath := resolvedPath
	if usingExample {
		// If using example, save to the non-example path
		modelPath = strings.TrimSuffix(resolvedPath, ".example")
	}

	if len(prototypes) > 0 && k > len(prototypes) {
		k = len(prototypes)
	}

	if zeroHarmonicCount > 0 {
		rcLogger.Warn("prototypes have invalid harmonic features",
			"count", zeroHarmonicCount,
			"total", len(prototypes),
			"message", "Detection accuracy will be poor. Regenerate prototypes with new feature extraction.")
	}

	return &Classifier{
		prototypes:    prototypes,
		k:             k,
		usingExample:  usingExample,
		modelPath:     modelPath,
		labelCategory: labelCategory,
		labelMetadata: labelMetadata,
		featureScaler: featureScaler,
	}, nil
}

func (c *Classifier) snapshot() (int, []Prototype, map[string]string, map[string]map[string]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	k := c.k
	usingExample := c.usingExample

	prototypes := make([]Prototype, len(c.prototypes))
	for idx, proto := range c.prototypes {
		protoCopy := proto
		if proto.Metadata != nil {
			metaCopy := make(map[string]string, len(proto.Metadata))
			for key, value := range proto.Metadata {
				metaCopy[key] = value
			}
			protoCopy.Metadata = metaCopy
		}
		featuresCopy := make([]float64, len(proto.Features))
		copy(featuresCopy, proto.Features)
		protoCopy.Features = featuresCopy
		prototypes[idx] = protoCopy
	}

	labelCategory := make(map[string]string, len(c.labelCategory))
	for label, category := range c.labelCategory {
		labelCategory[label] = category
	}

	labelMetadata := make(map[string]map[string]string, len(c.labelMetadata))
	for label, meta := range c.labelMetadata {
		if meta == nil {
			continue
		}
		metaCopy := make(map[string]string, len(meta))
		for key, value := range meta {
			metaCopy[key] = value
		}
		labelMetadata[label] = metaCopy
	}

	return k, prototypes, labelCategory, labelMetadata, usingExample
}

func (c *Classifier) AddPrototype(proto Prototype) (Prototype, error) {
	if len(proto.Features) == 0 {
		return Prototype{}, errors.New("prototype has no features")
	}

	features := append([]float64(nil), proto.Features...)

	// Apply feature scaling if available
	c.mu.RLock()
	scaler := c.featureScaler
	c.mu.RUnlock()

	if scaler != nil {
		features = scaler.Transform(features)
	}

	NormaliseVectorInPlace(features)
	proto.Features = features

	metadataCopy := make(map[string]string, len(proto.Metadata))
	for key, value := range proto.Metadata {
		metadataCopy[key] = value
	}
	if proto.Description != "" {
		if _, ok := metadataCopy["description"]; !ok {
			metadataCopy["description"] = proto.Description
		}
	}
	proto.Metadata = metadataCopy

	c.mu.Lock()
	defer c.mu.Unlock()

	c.prototypes = append(c.prototypes, proto)
	if proto.Label != "" {
		if proto.Category != "" {
			c.labelCategory[proto.Label] = proto.Category
		}
		if _, ok := c.labelMetadata[proto.Label]; !ok {
			c.labelMetadata[proto.Label] = map[string]string{}
		}
		for key, value := range proto.Metadata {
			c.labelMetadata[proto.Label][key] = value
		}
	}
	// once custom prototypes are added, mark underlying set as bespoke
	c.usingExample = false

	return proto, nil
}

// SavePrototypesToFile persists all prototypes to the model file.
// This ensures uploaded prototypes survive server restarts.
func (c *Classifier) SavePrototypesToFile() error {
	if c.modelPath == "" {
		return errors.New("model path not set")
	}

	// Get a snapshot of all prototypes
	_, prototypes, _, _, _ := c.snapshot()

	// Ensure directory exists
	dir := filepath.Dir(c.modelPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first, then rename for atomic operation
	tempPath := c.modelPath + ".tmp"
	data, err := json.MarshalIndent(prototypes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prototypes: %w", err)
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write prototypes: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, c.modelPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Mark as no longer using example
	c.mu.Lock()
	c.usingExample = false
	c.mu.Unlock()

	return nil
}

// Stats returns summary metadata about the loaded prototype set.
func (c *Classifier) Stats() ModelStats {
	_, prototypes, _, _, usingExample := c.snapshot()

	labelBuckets := make(map[string]int)
	entries := make(map[string]ModelLabelStat)

	for _, proto := range prototypes {
		labelBuckets[proto.Label]++
		entries[proto.Label] = ModelLabelStat{
			Label:      proto.Label,
			Category:   proto.Category,
			Prototypes: labelBuckets[proto.Label],
		}
	}

	labels := make([]ModelLabelStat, 0, len(entries))
	for _, stat := range entries {
		labels = append(labels, stat)
	}
	// keep labels sorted for deterministic responses
	sort.Slice(labels, func(i, j int) bool { return labels[i].Label < labels[j].Label })

	return ModelStats{
		PrototypeCount: len(prototypes),
		LabelCount:     len(labelBuckets),
		Labels:         labels,
		UsingExample:   usingExample,
	}
}

// Predict finds the best prototype matches for a feature vector.
func (c *Classifier) Predict(features []float64) ([]Prediction, error) {
	if len(features) == 0 {
		return nil, errors.New("feature vector is empty")
	}

	// Apply feature scaling to incoming features (critical for correct classification)
	// However, skip scaling for PANNS embeddings (2048 dims) since they're already properly scaled
	c.mu.RLock()
	scaler := c.featureScaler
	c.mu.RUnlock()

	if scaler != nil && len(features) != 2048 {
		// Only scale legacy hand-crafted features, NOT PANNS embeddings
		features = scaler.Transform(features)
		NormaliseVectorInPlace(features)
		log.Printf("[Classifier] Applied scaling to %d-dim features", len(features))
	} else if len(features) == 2048 {
		log.Printf("[Classifier] Skipping scaling for PANNS embeddings (2048 dims)")
	}

	k, prototypes, labelCategory, labelMetadata, _ := c.snapshot()

	if len(prototypes) == 0 {
		return []Prediction{}, nil
	}

	k = c.k
	if len(prototypes) < k {
		k = max(1, len(prototypes))
	}

	// Find the k-nearest prototypes
	distances := make([]distancePair, len(prototypes))
	for i := range prototypes {
		// Cosine similarity returns a value between -1 and 1 (1 is most similar).
		// We convert it to a distance measure (0 is most similar) by subtracting from 1.
		similarity := cosineSimilarity(features, prototypes[i].Features, featureWeights)
		distances[i] = distancePair{index: i, distance: 1 - similarity}
	}
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].distance < distances[j].distance
	})

	labelScores := make(map[string]struct {
		weightSum  float64
		distSum    float64
		count      int
		prototypes []PrototypeScore
	})

	var totalWeight float64
	for idx := 0; idx < len(distances) && idx < k; idx++ {
		neighbor := distances[idx]
		weight := 1.0 / (neighbor.distance + 1e-9) // Add a small epsilon to avoid division by zero

		stats := labelScores[prototypes[neighbor.index].Label]
		stats.weightSum += weight
		stats.distSum += neighbor.distance
		stats.count++
		stats.prototypes = append(stats.prototypes, PrototypeScore{
			ID:       prototypes[neighbor.index].ID,
			Distance: neighbor.distance,
			Weight:   weight,
			Source:   prototypes[neighbor.index].Source,
		})

		labelScores[prototypes[neighbor.index].Label] = stats
		totalWeight += weight
	}

	if totalWeight == 0 {
		return []Prediction{}, nil
	}

	predictions := make([]Prediction, 0, len(labelScores))
	for label, stats := range labelScores {
		labelMeta := labelMetadata[label]
		description := ""
		if labelMeta != nil {
			description = labelMeta["description"]
		}
		confidence := 0.0
		if totalWeight > 0 {
			confidence = stats.weightSum / totalWeight
		}
		avgDist := 0.0
		if stats.count > 0 {
			avgDist = stats.distSum / float64(stats.count)
		}

		entry := Prediction{
			Label:         label,
			Category:      labelCategory[label],
			Type:          derivePredictionType(label, labelCategory[label], labelMeta),
			Description:   description,
			Confidence:    confidence,
			AverageDist:   avgDist,
			Support:       stats.count,
			TopPrototypes: stats.prototypes,
			Metadata:      labelMeta,
		}

		// Extract threat assessment for defense applications
		if labelMeta != nil && labelCategory[label] == "drone" {
			threatAssessment := ExtractThreatAssessment(entry)
			if threatAssessment.ThreatLevel != "" || threatAssessment.RiskCategory != "" {
				entry.ThreatAssessment = &threatAssessment
			}
		}

		predictions = append(predictions, entry)
	}

	sort.Slice(predictions, func(i, j int) bool {
		if math.Abs(predictions[i].Confidence-predictions[j].Confidence) > 1e-9 {
			return predictions[i].Confidence > predictions[j].Confidence
		}
		return predictions[i].AverageDist < predictions[j].AverageDist
	})

	return predictions, nil
}

// PredictWithSlidingWindows analyses raw samples using overlapping windows and aggregates
// the per-window predictions into a consolidated decision.
func (c *Classifier) PredictWithSlidingWindows(samples []float64, sampleRate int, windowSeconds float64, overlapSeconds float64) ([]Prediction, []WindowPrediction, error) {
	if len(samples) == 0 {
		return nil, nil, errors.New("audio sample is empty")
	}
	if sampleRate <= 0 {
		return nil, nil, errors.New("invalid sample rate")
	}

	if windowSeconds <= 0 {
		windowSeconds = 3.0
	}
	if overlapSeconds < 0 {
		overlapSeconds = 0
	}

	windowSize := int(windowSeconds * float64(sampleRate))
	if windowSize <= 0 {
		windowSize = sampleRate * 3
	}
	if windowSize > len(samples) {
		windowSize = len(samples)
	}

	const minWindowSize = 1024
	if windowSize < minWindowSize {
		windowSize = minWindowSize
		if windowSize > len(samples) {
			windowSize = len(samples)
		}
	}

	overlapSamples := int(overlapSeconds * float64(sampleRate))
	hopSize := windowSize - overlapSamples
	if hopSize <= 0 {
		hopSize = windowSize / 2
		if hopSize == 0 {
			hopSize = 1
		}
	}
	if hopSize > windowSize {
		hopSize = windowSize
	}

	type aggregatedLabelStats struct {
		weightSum       float64
		distWeightedSum float64
		support         int
		category        string
		description     string
		metadata        map[string]string
		topPrototypes   []PrototypeScore
	}

	labelAggregates := make(map[string]*aggregatedLabelStats)
	var windowPredictions []WindowPrediction
	totalWeight := 0.0

	for start := 0; start < len(samples); {
		end := start + windowSize
		if end > len(samples) {
			end = len(samples)
		}

		windowSamples := samples[start:end]
		if len(windowSamples) < 256 {
			break
		}

		features, err := ExtractFeatureVector(windowSamples, sampleRate)
		if err != nil {
			return nil, nil, err
		}
		// Don't normalize here - Predict() will handle scaling and normalization

		windowPreds, err := c.Predict(features)
		if err != nil {
			return nil, nil, err
		}

		windowPredictions = append(windowPredictions, WindowPrediction{
			Index:       len(windowPredictions),
			Start:       float64(start) / float64(sampleRate),
			End:         float64(end) / float64(sampleRate),
			Predictions: windowPreds,
		})

		for _, pred := range windowPreds {
			if pred.Confidence <= 0 {
				continue
			}
			stats, ok := labelAggregates[pred.Label]
			if !ok {
				stats = &aggregatedLabelStats{}
				labelAggregates[pred.Label] = stats
			}

			stats.weightSum += pred.Confidence
			stats.distWeightedSum += pred.AverageDist * pred.Confidence
			stats.support += pred.Support
			if stats.category == "" {
				stats.category = pred.Category
			}
			if stats.description == "" && pred.Description != "" {
				stats.description = pred.Description
			}
			if stats.metadata == nil && pred.Metadata != nil {
				stats.metadata = copyMetadata(pred.Metadata)
			}
			stats.topPrototypes = mergePrototypeScores(stats.topPrototypes, pred.TopPrototypes, 5)

			totalWeight += pred.Confidence
		}

		if end == len(samples) {
			break
		}
		start += hopSize
		if start >= len(samples) {
			break
		}
	}

	if len(windowPredictions) == 0 {
		return nil, nil, errors.New("no analysis windows produced predictions")
	}

	if len(labelAggregates) == 0 || totalWeight == 0 {
		return []Prediction{}, windowPredictions, nil
	}

	predictions := make([]Prediction, 0, len(labelAggregates))
	for label, stats := range labelAggregates {
		confidence := stats.weightSum / totalWeight
		avgDist := 0.0
		if stats.weightSum > 0 {
			avgDist = stats.distWeightedSum / stats.weightSum
		}

		labelMeta := stats.metadata
		entry := Prediction{
			Label:         label,
			Category:      stats.category,
			Type:          derivePredictionType(label, stats.category, labelMeta),
			Description:   stats.description,
			Confidence:    confidence,
			AverageDist:   avgDist,
			Support:       stats.support,
			TopPrototypes: stats.topPrototypes,
			Metadata:      labelMeta,
		}

		if labelMeta != nil && strings.EqualFold(stats.category, "drone") {
			threatAssessment := ExtractThreatAssessment(entry)
			if threatAssessment.ThreatLevel != "" || threatAssessment.RiskCategory != "" {
				entry.ThreatAssessment = &threatAssessment
			}
		}

		predictions = append(predictions, entry)
	}

	sort.Slice(predictions, func(i, j int) bool {
		if math.Abs(predictions[i].Confidence-predictions[j].Confidence) > 1e-9 {
			return predictions[i].Confidence > predictions[j].Confidence
		}
		return predictions[i].AverageDist < predictions[j].AverageDist
	})

	return predictions, windowPredictions, nil
}

type neighborMatch struct {
	prototype Prototype
	distance  float64
	weight    float64
}

func rankNeighbors(features []float64, prototypes []Prototype) []neighborMatch {
	neighbors := make([]neighborMatch, 0, len(prototypes))
	for _, proto := range prototypes {
		dist := weightedDistance(features, proto.Features)
		weight := 1.0 / (dist + 1e-9)
		neighbors = append(neighbors, neighborMatch{prototype: proto, distance: dist, weight: weight})
	}

	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].distance < neighbors[j].distance
	})

	return neighbors
}

func weightedDistance(a, b []float64) float64 {
	minLength := len(a)
	if len(b) < minLength {
		minLength = len(b)
	}
	if minLength == 0 {
		return math.MaxFloat64
	}

	var sum float64
	for i := 0; i < minLength; i++ {
		diff := a[i] - b[i]
		weight := 1.0
		if i < len(featureWeights) {
			weight = featureWeights[i]
		}
		sum += weight * diff * diff
	}

	// Account for any remaining dimensions if vectors differ in length
	for i := minLength; i < len(a); i++ {
		weight := 1.0
		if i < len(featureWeights) {
			weight = featureWeights[i]
		}
		sum += weight * a[i] * a[i]
	}
	for i := minLength; i < len(b); i++ {
		weight := 1.0
		if i < len(featureWeights) {
			weight = featureWeights[i]
		}
		sum += weight * b[i] * b[i]
	}

	// Penalize comparisons where harmonic features (last harmonicFeatureCount) mismatch zeros
	zeroFeaturePenalty := 0.0
	if len(a) >= harmonicFeatureCount && len(b) >= harmonicFeatureCount {
		for i := 0; i < harmonicFeatureCount; i++ {
			idxA := len(a) - harmonicFeatureCount + i
			idxB := len(b) - harmonicFeatureCount + i
			if idxA >= len(a) || idxB >= len(b) {
				continue
			}
			if (a[idxA] == 0 && b[idxB] != 0) || (a[idxA] != 0 && b[idxB] == 0) {
				zeroFeaturePenalty += 0.5
			}
		}
	}

	return math.Sqrt(sum) + zeroFeaturePenalty
}

func copyMetadata(meta map[string]string) map[string]string {
	if meta == nil {
		return nil
	}
	clone := make(map[string]string, len(meta))
	for key, value := range meta {
		clone[key] = value
	}
	return clone
}

func mergePrototypeScores(existing []PrototypeScore, additional []PrototypeScore, limit int) []PrototypeScore {
	if len(existing) == 0 && len(additional) == 0 {
		return nil
	}

	combined := make(map[string]PrototypeScore, len(existing)+len(additional))
	for _, score := range existing {
		combined[score.ID] = score
	}
	for _, score := range additional {
		if current, ok := combined[score.ID]; ok {
			if score.Weight > current.Weight || (math.Abs(score.Weight-current.Weight) < 1e-9 && score.Distance < current.Distance) {
				combined[score.ID] = score
			}
		} else {
			combined[score.ID] = score
		}
	}

	result := make([]PrototypeScore, 0, len(combined))
	for _, score := range combined {
		result = append(result, score)
	}

	sort.Slice(result, func(i, j int) bool {
		if math.Abs(result[i].Weight-result[j].Weight) > 1e-9 {
			return result[i].Weight > result[j].Weight
		}
		return result[i].Distance < result[j].Distance
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result
}

func derivePredictionType(label, category string, metadata map[string]string) string {
	if metadata != nil {
		if value := strings.TrimSpace(metadata["model"]); value != "" {
			return value
		}
		if value := strings.TrimSpace(metadata["type"]); value != "" {
			return value
		}
		if value := strings.TrimSpace(metadata["description"]); value != "" {
			return value
		}
	}

	if category != "" {
		return fmt.Sprintf("%s (%s)", label, category)
	}

	return label
}

// DetermineDroneLikely interprets the prediction list to understand whether the
// analysed audio likely corresponds to a drone target.
// Uses adaptive threshold based on SNR if provided.
func DetermineDroneLikely(predictions []Prediction, threshold float64) bool {
	return DetermineDroneLikelyWithSNR(predictions, threshold, 0.0)
}

// DetermineDroneLikelyWithSNR uses SNR-adjusted threshold for better noise handling
func DetermineDroneLikelyWithSNR(predictions []Prediction, baseThreshold float64, snrDb float64) bool {
	if len(predictions) == 0 {
		return false
	}

	best := predictions[0]
	if strings.EqualFold(best.Category, "noise") {
		return false
	}

	// Use adaptive threshold if SNR is provided
	threshold := baseThreshold
	if snrDb != 0.0 {
		threshold = AdaptiveThreshold(baseThreshold, snrDb)
	}

	return best.Confidence >= threshold
}

// cosineSimilarity computes the weighted cosine similarity between two vectors.
// A higher value indicates greater similarity.
func cosineSimilarity(a, b, weights []float64) float64 {
	var dotProduct, normA, normB float64
	limit := min(len(a), len(b))

	for i := 0; i < limit; i++ {
		weight := 1.0
		if i < len(weights) {
			weight = weights[i]
		}
		weightedA := a[i] * weight
		weightedB := b[i] * weight
		dotProduct += weightedA * weightedB
		normA += weightedA * weightedA
		normB += weightedB * weightedB
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

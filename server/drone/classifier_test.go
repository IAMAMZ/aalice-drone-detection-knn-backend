package drone

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrototypesJSONStructure(t *testing.T) {
	t.Parallel()

	path := prototypesFilePath(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	var protos []Prototype
	if err := json.Unmarshal(raw, &protos); err != nil {
		t.Fatalf("failed to parse %s: %v", path, err)
	}
	if len(protos) == 0 {
		t.Fatalf("no prototypes defined in %s", path)
	}

	for _, proto := range protos {
		if len(proto.Features) != len(featureWeights) {
			t.Fatalf("prototype %s has %d features (expected %d)", proto.ID, len(proto.Features), len(featureWeights))
		}
		norm := vectorNorm(proto.Features)
		if math.Abs(norm-1.0) > 1e-6 {
			t.Errorf("prototype %s is not normalised (||v||=%f)", proto.ID, norm)
		}
	}

	if len(protos) > 1 {
		var minSim, maxSim, sumSim float64
		minSim = 1
		maxSim = -1
		pairs := 0
		for i := 0; i < len(protos); i++ {
			for j := i + 1; j < len(protos); j++ {
				sim := cosineSimilarity(protos[i].Features, protos[j].Features, featureWeights)
				if sim < minSim {
					minSim = sim
				}
				if sim > maxSim {
					maxSim = sim
				}
				sumSim += sim
				pairs++
			}
		}
		avg := sumSim / float64(pairs)
		t.Logf("prototype cosine similarity stats: min=%.6f max=%.6f avg=%.6f (higher => more alike)", minSim, maxSim, avg)
	}
}

func TestClassifierPredictPrefersMajorityLabel(t *testing.T) {
	t.Parallel()

	protos := []Prototype{
		newSyntheticPrototype("alpha", "alpha_1", map[int]float64{0: 1.0}),
		newSyntheticPrototype("alpha", "alpha_2", map[int]float64{0: 0.8, 1: 0.2}),
		newSyntheticPrototype("beta", "beta_1", map[int]float64{8: 1.0}),
	}

	classifier := newTestClassifier(protos, 3)
	target := featureVector(map[int]float64{0: 1.0})

	predictions, err := classifier.Predict(target)
	if err != nil {
		t.Fatalf("Predict returned error: %v", err)
	}
	if len(predictions) == 0 {
		t.Fatalf("no predictions returned")
	}
	if predictions[0].Label != "alpha" {
		t.Fatalf("expected alpha as top prediction, got %s", predictions[0].Label)
	}
	if predictions[0].Support != 2 {
		t.Fatalf("expected support=2 for alpha, got %d", predictions[0].Support)
	}
	if predictions[0].Confidence <= 0.5 {
		t.Fatalf("expected confidence > 0.5 for alpha, got %.3f", predictions[0].Confidence)
	}
}

func TestClassifierPredictRespondsToFeatureShift(t *testing.T) {
	t.Parallel()

	protos := []Prototype{
		newSyntheticPrototype("alpha", "alpha_1", map[int]float64{0: 1.0}),
		newSyntheticPrototype("alpha", "alpha_2", map[int]float64{0: 0.8, 1: 0.2}),
		newSyntheticPrototype("beta", "beta_1", map[int]float64{10: 1.0}),
	}

	classifier := newTestClassifier(protos, 3)
	target := featureVector(map[int]float64{10: 1.0})

	predictions, err := classifier.Predict(target)
	if err != nil {
		t.Fatalf("Predict returned error: %v", err)
	}
	if len(predictions) == 0 {
		t.Fatalf("no predictions returned")
	}
	if predictions[0].Label != "beta" {
		t.Fatalf("expected beta as top prediction, got %s", predictions[0].Label)
	}
	if predictions[0].Confidence < 0.9 {
		t.Fatalf("expected beta confidence >= 0.9, got %.3f", predictions[0].Confidence)
	}
}

func featureVector(peaks map[int]float64) []float64 {
	vec := make([]float64, len(featureWeights))
	for idx, value := range peaks {
		if idx >= len(vec) {
			continue
		}
		vec[idx] = value
	}
	NormaliseVectorInPlace(vec)
	return vec
}

func newSyntheticPrototype(label, id string, peaks map[int]float64) Prototype {
	return Prototype{
		ID:       id,
		Label:    label,
		Category: "drone",
		Features: featureVector(peaks),
	}
}

func newTestClassifier(protos []Prototype, k int) *Classifier {
	labelCategory := make(map[string]string)
	labelMetadata := make(map[string]map[string]string)
	copies := make([]Prototype, len(protos))
	for i, proto := range protos {
		labelCategory[proto.Label] = proto.Category
		features := make([]float64, len(proto.Features))
		copy(features, proto.Features)
		proto.Features = features
		copies[i] = proto
	}
	return &Classifier{
		prototypes:    copies,
		k:             k,
		labelCategory: labelCategory,
		labelMetadata: labelMetadata,
	}
}

func prototypesFilePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine caller information")
	}
	return filepath.Join(filepath.Dir(file), "prototypes.json")
}

func vectorNorm(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += v * v
	}
	return math.Sqrt(sum)
}

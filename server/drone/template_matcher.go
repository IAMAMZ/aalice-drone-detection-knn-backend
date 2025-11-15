package drone

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Template captures a single reference embedding derived from a labelled audio sample.
type Template struct {
	Label    string    `json:"label"`
	Source   string    `json:"source"`
	Features []float64 `json:"features"`
}

// TemplateMatcher performs cosine-similarity lookups against a small template bank.
type TemplateMatcher struct {
	templates []Template
	threshold float64
}

// TemplateCount exposes number of loaded templates.
func (tm *TemplateMatcher) TemplateCount() int {
	if tm == nil {
		return 0
	}
	return len(tm.templates)
}

// NewTemplateMatcherFromFile loads template embeddings from disk.
func NewTemplateMatcherFromFile(path string, threshold float64) (*TemplateMatcher, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	var templates []Template
	if err := json.Unmarshal(data, &templates); err != nil {
		return nil, fmt.Errorf("failed to parse template file: %w", err)
	}

	if len(templates) == 0 {
		return nil, fmt.Errorf("template file %s contained no entries", path)
	}

	for idx := range templates {
		if len(templates[idx].Features) != len(featureWeights) {
			return nil, fmt.Errorf("template %s has %d features, expected %d",
				templates[idx].Label, len(templates[idx].Features), len(featureWeights))
		}
		NormaliseVectorInPlace(templates[idx].Features)
	}

	return &TemplateMatcher{
		templates: templates,
		threshold: clamp01(threshold),
	}, nil
}

// Predict emits ranked predictions based on cosine similarity between
// the analysed feature vector and each stored template.
func (tm *TemplateMatcher) Predict(features []float64) []Prediction {
	if tm == nil || len(features) == 0 {
		return nil
	}

	results := make([]Prediction, 0, len(tm.templates))
	for _, tpl := range tm.templates {
		similarity := cosineSimilarity(features, tpl.Features, featureWeights)
		confidence := similarityToConfidence(similarity)
		if tm.threshold > 0 && confidence < tm.threshold {
			continue
		}

		results = append(results, Prediction{
			Label:       tpl.Label,
			Category:    "template",
			Type:        tpl.Label,
			Description: fmt.Sprintf("template:%s", tpl.Source),
			Confidence:  confidence,
			AverageDist: 1 - similarity,
			Support:     1,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Confidence != results[j].Confidence {
			return results[i].Confidence > results[j].Confidence
		}
		return results[i].AverageDist < results[j].AverageDist
	})

	return results
}

// MergePredictions merges template predictions into the canonical list,
// keeping the higher-confidence entry when labels overlap.
func MergePredictions(base []Prediction, additions []Prediction) []Prediction {
	if len(additions) == 0 {
		return base
	}

	index := make(map[string]Prediction, len(base)+len(additions))
	for _, pred := range base {
		index[strings.ToLower(pred.Label)] = pred
	}

	for _, pred := range additions {
		key := strings.ToLower(pred.Label)
		if existing, ok := index[key]; ok {
			if pred.Confidence > existing.Confidence {
				index[key] = pred
			}
		} else {
			index[key] = pred
		}
	}

	merged := make([]Prediction, 0, len(index))
	for _, pred := range index {
		merged = append(merged, pred)
	}

	sort.Slice(merged, func(i, j int) bool {
		if merged[i].Confidence != merged[j].Confidence {
			return merged[i].Confidence > merged[j].Confidence
		}
		return merged[i].AverageDist < merged[j].AverageDist
	})

	return merged
}

// BuildTemplatesFromDir ingests every WAV file in the dir and emits template embeddings.
func BuildTemplatesFromDir(dir string) ([]Template, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	templates := make([]Template, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".wav") {
			continue
		}

		label := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		audioPath := filepath.Join(dir, entry.Name())
		proto, err := BuildPrototypeFromPath(audioPath, label, "drone", "", entry.Name(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build template from %s: %w", entry.Name(), err)
		}

		templates = append(templates, Template{
			Label:    label,
			Source:   entry.Name(),
			Features: proto.Features,
		})
	}

	if len(templates) == 0 {
		return nil, fmt.Errorf("no WAV files found in %s", dir)
	}

	return templates, nil
}

// SaveTemplates writes templates to disk.
func SaveTemplates(path string, templates []Template) error {
	if len(templates) == 0 {
		return fmt.Errorf("no templates to save")
	}

	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal templates: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func similarityToConfidence(sim float64) float64 {
	// sim ranges [-1,1]; convert to [0,1]
	conf := (sim + 1) / 2
	if conf < 0 {
		return 0
	}
	if conf > 1 {
		return 1
	}
	return conf
}

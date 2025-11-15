package drone

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"song-recognition/utils"
	"song-recognition/wav"
)

// BuildPrototypeFromPath ingests an audio asset, normalises it and emits a Prototype.
func BuildPrototypeFromPath(path string, label string, category string, description string, source string, metadata map[string]string) (Prototype, error) {
	if label == "" {
		return Prototype{}, errors.New("label is required")
	}

	if category == "" {
		category = "drone"
	}

	workingPath := path
	var cleanup []string

	convertedPath, err := wav.ConvertToWAV(workingPath, 1)
	if err != nil {
		return Prototype{}, fmt.Errorf("failed to convert audio: %w", err)
	}
	if convertedPath != path {
		cleanup = append(cleanup, convertedPath)
	}
	workingPath = convertedPath

	wavInfo, err := wav.ReadWavInfo(workingPath)
	if err != nil {
		discardTempFiles(cleanup)
		return Prototype{}, fmt.Errorf("failed to read wav info: %w", err)
	}

	samples, err := wav.WavBytesToSamples(wavInfo.Data)
	if err != nil {
		discardTempFiles(cleanup)
		return Prototype{}, fmt.Errorf("failed to decode samples: %w", err)
	}

	// Apply the exact same preprocessing used during live detection to avoid
	// feature drift between prototypes and inference samples.
	preprocessCfg := DefaultPreprocessingConfig()
	processedSamples := PreprocessAudio(samples, wavInfo.SampleRate, preprocessCfg)

	features, err := ExtractFeatureVector(processedSamples, wavInfo.SampleRate)
	if err != nil {
		discardTempFiles(cleanup)
		return Prototype{}, fmt.Errorf("failed to extract features: %w", err)
	}

	// Don't normalize here - let the classifier handle scaling and normalization
	// to ensure consistency with existing prototypes

	metaCopy := make(map[string]string, len(metadata))
	for key, value := range metadata {
		metaCopy[key] = value
	}

	proto := Prototype{
		ID:          buildPrototypeID(label),
		Label:       label,
		Category:    category,
		Description: description,
		Source:      source,
		Features:    features,
		Metadata:    metaCopy,
	}

	discardTempFiles(cleanup)

	return proto, nil
}

func buildPrototypeID(label string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + 32
		case r >= '0' && r <= '9':
			return r
		case r == '_' || r == '-':
			return r
		case r == ' ':
			return '_'
		default:
			return -1
		}
	}, label)

	if safe == "" {
		safe = "prototype"
	}

	return fmt.Sprintf("proto_%s_%08x", safe, utils.GenerateUniqueID())
}

func discardTempFiles(paths []string) {
	for _, file := range paths {
		if file == "" {
			continue
		}
		_ = os.Remove(file)
	}
}

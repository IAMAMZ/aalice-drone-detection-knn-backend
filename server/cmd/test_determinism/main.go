package main

import (
	"fmt"
	"log"
	"math"
	"os"

	"song-recognition/drone"
	"song-recognition/wav"
)

// Test if feature extraction is deterministic
func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <path-to-wav-file>")
	}

	testFile := os.Args[1]
	log.Printf("Testing determinism with: %s\n", testFile)

	// Extract features 5 times from the same file
	const numRuns = 5
	var featureSets [][]float64

	for i := 0; i < numRuns; i++ {
		features, err := extractFeatures(testFile)
		if err != nil {
			log.Fatalf("Run %d failed: %v", i+1, err)
		}
		featureSets = append(featureSets, features)
		log.Printf("Run %d: First 5 features: %.10f, %.10f, %.10f, %.10f, %.10f",
			i+1, features[0], features[1], features[2], features[3], features[4])
	}

	// Check if all runs produced identical results
	fmt.Println("\n=== Determinism Check ===")
	allIdentical := true
	maxDiff := 0.0

	for i := 1; i < numRuns; i++ {
		for j := 0; j < len(featureSets[0]); j++ {
			diff := math.Abs(featureSets[0][j] - featureSets[i][j])
			if diff > maxDiff {
				maxDiff = diff
			}
			if diff > 1e-12 { // Allow tiny floating point errors
				allIdentical = false
				fmt.Printf("❌ Feature %d differs between run 1 and run %d: %.15f vs %.15f (diff: %e)\n",
					j, i+1, featureSets[0][j], featureSets[i][j], diff)
			}
		}
	}

	if allIdentical {
		fmt.Println("✅ All runs produced IDENTICAL features (deterministic)")
		fmt.Printf("   Max difference: %e\n", maxDiff)
	} else {
		fmt.Printf("❌ Feature extraction is NON-DETERMINISTIC (max diff: %e)\n", maxDiff)
		fmt.Println("   This explains why same samples don't get 100% match!")
	}

	// Now test if the same sample matches its own prototype
	fmt.Println("\n=== Self-Match Test ===")
	features1 := featureSets[0]
	features2 := featureSets[1]

	// Simulate what classifier does: normalize
	norm1 := drone.NormaliseVector(features1)
	norm2 := drone.NormaliseVector(features2)

	// Calculate distance
	var dist float64
	for i := 0; i < len(norm1); i++ {
		diff := norm1[i] - norm2[i]
		dist += diff * diff
	}
	dist = math.Sqrt(dist)

	fmt.Printf("Distance between two extractions of same file: %.10f\n", dist)
	if dist < 1e-10 {
		fmt.Println("✅ Perfect match - distance near zero")
	} else if dist < 0.01 {
		fmt.Println("⚠️  Small difference - likely floating point accumulation")
	} else {
		fmt.Println("❌ Large difference - non-deterministic pipeline!")
	}
}

func extractFeatures(filePath string) ([]float64, error) {
	// Convert to WAV
	convertedPath, err := wav.ConvertToWAV(filePath, 1)
	if err != nil {
		return nil, fmt.Errorf("convert to WAV: %w", err)
	}
	defer func() {
		if convertedPath != filePath {
			os.Remove(convertedPath)
		}
	}()

	// Read WAV
	wavInfo, err := wav.ReadWavInfo(convertedPath)
	if err != nil {
		return nil, fmt.Errorf("read WAV info: %w", err)
	}

	// Decode samples
	samples, err := wav.WavBytesToSamples(wavInfo.Data)
	if err != nil {
		return nil, fmt.Errorf("decode samples: %w", err)
	}

	// Preprocess (this is where non-determinism might occur)
	preprocessCfg := drone.DefaultPreprocessingConfig()
	processed := drone.PreprocessAudio(samples, wavInfo.SampleRate, preprocessCfg)

	// Extract features
	features, err := drone.ExtractFeatureVector(processed, wavInfo.SampleRate)
	if err != nil {
		return nil, fmt.Errorf("extract features: %w", err)
	}

	return features, nil
}

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"song-recognition/drone"
	"song-recognition/utils"
	"song-recognition/wav"
)

// Test if a prototype file matches itself with high confidence
func main() {
	// Load the classifier
	modelPath := utils.GetEnv("DRONE_MODEL_PATH", filepath.Join("drone", "prototypes.json"))
	classifier, err := drone.NewClassifierFromFile(modelPath, 3)
	if err != nil {
		log.Fatalf("Failed to load classifier: %v", err)
	}

	// Get list of prototype sources
	stats := classifier.Stats()
	fmt.Printf("Loaded classifier with %d prototypes across %d labels\n\n", stats.PrototypeCount, stats.LabelCount)

	// Try to find the source files used to create prototypes
	protoFiles := []string{
		"frontendrecording/rec_1763188346576532000rfm.wav",
		"frontendrecording/rec_1763188403250396000rfm.wav",
		"frontendrecording/rec_1763188438939066000rfm.wav",
	}

	fmt.Println("=== Testing Self-Match ===")
	fmt.Println("If a file was used to create a prototype, it should match with ~100% confidence\n")

	for _, testFile := range protoFiles {
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			continue
		}

		fmt.Printf("Testing: %s\n", filepath.Base(testFile))

		// Extract features (same as how prototypes were created)
		convertedPath, err := wav.ConvertToWAV(testFile, 1)
		if err != nil {
			log.Printf("  ERROR: %v\n", err)
			continue
		}
		defer os.Remove(convertedPath)

		wavInfo, err := wav.ReadWavInfo(convertedPath)
		if err != nil {
			log.Printf("  ERROR: %v\n", err)
			continue
		}

		samples, err := wav.WavBytesToSamples(wavInfo.Data)
		if err != nil {
			log.Printf("  ERROR: %v\n", err)
			continue
		}

		// Apply same preprocessing as prototypes
		preprocessCfg := drone.DefaultPreprocessingConfig()
		processed := drone.PreprocessAudio(samples, wavInfo.SampleRate, preprocessCfg)

		// Extract features (raw, not scaled/normalized - classifier will do that)
		features, err := drone.ExtractFeatureVector(processed, wavInfo.SampleRate)
		if err != nil {
			log.Printf("  ERROR: %v\n", err)
			continue
		}

		// Classify
		predictions, err := classifier.Predict(features)
		if err != nil {
			log.Printf("  ERROR: %v\n", err)
			continue
		}

		if len(predictions) == 0 {
			fmt.Println("  ❌ No predictions returned!")
			continue
		}

		// Show top 3 predictions
		for i, pred := range predictions {
			if i >= 3 {
				break
			}
			emoji := "  "
			if i == 0 && pred.Confidence >= 0.90 {
				emoji = "✅"
			} else if i == 0 && pred.Confidence >= 0.70 {
				emoji = "⚠️"
			} else if i == 0 {
				emoji = "❌"
			}
			fmt.Printf("  %s #%d: %s (%.1f%% confidence, avgDist=%.4f, support=%d)\n",
				emoji, i+1, pred.Label, pred.Confidence*100, pred.AverageDist, pred.Support)
		}
		fmt.Println()
	}

	fmt.Println("\n=== Expected Results ===")
	fmt.Println("✅ 90-100% confidence: Perfect (same file as prototype)")
	fmt.Println("⚠️  70-90% confidence: Good (similar but not identical)")
	fmt.Println("❌ <70% confidence: Poor (should be much higher for same file)")
	fmt.Println("\nIf you see low confidence for files that should match, the issue is:")
	fmt.Println("1. Feature scaler computed from too few samples (only 6 prototypes)")
	fmt.Println("2. High variance in features makes scaling unstable")
	fmt.Println("3. Need more diverse training data to compute stable mean/stddev")
}

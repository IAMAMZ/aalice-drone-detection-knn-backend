package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"song-recognition/drone"
	"song-recognition/utils"
	"song-recognition/wav"
)

// Test if the EXACT files used to create prototypes match with 100% confidence
func main() {
	// Load the classifier
	modelPath := utils.GetEnv("DRONE_MODEL_PATH", filepath.Join("drone", "prototypes.json"))

	// First, read the prototypes JSON to get source files
	data, err := os.ReadFile(modelPath)
	if err != nil {
		log.Fatalf("Failed to read prototypes: %v", err)
	}

	var prototypes []drone.Prototype
	if err := json.Unmarshal(data, &prototypes); err != nil {
		log.Fatalf("Failed to parse prototypes: %v", err)
	}

	fmt.Printf("=== Prototype Sources ===\n")
	for i, proto := range prototypes {
		fmt.Printf("%d. %s (label: %s) from: %s\n", i+1, proto.ID, proto.Label, proto.Source)
	}

	// Load classifier
	classifier, err := drone.NewClassifierFromFile(modelPath, 3)
	if err != nil {
		log.Fatalf("Failed to load classifier: %v", err)
	}

	fmt.Printf("\n=== Testing EXACT Source Files ===\n")
	fmt.Println("These files were used to create prototypes, so they should match with very high confidence\n")

	tested := make(map[string]bool)
	for _, proto := range prototypes {
		if proto.Source == "" {
			continue
		}

		// Skip if already tested
		if tested[proto.Source] {
			continue
		}
		tested[proto.Source] = true

		if _, err := os.Stat(proto.Source); os.IsNotExist(err) {
			fmt.Printf("⚠️  File not found: %s\n", proto.Source)
			continue
		}

		fmt.Printf("Testing: %s (expected label: %s)\n", filepath.Base(proto.Source), proto.Label)

		// Extract features EXACTLY as prototypes were created
		convertedPath, err := wav.ConvertToWAV(proto.Source, 1)
		if err != nil {
			log.Printf("  ERROR: %v\n", err)
			continue
		}
		defer func() {
			if convertedPath != proto.Source {
				os.Remove(convertedPath)
			}
		}()

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

		// Same preprocessing as BuildPrototypeFromPath
		preprocessCfg := drone.DefaultPreprocessingConfig()
		processed := drone.PreprocessAudio(samples, wavInfo.SampleRate, preprocessCfg)

		// Extract features (raw - classifier scales them)
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

		// Check if top prediction matches expected label
		topPred := predictions[0]
		expectedMatch := topPred.Label == proto.Label

		emoji := "❌"
		if expectedMatch && topPred.Confidence >= 0.95 {
			emoji = "✅"
		} else if expectedMatch && topPred.Confidence >= 0.80 {
			emoji = "⚠️"
		}

		fmt.Printf("  %s Top: %s (%.1f%% confidence, avgDist=%.4f)\n",
			emoji, topPred.Label, topPred.Confidence*100, topPred.AverageDist)

		// Show all predictions
		fmt.Println("  All predictions:")
		for i, pred := range predictions {
			fmt.Printf("    #%d: %s (%.1f%%, dist=%.4f, support=%d)\n",
				i+1, pred.Label, pred.Confidence*100, pred.AverageDist, pred.Support)
		}

		// Show top matching prototypes
		if len(topPred.TopPrototypes) > 0 {
			fmt.Println("  Closest prototypes:")
			for i, ps := range topPred.TopPrototypes {
				if i >= 3 {
					break
				}
				matchedProto := findProtoByID(prototypes, ps.ID)
				fmt.Printf("    - %s (dist=%.4f, weight=%.2f) from %s\n",
					ps.ID, ps.Distance, ps.Weight, filepath.Base(matchedProto.Source))
			}
		}
		fmt.Println()
	}

	fmt.Println("\n=== Analysis ===")
	fmt.Println("✅ >95%: Excellent - exact match as expected")
	fmt.Println("⚠️  80-95%: Good - but should be higher for exact file")
	fmt.Println("❌ <80%: PROBLEM - same file should match better!")
	fmt.Println("\nRoot causes of low confidence on exact matches:")
	fmt.Println("1. Only 6 prototypes → unstable feature scaler (mean/stddev unreliable)")
	fmt.Println("2. Feature scaler transforms based on this tiny dataset")
	fmt.Println("3. After z-score scaling, small sample variance causes distortion")
	fmt.Println("4. Solution: Add 20-50 more prototypes OR disable feature scaling")
}

func findProtoByID(prototypes []drone.Prototype, id string) drone.Prototype {
	for _, p := range prototypes {
		if p.ID == id {
			return p
		}
	}
	return drone.Prototype{}
}

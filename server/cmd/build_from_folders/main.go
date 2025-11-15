package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"song-recognition/drone"
)

func main() {
	rootDir := flag.String("dir", "", "Root directory containing subdirectories (e.g., droneA-B/)")
	outputFile := flag.String("out", "drone/prototypes.json", "Output prototypes JSON file")
	defaultCategory := flag.String("category", "drone", "Default category (drone/noise)")
	flag.Parse()

	if *rootDir == "" {
		log.Fatal("Usage: go run . -dir <directory> [-out <file>] [-category drone|noise]\n\n" +
			"Example structure:\n" +
			"  droneA-B/\n" +
			"    DroneA/\n" +
			"      sample1.wav\n" +
			"      sample2.wav\n" +
			"    DroneB/\n" +
			"      sample1.wav\n" +
			"      sample2.wav\n" +
			"    Noise/\n" +
			"      ambient.wav\n" +
			"      silence.wav\n")
	}

	// Discover subdirectories
	subdirs, err := discoverSubdirectories(*rootDir)
	if err != nil {
		log.Fatalf("failed to read directory: %v", err)
	}

	if len(subdirs) == 0 {
		log.Fatalf("no subdirectories found in %s", *rootDir)
	}

	log.Printf("Found %d subdirectories in %s:\n", len(subdirs), *rootDir)
	for _, dir := range subdirs {
		log.Printf("  - %s", filepath.Base(dir))
	}
	log.Println()

	var allPrototypes []drone.Prototype
	stats := make(map[string]int) // label -> count

	// Process each subdirectory
	for _, subdir := range subdirs {
		label := inferLabelFromDirectory(subdir)
		category := inferCategory(label, *defaultCategory)
		
		log.Printf("Processing subdirectory: %s (label: '%s', category: %s)\n", 
			filepath.Base(subdir), label, category)

		files, err := collectWAVFiles(subdir)
		if err != nil {
			log.Printf("  ERROR reading directory: %v\n", err)
			continue
		}

		if len(files) == 0 {
			log.Printf("  WARNING: no WAV files found, skipping\n")
			continue
		}

		log.Printf("  Found %d WAV files\n", len(files))

		// Process each WAV file in this subdirectory
		for i, filePath := range files {
			log.Printf("  [%d/%d] Processing: %s", i+1, len(files), filepath.Base(filePath))

			proto, err := drone.BuildPrototypeFromPath(
				filePath,
				label,
				category,
				fmt.Sprintf("%s from %s", label, filepath.Base(filePath)),
				filePath,
				nil,
			)
			if err != nil {
				log.Printf(" ✗ ERROR: %v\n", err)
				continue
			}

			allPrototypes = append(allPrototypes, proto)
			stats[label]++
			log.Printf(" ✓\n")
		}
		log.Println()
	}

	if len(allPrototypes) == 0 {
		log.Fatalf("no prototypes were created")
	}

	// Write to output file
	outDir := filepath.Dir(*outputFile)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}

	data, err := json.MarshalIndent(allPrototypes, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal prototypes: %v", err)
	}

	if err := os.WriteFile(*outputFile, data, 0644); err != nil {
		log.Fatalf("failed to write output file: %v", err)
	}

	log.Printf("✓ Successfully created %d prototypes in %s\n\n", len(allPrototypes), *outputFile)
	
	// Show statistics
	log.Println("Label distribution:")
	for label, count := range stats {
		log.Printf("  %-20s: %d prototypes\n", label, count)
	}

	// Show category distribution
	categoryCount := make(map[string]int)
	for _, p := range allPrototypes {
		categoryCount[p.Category]++
	}
	log.Println("\nCategory distribution:")
	for category, count := range categoryCount {
		log.Printf("  %-20s: %d prototypes\n", category, count)
	}

	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("Next steps:")
	log.Println("1. Test the classifier:")
	log.Println("   go run ./cmd/train_eval -dir", *rootDir, "-k 5")
	log.Println("2. Start the server:")
	log.Println("   go run . serve -proto http -p 5000")
	log.Println(strings.Repeat("=", 60))
}

func discoverSubdirectories(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	var subdirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			subdirs = append(subdirs, filepath.Join(rootDir, entry.Name()))
		}
	}

	return subdirs, nil
}

func collectWAVFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".wav" || ext == ".mp3" {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

func inferLabelFromDirectory(dirPath string) string {
	base := filepath.Base(dirPath)
	
	// Clean up the name
	label := strings.ToLower(base)
	label = strings.ReplaceAll(label, "_", " ")
	label = strings.ReplaceAll(label, "-", " ")
	label = strings.TrimSpace(label)
	
	return label
}

func inferCategory(label string, defaultCategory string) string {
	labelLower := strings.ToLower(label)
	
	// Auto-detect noise/non-drone samples
	noiseKeywords := []string{"noise", "ambient", "silence", "background", 
		"music", "voice", "speech", "traffic", "nature", "wind", "rain"}
	
	for _, keyword := range noiseKeywords {
		if strings.Contains(labelLower, keyword) {
			return "noise"
		}
	}
	
	return defaultCategory
}


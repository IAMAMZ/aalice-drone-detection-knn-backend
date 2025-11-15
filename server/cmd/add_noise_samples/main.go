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
	noiseDir := flag.String("noise-dir", "", "Directory containing noise/non-drone WAV files")
	existingProtos := flag.String("prototypes", "drone/prototypes.json", "Existing prototypes file")
	outputFile := flag.String("out", "drone/prototypes.json", "Output JSON file")
	flag.Parse()

	if *noiseDir == "" {
		log.Fatal("Usage: go run . -noise-dir <directory> [-prototypes <file>] [-out <file>]")
	}

	// Load existing prototypes
	var existingPrototypes []drone.Prototype
	if data, err := os.ReadFile(*existingProtos); err == nil {
		if err := json.Unmarshal(data, &existingPrototypes); err != nil {
			log.Fatalf("failed to parse existing prototypes: %v", err)
		}
		log.Printf("Loaded %d existing prototypes from %s\n", len(existingPrototypes), *existingProtos)
	} else {
		log.Printf("No existing prototypes found, starting fresh\n")
	}

	// Find noise files
	files, err := collectWAVFiles(*noiseDir)
	if err != nil {
		log.Fatalf("failed to list directory: %v", err)
	}

	if len(files) == 0 {
		log.Fatalf("no WAV files found in %s", *noiseDir)
	}

	log.Printf("Found %d noise WAV files in %s\n", len(files), *noiseDir)
	log.Println("Building noise prototypes...")

	// Add noise prototypes
	noiseCount := 0
	for _, filePath := range files {
		label := inferLabel(filePath)
		log.Printf("Processing: %s (label: %s, category: noise)", filepath.Base(filePath), label)

		proto, err := drone.BuildPrototypeFromPath(
			filePath,
			label,
			"noise", // Category is "noise" not "drone"
			fmt.Sprintf("Noise/non-drone sample: %s", filepath.Base(filePath)),
			filePath,
			nil,
		)
		if err != nil {
			log.Printf("  ERROR: %v", err)
			continue
		}

		existingPrototypes = append(existingPrototypes, proto)
		noiseCount++
		log.Printf("  ✓ Created noise prototype %s", proto.ID)
	}

	if noiseCount == 0 {
		log.Fatalf("no noise prototypes were created")
	}

	// Write combined prototypes
	outDir := filepath.Dir(*outputFile)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}

	data, err := json.MarshalIndent(existingPrototypes, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal prototypes: %v", err)
	}

	if err := os.WriteFile(*outputFile, data, 0644); err != nil {
		log.Fatalf("failed to write output file: %v", err)
	}

	log.Printf("\n✓ Successfully added %d noise prototypes to %s", noiseCount, *outputFile)
	log.Printf("Total prototypes: %d\n", len(existingPrototypes))

	// Show category distribution
	categoryCount := make(map[string]int)
	for _, p := range existingPrototypes {
		categoryCount[p.Category]++
	}

	log.Println("\nCategory distribution:")
	for category, count := range categoryCount {
		log.Printf("  %s: %d", category, count)
	}
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
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".wav") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	return files, nil
}

func inferLabel(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// Clean up the name
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	// Prefix with "noise" if not already there
	if !strings.Contains(name, "noise") && !strings.Contains(name, "ambient") {
		name = "noise " + name
	}

	return name
}

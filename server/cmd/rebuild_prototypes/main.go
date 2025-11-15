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
	inputDir := flag.String("dir", "train_data", "Directory containing WAV files")
	outputFile := flag.String("out", "drone/prototypes.json", "Output JSON file")
	category := flag.String("category", "drone", "Default category for prototypes")
	flag.Parse()

	files, err := collectWAVFiles(*inputDir)
	if err != nil {
		log.Fatalf("failed to list directory: %v", err)
	}

	if len(files) == 0 {
		log.Fatalf("no WAV files found in %s", *inputDir)
	}

	log.Printf("Found %d WAV files in %s\n", len(files), *inputDir)
	log.Println("Building prototypes...")

	var prototypes []drone.Prototype
	for _, filePath := range files {
		label := inferLabel(filePath)
		log.Printf("Processing: %s (label: %s)", filepath.Base(filePath), label)

		proto, err := drone.BuildPrototypeFromPath(
			filePath,
			label,
			*category,
			fmt.Sprintf("Generated from %s", filepath.Base(filePath)),
			filePath,
			nil,
		)
		if err != nil {
			log.Printf("  ERROR: %v", err)
			continue
		}

		prototypes = append(prototypes, proto)
		log.Printf("  ✓ Created prototype %s", proto.ID)
	}

	if len(prototypes) == 0 {
		log.Fatalf("no prototypes were created")
	}

	// Ensure output directory exists
	outDir := filepath.Dir(*outputFile)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}

	// Write prototypes to JSON
	data, err := json.MarshalIndent(prototypes, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal prototypes: %v", err)
	}

	if err := os.WriteFile(*outputFile, data, 0644); err != nil {
		log.Fatalf("failed to write output file: %v", err)
	}

	log.Printf("\n✓ Successfully created %d prototypes in %s", len(prototypes), *outputFile)

	// Show label distribution
	labelCounts := make(map[string]int)
	for _, p := range prototypes {
		labelCounts[p.Label]++
	}

	log.Println("\nLabel distribution:")
	for label, count := range labelCounts {
		log.Printf("  %s: %d", label, count)
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

	// Remove common suffixes like "_fast", "_slow", "_noisy"
	name = strings.TrimSuffix(name, "_fast")
	name = strings.TrimSuffix(name, "_slow")
	name = strings.TrimSuffix(name, "_noisy")

	// Convert to lowercase and replace underscores with spaces
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", " ")

	return name
}

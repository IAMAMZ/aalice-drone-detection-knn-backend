package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"song-recognition/drone"
)

// Config holds training configuration
type Config struct {
	TrainingDataDir string
	OutputPath      string
	Category        string
	Verbose         bool
}

// TrainingStats tracks training process statistics
type TrainingStats struct {
	TotalSamples     int
	SuccessfulCount  int
	FailedCount      int
	LabelCounts      map[string]int
	ProcessingTimeMs float64
}

func main() {
	config := parseFlags()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("=== Drone Classifier Training Pipeline ===\n")
	log.Printf("Training data: %s\n", config.TrainingDataDir)
	log.Printf("Output model: %s\n", config.OutputPath)
	log.Println()

	startTime := time.Now()

	// Step 1: Discover training data structure
	log.Println("Step 1: Discovering training data...")
	subdirs, err := discoverSubdirectories(config.TrainingDataDir)
	if err != nil {
		log.Fatalf("ERROR: Failed to read training directory: %v", err)
	}

	if len(subdirs) == 0 {
		log.Fatalf("ERROR: No subdirectories found in %s", config.TrainingDataDir)
	}

	log.Printf("Found %d classes:\n", len(subdirs))
	for _, dir := range subdirs {
		files, _ := collectAudioFiles(dir)
		log.Printf("  - %s: %d samples\n", filepath.Base(dir), len(files))
	}
	log.Println()

	// Step 2: Build prototypes
	log.Println("Step 2: Building prototypes from audio files...")
	prototypes, stats := buildPrototypes(subdirs, config)

	if len(prototypes) == 0 {
		log.Fatalf("ERROR: No prototypes were created")
	}

	log.Printf("Successfully created %d/%d prototypes\n",
		stats.SuccessfulCount, stats.TotalSamples)
	if stats.FailedCount > 0 {
		log.Printf("WARNING: %d samples failed to process\n", stats.FailedCount)
	}
	log.Println()

	// Step 3: Save prototypes
	log.Println("Step 3: Saving model to disk...")
	if err := savePrototypes(prototypes, config.OutputPath); err != nil {
		log.Fatalf("ERROR: Failed to save prototypes: %v", err)
	}

	log.Printf("Model saved to: %s\n", config.OutputPath)
	log.Println()

	// Step 4: Print summary
	printTrainingSummary(prototypes, stats, startTime)
}

func parseFlags() Config {
	config := Config{}

	flag.StringVar(&config.TrainingDataDir, "train-dir", "Drone-Training-Data",
		"Directory containing training data organized by class folders")
	flag.StringVar(&config.OutputPath, "output", "drone/prototypes.json",
		"Output path for trained model (prototypes JSON file)")
	flag.StringVar(&config.Category, "category", "drone",
		"Default category for samples (drone/noise)")
	flag.BoolVar(&config.Verbose, "verbose", false,
		"Enable verbose logging")

	flag.Parse()

	// Validate paths
	if _, err := os.Stat(config.TrainingDataDir); os.IsNotExist(err) {
		log.Fatalf("ERROR: Training directory does not exist: %s", config.TrainingDataDir)
	}

	return config
}

func discoverSubdirectories(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	var subdirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		subdirs = append(subdirs, filepath.Join(rootDir, entry.Name()))
	}

	return subdirs, nil
}

func collectAudioFiles(dir string) ([]string, error) {
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

func buildPrototypes(subdirs []string, config Config) ([]drone.Prototype, TrainingStats) {
	var allPrototypes []drone.Prototype
	stats := TrainingStats{
		LabelCounts: make(map[string]int),
	}

	for _, subdir := range subdirs {
		label := inferLabelFromDirectory(subdir)
		category := inferCategory(label, config.Category)

		if config.Verbose {
			log.Printf("Processing class: %s (category: %s)\n", label, category)
		}

		files, err := collectAudioFiles(subdir)
		if err != nil {
			log.Printf("  WARNING: Failed to read directory %s: %v\n", subdir, err)
			continue
		}

		if len(files) == 0 {
			log.Printf("  WARNING: No audio files in %s\n", subdir)
			continue
		}

		// Process each audio file
		for i, filePath := range files {
			stats.TotalSamples++

			if config.Verbose {
				log.Printf("  [%d/%d] %s", i+1, len(files), filepath.Base(filePath))
			}

			proto, err := drone.BuildPrototypeFromPath(
				filePath,
				label,
				category,
				fmt.Sprintf("%s from %s", label, filepath.Base(filePath)),
				filePath,
				nil,
			)

			if err != nil {
				log.Printf("  ERROR processing %s: %v\n", filepath.Base(filePath), err)
				stats.FailedCount++
				continue
			}

			allPrototypes = append(allPrototypes, proto)
			stats.LabelCounts[label]++
			stats.SuccessfulCount++

			if config.Verbose {
				log.Printf(" ✓\n")
			}
		}
	}

	return allPrototypes, stats
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
	noiseKeywords := []string{"noise", "ambient", "silence", "background"}

	for _, keyword := range noiseKeywords {
		if strings.Contains(labelLower, keyword) {
			return "noise"
		}
	}

	return defaultCategory
}

func savePrototypes(prototypes []drone.Prototype, outputPath string) error {
	// Ensure output directory exists
	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(prototypes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prototypes: %w", err)
	}

	// Write atomically using temp file
	tempPath := outputPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempPath, outputPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func printTrainingSummary(prototypes []drone.Prototype, stats TrainingStats, startTime time.Time) {
	elapsed := time.Since(startTime)

	// Calculate category distribution
	categoryCount := make(map[string]int)
	for _, p := range prototypes {
		categoryCount[p.Category]++
	}

	log.Println("=== Training Summary ===")
	log.Println()
	log.Printf("Total training samples: %d\n", stats.TotalSamples)
	log.Printf("Successfully processed: %d (%.1f%%)\n",
		stats.SuccessfulCount,
		float64(stats.SuccessfulCount)/float64(stats.TotalSamples)*100)
	log.Printf("Failed to process: %d\n", stats.FailedCount)
	log.Println()

	log.Println("Class distribution:")
	for label, count := range stats.LabelCounts {
		log.Printf("  %-20s: %3d prototypes\n", label, count)
	}
	log.Println()

	log.Println("Category distribution:")
	for category, count := range categoryCount {
		log.Printf("  %-20s: %3d prototypes\n", category, count)
	}
	log.Println()

	log.Printf("Total training time: %.2f seconds\n", elapsed.Seconds())
	log.Printf("Average time per sample: %.2f ms\n",
		elapsed.Seconds()*1000/float64(stats.TotalSamples))
	log.Println()
	log.Println("✓ Training complete!")
}

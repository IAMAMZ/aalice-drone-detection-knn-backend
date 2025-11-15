package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"song-recognition/drone"
	"song-recognition/wav"
)

// TestConfig holds test configuration
type TestConfig struct {
	ModelPath   string
	TestDataDir string
	K           int
	OutputCSV   string
	OutputJSON  string
	TopK        int
	Verbose     bool
}

// TestPrediction stores prediction results for a single test sample
type TestPrediction struct {
	Filename       string             `json:"filename"`
	PredictedClass string             `json:"predicted_class"`
	Confidence     float64            `json:"confidence"`
	TopPredictions []drone.Prediction `json:"top_predictions"`
	ProcessingTime float64            `json:"processing_time_ms"`
	SNR            float64            `json:"snr_db"`
}

// TestReport contains all test results
type TestReport struct {
	Timestamp     time.Time        `json:"timestamp"`
	ModelPath     string           `json:"model_path"`
	TestDataDir   string           `json:"test_data_dir"`
	TotalSamples  int              `json:"total_samples"`
	Predictions   []TestPrediction `json:"predictions"`
	AvgConfidence float64          `json:"avg_confidence"`
	AvgProcessing float64          `json:"avg_processing_ms"`
}

func main() {
	config := parseFlags()

	log.SetFlags(log.Ldate | log.Ltime)
	log.Println("=== Model Testing Pipeline ===")
	log.Printf("Model: %s\n", config.ModelPath)
	log.Printf("Test data: %s\n", config.TestDataDir)
	log.Printf("K neighbors: %d\n", config.K)
	log.Println()

	// Load classifier
	log.Println("Loading trained model...")
	classifier, err := drone.NewClassifierFromFile(config.ModelPath, config.K)
	if err != nil {
		log.Fatalf("ERROR: Failed to load model: %v", err)
	}

	stats := classifier.Stats()
	log.Printf("Loaded %d prototypes covering %d classes\n",
		stats.PrototypeCount, stats.LabelCount)
	log.Println("Classes: ", formatClassList(stats.Labels))
	log.Println()

	// Find test files
	log.Println("Discovering test samples...")
	testFiles, err := collectTestFiles(config.TestDataDir)
	if err != nil {
		log.Fatalf("ERROR: Failed to read test directory: %v", err)
	}

	if len(testFiles) == 0 {
		log.Fatalf("ERROR: No test files found in %s", config.TestDataDir)
	}

	log.Printf("Found %d test samples\n", len(testFiles))
	log.Println()

	// Run predictions
	log.Println("Running predictions...")
	report := runPredictions(classifier, testFiles, config)

	// Print results
	printTestReport(report, config)

	// Save outputs
	if config.OutputCSV != "" {
		if err := saveCSV(report, config.OutputCSV); err != nil {
			log.Printf("WARNING: Failed to save CSV: %v\n", err)
		} else {
			log.Printf("CSV results saved to: %s\n", config.OutputCSV)
		}
	}

	if config.OutputJSON != "" {
		if err := saveJSON(report, config.OutputJSON); err != nil {
			log.Printf("WARNING: Failed to save JSON: %v\n", err)
		} else {
			log.Printf("JSON results saved to: %s\n", config.OutputJSON)
		}
	}

	log.Println()
	log.Println("✓ Testing complete!")
}

func parseFlags() TestConfig {
	config := TestConfig{}

	flag.StringVar(&config.ModelPath, "model", "drone/prototypes.json",
		"Path to trained model (prototypes JSON)")
	flag.StringVar(&config.TestDataDir, "test-dir", "Test data",
		"Directory containing test samples")
	flag.IntVar(&config.K, "k", 5,
		"Number of nearest neighbors")
	flag.StringVar(&config.OutputCSV, "output-csv", "test_predictions.csv",
		"Path to save predictions as CSV")
	flag.StringVar(&config.OutputJSON, "output-json", "test_predictions.json",
		"Path to save predictions as JSON")
	flag.IntVar(&config.TopK, "top-k", 3,
		"Number of top predictions to include")
	flag.BoolVar(&config.Verbose, "verbose", false,
		"Enable verbose logging")

	flag.Parse()

	return config
}

func collectTestFiles(dir string) ([]string, error) {
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

	// Sort for consistent ordering
	sort.Strings(files)

	return files, nil
}

func runPredictions(classifier *drone.Classifier, testFiles []string, config TestConfig) TestReport {
	report := TestReport{
		Timestamp:   time.Now(),
		ModelPath:   config.ModelPath,
		TestDataDir: config.TestDataDir,
	}

	totalConfidence := 0.0
	totalProcessing := 0.0

	for i, filePath := range testFiles {
		if config.Verbose {
			log.Printf("[%d/%d] Processing %s...\n", i+1, len(testFiles), filepath.Base(filePath))
		}

		prediction := predictFile(classifier, filePath, config)
		report.Predictions = append(report.Predictions, prediction)

		totalConfidence += prediction.Confidence
		totalProcessing += prediction.ProcessingTime

		if !config.Verbose {
			// Show progress
			if (i+1)%5 == 0 || i+1 == len(testFiles) {
				log.Printf("Progress: %d/%d (%.1f%%)\n", i+1, len(testFiles),
					float64(i+1)/float64(len(testFiles))*100)
			}
		}
	}

	report.TotalSamples = len(testFiles)
	if report.TotalSamples > 0 {
		report.AvgConfidence = totalConfidence / float64(report.TotalSamples)
		report.AvgProcessing = totalProcessing / float64(report.TotalSamples)
	}

	return report
}

func predictFile(classifier *drone.Classifier, filePath string, config TestConfig) TestPrediction {
	startTime := time.Now()

	pred := TestPrediction{
		Filename: filepath.Base(filePath),
	}

	// Convert to WAV if needed
	wavPath, err := wav.ConvertToWAV(filePath, 1)
	if err != nil {
		log.Printf("ERROR converting %s: %v\n", pred.Filename, err)
		return pred
	}
	defer func() {
		if wavPath != filePath {
			os.Remove(wavPath)
		}
	}()

	// Read WAV
	wavInfo, err := wav.ReadWavInfo(wavPath)
	if err != nil {
		log.Printf("ERROR reading %s: %v\n", pred.Filename, err)
		return pred
	}

	// Extract samples
	samples, err := wav.WavBytesToSamples(wavInfo.Data)
	if err != nil {
		log.Printf("ERROR extracting samples from %s: %v\n", pred.Filename, err)
		return pred
	}

	// Estimate SNR
	pred.SNR = drone.EstimateSNR(samples)

	// Preprocess
	preprocessCfg := drone.DefaultPreprocessingConfig()
	processed := drone.PreprocessAudio(samples, wavInfo.SampleRate, preprocessCfg)

	// Extract features
	features, err := drone.ExtractFeatureVector(processed, wavInfo.SampleRate)
	if err != nil {
		log.Printf("ERROR extracting features from %s: %v\n", pred.Filename, err)
		return pred
	}

	// Classify
	predictions, err := classifier.Predict(features)
	if err != nil || len(predictions) == 0 {
		log.Printf("ERROR classifying %s: %v\n", pred.Filename, err)
		return pred
	}

	// Store results
	pred.PredictedClass = predictions[0].Label
	pred.Confidence = predictions[0].Confidence

	// Store top-K predictions
	topK := config.TopK
	if topK > len(predictions) {
		topK = len(predictions)
	}
	pred.TopPredictions = predictions[:topK]

	pred.ProcessingTime = time.Since(startTime).Seconds() * 1000

	return pred
}

func printTestReport(report TestReport, config TestConfig) {
	log.Println()
	log.Println("=" + strings.Repeat("=", 79))
	log.Println("TEST RESULTS")
	log.Println("=" + strings.Repeat("=", 79))
	log.Println()

	log.Printf("Total test samples: %d\n", report.TotalSamples)
	log.Printf("Average confidence: %.2f%%\n", report.AvgConfidence*100)
	log.Printf("Average processing time: %.2f ms/sample\n", report.AvgProcessing)
	log.Println()

	// Class distribution of predictions
	classCount := make(map[string]int)
	for _, pred := range report.Predictions {
		classCount[pred.PredictedClass]++
	}

	log.Println("Predicted class distribution:")
	log.Println(strings.Repeat("-", 80))

	// Sort by count
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range classCount {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	for _, kv := range sorted {
		percentage := float64(kv.Value) / float64(report.TotalSamples) * 100
		log.Printf("  %-20s: %3d samples (%.1f%%)\n", kv.Key, kv.Value, percentage)
	}
	log.Println()

	// Individual predictions
	if config.Verbose {
		log.Println("Individual predictions:")
		log.Println(strings.Repeat("-", 80))
		log.Printf("%-20s %-15s %10s %10s\n", "Filename", "Predicted", "Confidence", "SNR (dB)")
		log.Println(strings.Repeat("-", 80))

		for _, pred := range report.Predictions {
			log.Printf("%-20s %-15s %9.1f%% %9.1f\n",
				pred.Filename, pred.PredictedClass, pred.Confidence*100, pred.SNR)
		}
		log.Println()
	}

	// Confidence statistics
	var confidences []float64
	for _, pred := range report.Predictions {
		confidences = append(confidences, pred.Confidence)
	}

	sort.Float64s(confidences)
	log.Println("Confidence statistics:")
	log.Printf("  Min: %.2f%%\n", confidences[0]*100)
	log.Printf("  Max: %.2f%%\n", confidences[len(confidences)-1]*100)
	log.Printf("  Median: %.2f%%\n", confidences[len(confidences)/2]*100)
	log.Println()
}

func saveCSV(report TestReport, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	if err := writer.Write([]string{
		"filename",
		"predicted_class",
		"confidence",
		"snr_db",
		"processing_time_ms",
	}); err != nil {
		return err
	}

	// Data rows
	for _, pred := range report.Predictions {
		if err := writer.Write([]string{
			pred.Filename,
			pred.PredictedClass,
			fmt.Sprintf("%.4f", pred.Confidence),
			fmt.Sprintf("%.2f", pred.SNR),
			fmt.Sprintf("%.2f", pred.ProcessingTime),
		}); err != nil {
			return err
		}
	}

	return nil
}

func saveJSON(report TestReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func formatClassList(labels []drone.ModelLabelStat) string {
	var names []string
	for _, l := range labels {
		names = append(names, l.Label)
	}
	return strings.Join(names, ", ")
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"song-recognition/drone"
	"song-recognition/wav"
)

// EvaluationConfig holds evaluation parameters
type EvaluationConfig struct {
	ModelPath       string
	TrainingDataDir string
	K               int
	ReportPath      string
	Verbose         bool
}

// ClassMetrics tracks per-class performance
type ClassMetrics struct {
	ClassName     string
	TotalSamples  int
	CorrectCount  int
	Accuracy      float64
	AvgConfidence float64
	ConfidenceStd float64
	Misclassified []MisclassificationInfo
}

// MisclassificationInfo stores details of incorrect predictions
type MisclassificationInfo struct {
	Filename       string
	TrueLabel      string
	PredictedLabel string
	Confidence     float64
}

// EvaluationReport contains comprehensive evaluation results
type EvaluationReport struct {
	Timestamp       time.Time
	ModelPath       string
	TotalSamples    int
	CorrectCount    int
	OverallAccuracy float64
	AvgConfidence   float64
	ClassMetrics    []ClassMetrics
	ConfusionMatrix map[string]map[string]int
	ProcessingTime  time.Duration
}

func main() {
	config := parseFlags()

	log.SetFlags(log.Ldate | log.Ltime)
	log.Println("=== Model Evaluation Pipeline ===")
	log.Printf("Model: %s\n", config.ModelPath)
	log.Printf("Training data: %s\n", config.TrainingDataDir)
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
	log.Println()

	// Discover evaluation data
	log.Println("Discovering evaluation data...")
	subdirs, err := discoverSubdirectories(config.TrainingDataDir)
	if err != nil {
		log.Fatalf("ERROR: Failed to read evaluation directory: %v", err)
	}

	log.Printf("Found %d classes to evaluate\n", len(subdirs))
	log.Println()

	// Evaluate each class
	log.Println("Evaluating model performance...")
	report := evaluateModel(classifier, subdirs, config)

	// Print results
	printEvaluationReport(report)

	// Save report if requested
	if config.ReportPath != "" {
		if err := saveReport(report, config.ReportPath); err != nil {
			log.Printf("WARNING: Failed to save report: %v\n", err)
		} else {
			log.Printf("\nReport saved to: %s\n", config.ReportPath)
		}
	}

	// Print final verdict
	log.Println()
	printVerdict(report)
}

func parseFlags() EvaluationConfig {
	config := EvaluationConfig{}

	flag.StringVar(&config.ModelPath, "model", "drone/prototypes.json",
		"Path to trained model (prototypes JSON)")
	flag.StringVar(&config.TrainingDataDir, "train-dir", "Drone-Training-Data",
		"Directory containing training data to evaluate")
	flag.IntVar(&config.K, "k", 5,
		"Number of nearest neighbors")
	flag.StringVar(&config.ReportPath, "report", "evaluation_report.json",
		"Path to save evaluation report (empty to skip)")
	flag.BoolVar(&config.Verbose, "verbose", false,
		"Enable verbose logging")

	flag.Parse()

	return config
}

func discoverSubdirectories(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	var subdirs []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		subdirs = append(subdirs, filepath.Join(rootDir, entry.Name()))
	}

	return subdirs, nil
}

func evaluateModel(classifier *drone.Classifier, subdirs []string, config EvaluationConfig) EvaluationReport {
	report := EvaluationReport{
		Timestamp:       time.Now(),
		ModelPath:       config.ModelPath,
		ConfusionMatrix: make(map[string]map[string]int),
	}

	var allMetrics []ClassMetrics
	totalCorrect := 0
	totalSamples := 0
	totalConfidence := 0.0

	for _, subdir := range subdirs {
		trueLabel := inferLabelFromDirectory(subdir)
		metrics := evaluateClass(classifier, subdir, trueLabel, config, &report)

		allMetrics = append(allMetrics, metrics)
		totalCorrect += metrics.CorrectCount
		totalSamples += metrics.TotalSamples
		totalConfidence += metrics.AvgConfidence * float64(metrics.TotalSamples)
	}

	report.ClassMetrics = allMetrics
	report.TotalSamples = totalSamples
	report.CorrectCount = totalCorrect
	report.OverallAccuracy = float64(totalCorrect) / float64(totalSamples) * 100
	report.AvgConfidence = totalConfidence / float64(totalSamples)
	report.ProcessingTime = time.Since(report.Timestamp)

	return report
}

func evaluateClass(classifier *drone.Classifier, classDir string, trueLabel string,
	config EvaluationConfig, report *EvaluationReport) ClassMetrics {

	metrics := ClassMetrics{
		ClassName: trueLabel,
	}

	files, err := collectAudioFiles(classDir)
	if err != nil {
		log.Printf("WARNING: Failed to read directory %s: %v\n", classDir, err)
		return metrics
	}

	if len(files) == 0 {
		log.Printf("WARNING: No audio files in %s\n", classDir)
		return metrics
	}

	var confidences []float64

	for _, filePath := range files {
		metrics.TotalSamples++

		// Load and process audio
		prediction, conf, err := classifyAudio(classifier, filePath)
		if err != nil {
			if config.Verbose {
				log.Printf("  ERROR processing %s: %v\n", filepath.Base(filePath), err)
			}
			continue
		}

		confidences = append(confidences, conf)

		// Update confusion matrix
		if report.ConfusionMatrix[trueLabel] == nil {
			report.ConfusionMatrix[trueLabel] = make(map[string]int)
		}
		report.ConfusionMatrix[trueLabel][prediction]++

		// Check if correct
		if prediction == trueLabel {
			metrics.CorrectCount++
		} else {
			metrics.Misclassified = append(metrics.Misclassified, MisclassificationInfo{
				Filename:       filepath.Base(filePath),
				TrueLabel:      trueLabel,
				PredictedLabel: prediction,
				Confidence:     conf,
			})
		}
	}

	// Calculate statistics
	if metrics.TotalSamples > 0 {
		metrics.Accuracy = float64(metrics.CorrectCount) / float64(metrics.TotalSamples) * 100
	}

	if len(confidences) > 0 {
		metrics.AvgConfidence = average(confidences)
		metrics.ConfidenceStd = stddev(confidences, metrics.AvgConfidence)
	}

	return metrics
}

func classifyAudio(classifier *drone.Classifier, filePath string) (string, float64, error) {
	// Convert to WAV if needed
	wavPath, err := wav.ConvertToWAV(filePath, 1)
	if err != nil {
		return "", 0, err
	}
	defer func() {
		if wavPath != filePath {
			os.Remove(wavPath)
		}
	}()

	// Read WAV
	wavInfo, err := wav.ReadWavInfo(wavPath)
	if err != nil {
		return "", 0, err
	}

	// Extract samples
	samples, err := wav.WavBytesToSamples(wavInfo.Data)
	if err != nil {
		return "", 0, err
	}

	// Preprocess
	preprocessCfg := drone.DefaultPreprocessingConfig()
	processed := drone.PreprocessAudio(samples, wavInfo.SampleRate, preprocessCfg)

	// Extract features
	features, err := drone.ExtractFeatureVector(processed, wavInfo.SampleRate)
	if err != nil {
		return "", 0, err
	}

	// Classify
	predictions, err := classifier.Predict(features)
	if err != nil || len(predictions) == 0 {
		return "", 0, fmt.Errorf("classification failed")
	}

	return predictions[0].Label, predictions[0].Confidence, nil
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

func inferLabelFromDirectory(dirPath string) string {
	base := filepath.Base(dirPath)
	label := strings.ToLower(base)
	label = strings.ReplaceAll(label, "_", " ")
	label = strings.ReplaceAll(label, "-", " ")
	return strings.TrimSpace(label)
}

func printEvaluationReport(report EvaluationReport) {
	log.Println()
	log.Println("=" + strings.Repeat("=", 79))
	log.Println("EVALUATION RESULTS")
	log.Println("=" + strings.Repeat("=", 79))
	log.Println()

	// Overall metrics
	log.Printf("Overall Accuracy: %.2f%% (%d/%d correct)\n",
		report.OverallAccuracy, report.CorrectCount, report.TotalSamples)
	log.Printf("Average Confidence: %.2f%%\n", report.AvgConfidence*100)
	log.Printf("Processing Time: %.2f seconds\n", report.ProcessingTime.Seconds())
	log.Println()

	// Per-class metrics
	log.Println("Per-Class Performance:")
	log.Println(strings.Repeat("-", 80))
	log.Printf("%-20s %8s %10s %12s\n", "Class", "Accuracy", "Confidence", "Samples")
	log.Println(strings.Repeat("-", 80))

	// Sort by accuracy for better readability
	sortedMetrics := make([]ClassMetrics, len(report.ClassMetrics))
	copy(sortedMetrics, report.ClassMetrics)
	sort.Slice(sortedMetrics, func(i, j int) bool {
		return sortedMetrics[i].Accuracy > sortedMetrics[j].Accuracy
	})

	for _, m := range sortedMetrics {
		status := "✓"
		if m.Accuracy < 70 {
			status = "⚠"
		}
		log.Printf("%-20s %7.1f%% %9.1f%% %10d   %s\n",
			m.ClassName, m.Accuracy, m.AvgConfidence*100, m.TotalSamples, status)
	}
	log.Println()

	// Confusion matrix
	printConfusionMatrix(report.ConfusionMatrix)

	// Misclassifications
	printMisclassifications(report.ClassMetrics)
}

func printConfusionMatrix(matrix map[string]map[string]int) {
	if len(matrix) == 0 {
		return
	}

	log.Println("Confusion Matrix:")
	log.Println(strings.Repeat("-", 80))

	// Get sorted labels
	var labels []string
	for label := range matrix {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	// Print header
	fmt.Printf("%-15s", "Actual \\ Pred")
	for _, label := range labels {
		fmt.Printf(" %6s", truncate(label, 6))
	}
	fmt.Println()
	log.Println(strings.Repeat("-", 80))

	// Print rows
	for _, trueLabel := range labels {
		fmt.Printf("%-15s", truncate(trueLabel, 15))
		for _, predLabel := range labels {
			count := matrix[trueLabel][predLabel]
			if count > 0 {
				fmt.Printf(" %6d", count)
			} else {
				fmt.Printf(" %6s", ".")
			}
		}
		fmt.Println()
	}
	log.Println()
}

func printMisclassifications(metrics []ClassMetrics) {
	totalMisclassified := 0
	for _, m := range metrics {
		totalMisclassified += len(m.Misclassified)
	}

	if totalMisclassified == 0 {
		log.Println("✓ No misclassifications!")
		return
	}

	log.Printf("Misclassifications (%d total):\n", totalMisclassified)
	log.Println(strings.Repeat("-", 80))

	for _, m := range metrics {
		if len(m.Misclassified) == 0 {
			continue
		}
		log.Printf("\n%s:", m.ClassName)
		for _, misc := range m.Misclassified {
			log.Printf("  %s → predicted as '%s' (%.1f%% confidence)\n",
				misc.Filename, misc.PredictedLabel, misc.Confidence*100)
		}
	}
	log.Println()
}

func printVerdict(report EvaluationReport) {
	log.Println("=" + strings.Repeat("=", 79))
	log.Println("VERDICT")
	log.Println("=" + strings.Repeat("=", 79))

	accuracy := report.OverallAccuracy
	confidence := report.AvgConfidence * 100

	var verdict string
	var recommendation string

	if accuracy >= 90 {
		verdict = "✓ EXCELLENT"
		recommendation = "Model is production-ready!"
	} else if accuracy >= 80 {
		verdict = "✓ GOOD"
		recommendation = "Model works well. Consider adding more diverse training data."
	} else if accuracy >= 70 {
		verdict = "⚠ FAIR"
		recommendation = "Model has significant room for improvement. Add more training data."
	} else {
		verdict = "✗ POOR"
		recommendation = "Model needs substantial improvement. Check data quality and add more samples."
	}

	log.Printf("Overall Assessment: %s\n", verdict)
	log.Printf("Accuracy: %.2f%%, Confidence: %.2f%%\n", accuracy, confidence)
	log.Printf("Recommendation: %s\n", recommendation)
	log.Println("=" + strings.Repeat("=", 79))
}

func saveReport(report EvaluationReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stddev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return math.Sqrt(variance / float64(len(values)))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}

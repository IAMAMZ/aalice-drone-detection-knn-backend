package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"song-recognition/drone"
	"song-recognition/utils"
	"song-recognition/wav"
)

func main() {
	defaultDir := filepath.Join("train_data")
	defaultModel := utils.GetEnv("DRONE_MODEL_PATH", filepath.Join("drone", "prototypes.json"))
	defaultK := getEnvInt("DRONE_MODEL_K", 5)

	dirFlag := flag.String("dir", defaultDir, "Directory containing WAV files to evaluate")
	modelFlag := flag.String("model", defaultModel, "Path to prototypes JSON")
	kFlag := flag.Int("k", defaultK, "Number of neighbours for classifier")
	windowDur := flag.Float64("window", 3.0, "Sliding window duration in seconds")
	windowOverlap := flag.Float64("overlap", 1.5, "Sliding window overlap in seconds")
	flag.Parse()

	classifier, err := drone.NewClassifierFromFile(*modelFlag, *kFlag)
	if err != nil {
		log.Fatalf("failed to load classifier: %v", err)
	}

	files, err := collectAudioFiles(*dirFlag)
	if err != nil {
		log.Fatalf("failed to list directory %s: %v", *dirFlag, err)
	}
	if len(files) == 0 {
		log.Fatalf("no WAV files found in %s", *dirFlag)
	}

	fmt.Printf("Evaluating %d files from %s using model %s (k=%d)\n\n", len(files), *dirFlag, *modelFlag, *kFlag)

	for _, filePath := range files {
		if err := evaluateFile(classifier, filePath, *windowDur, *windowOverlap); err != nil {
			log.Printf("ERROR processing %s: %v\n", filePath, err)
		}
	}
}

func collectAudioFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
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

func evaluateFile(classifier *drone.Classifier, path string, windowSeconds, overlapSeconds float64) error {
	fmt.Printf("=== %s ===\n", filepath.Base(path))
	start := time.Now()

	tempPath, err := wav.ConvertToWAV(path, 1)
	if err != nil {
		return fmt.Errorf("convert to wav: %w", err)
	}
	defer os.Remove(tempPath)

	wavInfo, err := wav.ReadWavInfo(tempPath)
	if err != nil {
		return fmt.Errorf("read wav info: %w", err)
	}

	samples, err := wav.WavBytesToSamples(wavInfo.Data)
	if err != nil {
		return fmt.Errorf("decode samples: %w", err)
	}

	duration := float64(len(samples)) / float64(wavInfo.SampleRate)
	snrDb := drone.EstimateSNR(samples)

	preCfg := drone.DefaultPreprocessingConfig()
	processed := drone.PreprocessAudio(samples, wavInfo.SampleRate, preCfg)

	features, err := drone.ExtractFeatureVector(processed, wavInfo.SampleRate)
	if err != nil {
		return fmt.Errorf("extract features: %w", err)
	}
	// Don't normalize here - the classifier will handle feature scaling and normalization

	predictions, _, err := classifier.PredictWithSlidingWindows(processed, wavInfo.SampleRate, windowSeconds, overlapSeconds)
	if err != nil || len(predictions) == 0 {
		predictions, err = classifier.Predict(features)
	}
	if err != nil {
		return fmt.Errorf("classifier error: %w", err)
	}

	if len(predictions) == 0 {
		fmt.Println("No predictions")
		return nil
	}

	best := predictions[0]
	fmt.Printf("Duration: %.2fs, SNR: %.1fdB, Best label: %s (%.1f%%, category=%s)\n",
		duration, snrDb, best.Label, best.Confidence*100, best.Category)
	fmt.Printf("Top predictions:\n")
	for idx, pred := range predictions {
		if idx >= 5 {
			break
		}
		fmt.Printf("  #%d %-20s conf=%.2f avgDist=%.4f support=%d\n",
			idx+1, pred.Label, pred.Confidence, pred.AverageDist, pred.Support)
	}
	fmt.Printf("Elapsed: %.2fms\n\n", time.Since(start).Seconds()*1000)
	return nil
}

func getEnvInt(key string, fallback int) int {
	val := utils.GetEnv(key, "")
	if val == "" {
		return fallback
	}
	if parsed, err := strconv.Atoi(val); err == nil {
		return parsed
	}
	return fallback
}

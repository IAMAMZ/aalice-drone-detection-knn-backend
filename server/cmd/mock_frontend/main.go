package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"song-recognition/drone"
	"song-recognition/models"
	"song-recognition/wav"
)

func main() {
	dir := flag.String("dir", filepath.Join("train_data"), "Directory containing WAV samples to upload (ignored if -file is set)")
	file := flag.String("file", "", "Single WAV file to upload (overrides -dir)")
	endpoint := flag.String("url", "http://localhost:5000/api/audio/classify", "Classification endpoint")
	latFlag := flag.Float64("lat", math.NaN(), "Optional latitude to include with uploads")
	lonFlag := flag.Float64("lon", math.NaN(), "Optional longitude to include with uploads")
	delay := flag.Duration("delay", 2*time.Second, "Delay between uploads when using -dir")
	flag.Parse()

	files, err := resolveFiles(*file, *dir)
	if err != nil {
		log.Fatalf("failed to resolve files: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("no WAV files found (file=%s dir=%s)", *file, *dir)
	}

	fmt.Printf("Uploading %d sample(s) to %s\n\n", len(files), *endpoint)
	for idx, path := range files {
		if err := uploadSample(path, *endpoint, latFlag, lonFlag); err != nil {
			log.Printf("upload failed for %s: %v\n", path, err)
		}

		if idx < len(files)-1 && *delay > 0 {
			time.Sleep(*delay)
		}
	}
}

func resolveFiles(single, dir string) ([]string, error) {
	if single != "" {
		return []string{single}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".wav" && filepath.Ext(entry.Name()) != ".WAV" {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	return files, nil
}

func uploadSample(path, endpoint string, latFlag, lonFlag *float64) error {
	fmt.Printf("→ %s\n", filepath.Base(path))

	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read wav: %w", err)
	}

	wavInfo, err := wav.ReadWavInfo(path)
	if err != nil {
		return fmt.Errorf("parse wav: %w", err)
	}

	record := models.RecordData{
		Audio:      base64.StdEncoding.EncodeToString(raw),
		Duration:   wavInfo.Duration,
		Channels:   wavInfo.Channels,
		SampleRate: wavInfo.SampleRate,
		SampleSize: wavInfo.BitsPerSample,
	}

	if !math.IsNaN(*latFlag) {
		record.Latitude = latFlag
	}
	if !math.IsNaN(*lonFlag) {
		record.Longitude = lonFlag
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post classification request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var summary drone.ClassificationSummary
	if err := json.Unmarshal(body, &summary); err != nil {
		return fmt.Errorf("decode classification response: %w", err)
	}

	if len(summary.Predictions) == 0 {
		fmt.Println("   no predictions returned")
	} else {
		best := summary.Predictions[0]
		fmt.Printf("   best=%s (%.1f%%) adjustedThreshold=%.2f templateHits=%d\n",
			best.Label, best.Confidence*100, summary.AdjustedThreshold, len(summary.TemplatePreds))
	}

	if summary.RecordingPath != "" {
		fmt.Printf("   saved recording: %s\n", summary.RecordingPath)
	}

	return nil
}

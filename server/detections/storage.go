package detections

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"song-recognition/models"
	"song-recognition/utils"
	"sync"
	"time"
)

var (
	detectionsFile = "detections.json"
	mu             sync.RWMutex
)

// loadDetectionsInternal loads all detections from the JSON file (without lock)
func loadDetectionsInternal() ([]models.Detection, error) {
	filePath := filepath.Join("server", detectionsFile)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Return empty slice if file doesn't exist
		return []models.Detection{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading detections file: %v", err)
	}

	if len(data) == 0 {
		return []models.Detection{}, nil
	}

	var detections []models.Detection
	if err := json.Unmarshal(data, &detections); err != nil {
		return nil, fmt.Errorf("error unmarshaling detections: %v", err)
	}

	return detections, nil
}

// LoadDetections loads all detections from the JSON file
func LoadDetections() ([]models.Detection, error) {
	mu.RLock()
	defer mu.RUnlock()
	return loadDetectionsInternal()
}

// SaveDetection appends a new detection to the JSON file
func SaveDetection(detection *models.Detection) error {
	mu.Lock()
	defer mu.Unlock()

	// Load existing detections (without lock since we already have write lock)
	detections, err := loadDetectionsInternal()
	if err != nil {
		return err
	}

	// Set ID and timestamp if not set
	if detection.ID == 0 {
		detection.ID = time.Now().UnixNano()
	}
	if detection.Timestamp.IsZero() {
		detection.Timestamp = time.Now()
	}

	// Append new detection
	detections = append(detections, *detection)

	// Ensure directory exists
	filePath := filepath.Join("server", detectionsFile)
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := utils.CreateFolder(dir); err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	}

	// Write back to file
	data, err := json.MarshalIndent(detections, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling detections: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing detections file: %v", err)
	}

	return nil
}

// GetAllDetections returns all detections
func GetAllDetections() ([]models.Detection, error) {
	return LoadDetections()
}


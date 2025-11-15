package drone

// Audio Processing Pipeline
//
// This package handles the conversion of raw audio data from the client into processed
// audio samples ready for feature extraction. The pipeline works as follows:
//
// 1. Base64 Decoding: Client sends audio as base64-encoded WAV data
// 2. WAV File Creation: Decoded audio is written to a temporary WAV file
// 3. Audio Reformatting: Audio is converted to mono channel (required for analysis)
// 4. Sample Extraction: PCM samples are extracted as float64 arrays
// 5. Duration Calculation: Audio duration is computed from sample count and sample rate
//
// The processed AudioSample contains normalized PCM samples that can be directly fed into
// the feature extraction pipeline for drone classification.

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"song-recognition/models"
	"song-recognition/utils"
	"song-recognition/wav"
)

// AudioSample bundles decoded PCM samples together with contextual metadata.
type AudioSample struct {
	Samples    []float64
	SampleRate int
	Duration   float64
	Persisted  string
	SNRDb      float64 // Signal-to-noise ratio in dB
}

// PrepareAudioSample converts the base64 payload emitted by the client into fixed
// format PCM samples suitable for feature extraction.
func PrepareAudioSample(recData models.RecordData, persist bool) (*AudioSample, error) {
	decodedAudioData, err := base64.StdEncoding.DecodeString(recData.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 audio: %w", err)
	}

	if err := utils.CreateFolder("tmp"); err != nil {
		return nil, fmt.Errorf("unable to create tmp folder: %w", err)
	}

	fileName := fmt.Sprintf("rec_%d.wav", time.Now().UnixNano())
	filePath := filepath.Join("tmp", fileName)

	if err := wav.WriteWavFile(filePath, decodedAudioData, recData.SampleRate, recData.Channels, recData.SampleSize); err != nil {
		return nil, fmt.Errorf("failed to write wav file: %w", err)
	}

	reformatted, err := wav.ReformatWAV(filePath, 1)
	if err != nil {
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to reformat wav: %w", err)
	}

	wavInfo, err := wav.ReadWavInfo(reformatted)
	if err != nil {
		_ = os.Remove(filePath)
		_ = os.Remove(reformatted)
		return nil, fmt.Errorf("failed to read wav info: %w", err)
	}

	samples, err := wav.WavBytesToSamples(wavInfo.Data)
	if err != nil {
		_ = os.Remove(filePath)
		_ = os.Remove(reformatted)
		return nil, fmt.Errorf("failed to convert samples: %w", err)
	}

	// clean temporary raw capture
	_ = os.Remove(filePath)

	duration := float64(len(samples)) / float64(wavInfo.SampleRate)

	// Estimate SNR before preprocessing
	snrDb := EstimateSNR(samples)

	// Apply audio preprocessing to improve detection in noisy environments
	config := DefaultPreprocessingConfig()
	// Enable preprocessing by default - can be configured via environment variables
	preprocessedSamples := PreprocessAudio(samples, wavInfo.SampleRate, config)

	result := &AudioSample{
		Samples:    preprocessedSamples,
		SampleRate: wavInfo.SampleRate,
		Duration:   duration,
		SNRDb:      snrDb,
	}

	if persist {
		recordingDir := utils.GetEnv("DRONE_RECORDING_DIR", "frontendrecording")
		if err := utils.CreateFolder(recordingDir); err == nil {
			destination := filepath.Join(recordingDir, filepath.Base(reformatted))
			if err := os.Rename(reformatted, destination); err == nil {
				result.Persisted = destination
			} else {
				_ = os.Remove(reformatted)
			}
		} else {
			_ = os.Remove(reformatted)
		}
	} else {
		_ = os.Remove(reformatted)
	}

	return result, nil
}

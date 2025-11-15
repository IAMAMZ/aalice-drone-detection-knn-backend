package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type GoogleTTSClient struct {
	apiKey string
}

type TTSRequest struct {
	Input struct {
		Text string `json:"text"`
	} `json:"input"`
	Voice struct {
		LanguageCode string `json:"languageCode"`
		Name         string `json:"name,omitempty"`
		SsmlGender   string `json:"ssmlGender"`
	} `json:"voice"`
	AudioConfig struct {
		AudioEncoding   string  `json:"audioEncoding"`
		SpeakingRate    float64 `json:"speakingRate,omitempty"`
		Pitch           float64 `json:"pitch,omitempty"`
		VolumeGainDb    float64 `json:"volumeGainDb,omitempty"`
		SampleRateHertz int     `json:"sampleRateHertz,omitempty"`
	} `json:"audioConfig"`
}

type TTSResponse struct {
	AudioContent string `json:"audioContent"`
}

func NewGoogleTTSClient() (*GoogleTTSClient, error) {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	apiKey := os.Getenv("GOOGLE_TTS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_TTS_API_KEY environment variable is required")
	}

	return &GoogleTTSClient{
		apiKey: apiKey,
	}, nil
}

func (g *GoogleTTSClient) SynthesizeText(text string) ([]byte, error) {
	ctx := context.Background()

	// Prepare the TTS request
	ttsReq := TTSRequest{}
	ttsReq.Input.Text = text
	ttsReq.Voice.LanguageCode = "en-US"
	ttsReq.Voice.Name = "en-GB-Standard-F" // Female voice (en-GB-Chirp3-HD-Achernar, en-GB-Chirp-HD-F, en-GB-Chirp3-HD-Sulafat)
	ttsReq.Voice.SsmlGender = "FEMALE"
	ttsReq.AudioConfig.AudioEncoding = "MP3"
	ttsReq.AudioConfig.SpeakingRate = 1.0
	ttsReq.AudioConfig.Pitch = 0.0
	ttsReq.AudioConfig.VolumeGainDb = 0.0
	ttsReq.AudioConfig.SampleRateHertz = 24000

	// Convert to JSON
	jsonData, err := json.Marshal(ttsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TTS request: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("https://texttospeech.googleapis.com/v1/text:synthesize?key=%s", g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send TTS request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read TTS response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TTS API error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var ttsResp TTSResponse
	if err := json.Unmarshal(body, &ttsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TTS response: %v", err)
	}

	// Decode base64 audio content
	audioData, err := base64.StdEncoding.DecodeString(ttsResp.AudioContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio content: %v", err)
	}

	return audioData, nil
}

// SynthesizeTextStream provides streaming TTS (for future implementation)
func (g *GoogleTTSClient) SynthesizeTextStream(text string, w io.Writer) error {
	// For now, we'll use the regular synthesize and write to the stream
	// Google Cloud TTS doesn't support streaming synthesis in the same way as some other services
	audioData, err := g.SynthesizeText(text)
	if err != nil {
		return err
	}

	_, err = w.Write(audioData)
	return err
}

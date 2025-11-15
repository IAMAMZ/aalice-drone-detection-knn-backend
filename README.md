# AALIS - Acoustic Autonomous Lightweight Interception System

> Advanced realtime acoustic detection and threat assessment system for autonomous drone interception.

## Overview

This application analyzes audio recordings to detect and classify drone acoustic signatures using signal processing and machine learning techniques. It captures audio from a microphone or system audio, processes it through a feature extraction pipeline, and compares it against a library of known drone prototypes using a k-nearest neighbors (KNN) classifier.

## How It Works

### High-Level Flow

1. **Audio Capture**: Browser captures up to 20 seconds of audio (microphone or system audio)
2. **Audio Preprocessing**: Server applies filters and noise reduction (high-pass, band-pass, AGC)
3. **SNR Estimation**: Signal-to-noise ratio is calculated for adaptive threshold adjustment
4. **Audio Processing**: Convert raw audio to normalized PCM samples
5. **Feature Extraction**: Extract 11 spectral and temporal features from the audio
6. **Classification**: Compare features against prototype library using KNN algorithm with adaptive K
7. **Adaptive Thresholding**: Confidence threshold adjusted based on SNR
8. **Results**: Return ranked predictions with confidence scores and SNR information

### Detailed Architecture

```
┌─────────────────┐
│   Browser UI    │  React frontend with audio recording
└────────┬────────┘
         │ WebSocket / HTTP POST
         │ (base64 encoded WAV)
         ▼
┌─────────────────┐
│  Go Backend     │
│  ┌───────────┐  │
│  │ Audio     │  │  Decode base64 → WAV file → PCM samples
│  │ Processing│  │
│  └─────┬─────┘  │
│        │        │
│  ┌─────▼─────┐  │
│  │ Feature   │  │  Extract 11 features (spectral + temporal)
│  │ Extraction│  │
│  └─────┬─────┘  │
│        │        │
│  ┌─────▼─────┐  │
│  │ Classifier │  │  KNN comparison against prototypes
│  └─────┬─────┘  │
└────────┼────────┘
         │ JSON response
         ▼
┌─────────────────┐
│  Results Panel  │  Display predictions with confidence
└─────────────────┘
```

## Audio Processing Pipeline

### 1. Client-Side Audio Capture (`client/src/App.js`)

**Recording Process:**
- User clicks "Start Listening" button
- Browser requests microphone or system audio permission
- `MediaRecorder` API captures audio stream
- Audio is encoded as WAV format using `extendable-media-recorder-wav-encoder`
- Recording automatically stops after 20 seconds

**Audio Format Conversion:**
- Raw audio blob is processed through FFmpeg (via `@ffmpeg/ffmpeg`)
- Converted to mono channel (required for analysis)
- Sample rate normalized to 44.1 kHz
- 16-bit PCM samples

**Data Transmission:**
- WAV file is converted to base64 string
- Sent to server via HTTP POST to `/api/audio/classify`
- Payload includes: `{ audio, channels, sampleRate, sampleSize, duration }`

### 2. Server-Side Audio Processing (`server/drone/audio.go`)

**Pipeline Steps:**

1. **Base64 Decoding**: Decode the base64 audio string back to binary WAV data
2. **WAV File Creation**: Write decoded data to temporary WAV file
3. **Audio Reformatting**: Convert to mono channel using `wav.ReformatWAV()`
4. **SNR Estimation**: Calculate signal-to-noise ratio from first 10% of audio
5. **Audio Preprocessing** (`server/drone/preprocessing.go`):
   - **High-pass filter**: Removes frequencies below 50 Hz (low-frequency noise)
   - **Band-pass filter**: Focuses on 100-5000 Hz range (drone frequencies)
   - **Automatic Gain Control (AGC)**: Normalizes audio levels to target RMS
   - **Simple noise reduction**: Basic spectral subtraction (optional)
6. **Sample Extraction**: Extract PCM samples as `[]float64` array
7. **Duration Calculation**: Compute duration from sample count and sample rate

**Output**: `AudioSample` struct containing:
- `Samples []float64`: Preprocessed PCM samples (-1.0 to 1.0)
- `SampleRate int`: Samples per second (typically 44100)
- `Duration float64`: Length in seconds
- `SNRDb float64`: Estimated signal-to-noise ratio in decibels

## Feature Extraction

### Overview (`server/drone/features.go`)

The system extracts **11 features** that capture the acoustic signature of drone propellers:

### Temporal Features (Time Domain)

1. **Energy (RMS)** - Root Mean Square amplitude
   - Formula: `√(Σ(sample²) / N)`
   - Measures overall signal strength
   - Drones typically have higher energy than background noise

2. **Zero Crossing Rate (ZCR)** - Frequency of sign changes
   - Counts transitions from positive to negative (or vice versa)
   - Formula: `count / (N - 1)`
   - Indicates pitch characteristics and noise level

3. **Variance** - Signal variability
   - Formula: `Σ((sample - mean)²) / N`
   - Helps distinguish steady tones from random noise

### Spectral Features (Frequency Domain)

These features are derived from the **Fast Fourier Transform (FFT)**:

4. **Spectral Centroid** - "Brightness" of sound
   - Weighted average frequency: `Σ(magnitude[i] × freq[i]) / Σ(magnitude[i])`
   - Higher values indicate more high-frequency content
   - Drones have distinct centroid values due to rotor harmonics

5. **Spectral Bandwidth** - Spread of frequencies around centroid
   - Formula: `√(Σ(magnitude[i] × (freq[i] - centroid)²) / Σ(magnitude[i]))`
   - Measures frequency distribution width

6. **Spectral Rolloff** - Frequency below which 85% of energy is contained
   - Cumulative energy threshold calculation
   - Indicates where most spectral energy is concentrated

7. **Spectral Flatness** - Ratio of geometric to arithmetic mean
   - Formula: `(Π(magnitude[i]))^(1/N) / (Σ(magnitude[i]) / N)`
   - Measures noisiness (1.0 = white noise, <1.0 = tonal content)
   - Drones have lower flatness due to harmonic structure

8. **Spectral Crest Factor** - Peak-to-average ratio
   - Formula: `max(magnitude) / mean(magnitude)`
   - Indicates tonal vs noisy content
   - Higher values suggest distinct frequency peaks

9. **Spectral Entropy** - Randomness in frequency distribution
   - Formula: `-Σ(p[i] × log₂(p[i])) / log₂(N)` where `p[i] = magnitude[i]² / Σ(magnitude²)`
   - Measures predictability of frequency content

10. **Dominant Frequency** - Frequency bin with maximum magnitude
    - Simply finds the peak frequency component
    - Often corresponds to rotor blade passing frequency

### Feature Extraction Process

1. **FFT Computation**: Convert time-domain signal to frequency domain
   - Uses Cooley-Tukey radix-2 FFT algorithm
   - Window size: next power of 2 ≥ sample length
   - Applies Hann window to reduce spectral leakage

2. **Magnitude Spectrum**: Extract magnitude from complex FFT results
   - `magnitude = |FFT(samples)|`
   - Creates frequency bins representing different frequency ranges

3. **Feature Calculation**: Compute each feature from magnitude spectrum
   - All features are calculated from the same FFT output
   - Ensures consistency across different audio lengths

4. **Normalization**: Vector is normalized to unit length (L2 normalization)
   - Formula: `vector / ||vector||`
   - Makes distance calculations scale-invariant

**Output**: 11-dimensional feature vector `[]float64` ready for classification

## Classification Algorithm

### K-Nearest Neighbors (KNN) Classifier (`server/drone/classifier.go`)

### Prototype Storage

- **Prototypes**: Pre-computed feature vectors from labeled audio samples
- Each prototype contains:
  - `ID`: Unique identifier
  - `Label`: Class name (e.g., "dji_mavic", "urban_noise")
  - `Category`: High-level category ("drone" or "noise")
  - `Features`: 11-dimensional normalized feature vector
  - `Metadata`: Additional information (model, rotor count, etc.)

### Classification Process

1. **Feature Extraction**: Extract 11 features from input audio (same as prototypes)

2. **Normalization**: Normalize feature vector to unit length

3. **Distance Calculation**: Compute Euclidean distance to all prototypes
   ```
   distance = √(Σ(input[i] - prototype[i])²)
   ```

4. **K-Nearest Selection**: Select K closest prototypes (default K=5)
   - Sorted by distance (closest first)

5. **Weight Calculation**: Assign weights to each neighbor
   ```
   weight = 1 / (distance + ε)
   ```
   - Closer prototypes have higher weight
   - ε (epsilon) = 1e-9 prevents division by zero

6. **Label Aggregation**: For each unique label:
   - Sum weights from matching prototypes
   - Count number of supporting prototypes
   - Calculate average distance

7. **Confidence Calculation**: 
   ```
   confidence = sum(weights for label) / sum(all weights)
   ```
   - Represents probability that input belongs to this label
   - Values range from 0.0 to 1.0

8. **Prediction Ranking**: Sort predictions by:
   - Primary: Confidence (descending)
   - Secondary: Average distance (ascending)

### Drone Detection Logic

The system determines if audio likely contains a drone:

```go
// Base threshold from environment (default 0.55)
baseThreshold := 0.55

// Adaptive threshold based on SNR
adjustedThreshold := AdaptiveThreshold(baseThreshold, snrDb)

// Detection logic
isDrone = (top_prediction.confidence >= adjustedThreshold) && 
          (top_prediction.category != "noise")
```

**Adaptive Threshold Adjustment:**
- **SNR < 10 dB** (very noisy): Threshold +0.15 (more conservative)
- **SNR 10-20 dB** (moderate noise): Threshold +0.10
- **SNR 20-30 dB** (good SNR): Threshold +0.05
- **SNR > 30 dB** (excellent): Uses base threshold

**Adaptive K Selection:**
- If K > prototype count: K is automatically set to prototype count
- If <10 prototypes: Uses K=3 for better reliability with small datasets
- Prevents overfitting and improves accuracy with limited training data

## Audio Fingerprinting (Background)

The system also includes a Shazam-like fingerprinting algorithm (`server/shazam/`) used for pattern matching:

### Spectrogram Generation (`server/shazam/spectrogram.go`)

1. **Preprocessing**:
   - Low-pass filter: Remove frequencies above 5kHz
   - Downsampling: Reduce sample rate by factor of 4

2. **Short-Time Fourier Transform (STFT)**:
   - Divide audio into overlapping windows (1024 samples)
   - Apply Hamming window function
   - Compute FFT for each window
   - Result: 2D array `[time_window][frequency_bin]`

### Peak Extraction (`server/shazam/spectrogram.go`)

- Analyze spectrogram to find frequency peaks (local maxima)
- Extract peaks in frequency bands: [0-10], [10-20], [20-40], [40-80], [80-160], [160-512]
- Filter peaks above average magnitude in each band

### Fingerprinting (`server/shazam/fingerprint.go`)

- For each anchor peak, create pairs with nearby peaks (within 5 peaks)
- Generate 32-bit hash address:
  - Bits 23-31: Anchor frequency (9 bits)
  - Bits 14-22: Target frequency (9 bits)
  - Bits 0-13: Time delta in milliseconds (14 bits)
- Store mapping: `address → (anchor_time_ms, song_id)`

**Note**: This fingerprinting system is available but the primary classification uses the feature-based KNN approach described above.

## API Endpoints

### HTTP Endpoints

#### `POST /api/audio/classify`

Classify audio sample and return predictions.

**Request Body:**
```json
{
  "audio": "base64_encoded_wav_data",
  "channels": 1,
  "sampleRate": 44100,
  "sampleSize": 16,
  "duration": 20.0
}
```

**Response:**
```json
{
  "predictions": [
    {
      "label": "dji_mavic",
      "category": "drone",
      "type": "DJI Mavic 2 Pro",
      "description": "DJI Mavic 2 Pro quadcopter",
      "confidence": 0.85,
      "averageDistance": 0.12,
      "support": 3,
      "topPrototypes": [
        {
          "id": "proto_dji_mavic_01",
          "distance": 0.10,
          "weight": 9.09
        }
      ],
      "metadata": {
        "model": "DJI Mavic 2 Pro",
        "rotor_count": "4",
        "threat_level": "medium",
        "payload_capacity_kg": "2.5"
      },
      "threatAssessment": {
        "threatLevel": "medium",
        "riskCategory": "surveillance",
        "payloadCapacityKg": 2.5,
        "maxRangeKm": 8,
        "maxSpeedMs": 20,
        "flightTimeMinutes": 31,
        "jammingSusceptible": true,
        "countermeasureRecommendations": "RF jamming, GPS spoofing",
        "operatorType": "commercial"
      }
    }
  ],
  "isDrone": true,
  "latencyMs": 45.2,
  "featureVector": [0.58, 0.04, 0.32, ...],
  "primaryType": "DJI Mavic 2 Pro",
  "snrDb": 25.3,
  "adjustedThreshold": 0.60
}
```

**New Response Fields:**
- `snrDb`: Signal-to-noise ratio in decibels (estimated from audio)
- `adjustedThreshold`: Actual confidence threshold used (adjusted based on SNR)
- `threatAssessment`: Defense-focused intelligence object (only for drone predictions)
  - `threatLevel`: Threat assessment (low/medium/high/critical)
  - `riskCategory`: Primary threat category (surveillance/payload_delivery/etc.)
  - `payloadCapacityKg`: Maximum payload weight
  - `maxRangeKm`: Operational range
  - `maxSpeedMs`: Maximum speed
  - `flightTimeMinutes`: Flight endurance
  - `jammingSusceptible`: Vulnerability to RF jamming
  - `countermeasureRecommendations`: Recommended countermeasures
  - `operatorType`: Typical operator type

#### `POST /api/prototypes/upload`

Upload new prototype audio samples to expand the classifier library.

**Request**: Multipart form data
- `samples`: Audio file(s) (WAV format)
- `label`: Class label (required)
- `category`: Category ("drone" or "noise", default: "drone")
- `description`: Optional description
- `model`, `type`, `rotor_count`, `manufacturer`: Optional metadata fields

**Defense-Focused Fields** :
- `threat_level`: Threat assessment ("low", "medium", "high", "critical")
- `risk_category`: Risk category ("surveillance", "payload_delivery", "reconnaissance", "swarm", "commercial", "hobbyist")
- `payload_capacity_kg`: Maximum payload weight in kilograms
- `max_range_km`: Maximum operational range in kilometers
- `max_speed_ms`: Maximum speed in meters per second
- `flight_time_minutes`: Typical flight endurance in minutes
- `jamming_susceptible`: Whether vulnerable to RF jamming ("true" or "false")
- `countermeasure_recommendations`: Recommended countermeasures (comma-separated)
- Additional fields: `max_altitude_m`, `weight_kg`, `has_gps`, `has_camera`, `has_autonomous_flight`, `swarm_capable`, `operator_type`, `typical_use_cases`, `detection_range_m`

See [`DEFENSE_METADATA_FIELDS.md`](DEFENSE_METADATA_FIELDS.md) for complete list of defense-focused metadata fields.

**Response:**
```json
{
  "added": [
    {
      "id": "proto_new_01",
      "label": "custom_drone",
      "features": [...],
      ...
    }
  ],
  "stats": {
    "prototypeCount": 7,
    "labelCount": 4,
    "labels": [...],
    "usingExample": false
  }
}
```

### WebSocket Events

#### Client → Server

- **`requestModelInfo`**: Request current model statistics
- **`newRecording`**: Send audio data (legacy, now uses HTTP POST)

#### Server → Client

- **`modelInfo`**: Model statistics (`{ prototypeCount, labelCount, labels[], usingExample }`)
- **`classification`**: Classification results (same format as HTTP response)
- **`analysisError`**: Error message (`{ message: "..." }`)

## Frontend Components

### Main App (`client/src/App.js`)

- Manages WebSocket connection to server
- Handles audio recording via MediaRecorder API
- Sends audio to server via HTTP POST
- Displays classification results

### Prototype Upload (`client/src/components/PrototypeUpload.js`)

- Form for uploading new audio samples
- Fields: label, category, description, metadata
- Uploads via multipart form to `/api/prototypes/upload`
- Updates model statistics after successful upload

### Results Panel (`client/src/components/ResultsPanel.js`)

- Displays classification predictions
- Shows confidence scores, support counts, distances
- Highlights drone vs noise predictions
- Shows metadata and top matching prototypes

### Listen Component (`client/src/components/Listen.js`)

- Start/stop recording button
- Visual feedback during recording
- Handles microphone/system audio selection

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DRONE_MODEL_PATH` | `drone/prototypes.json` | Path to prototype JSON file |
| `DRONE_MODEL_K` | `5` | Number of nearest neighbors for KNN |
| `DRONE_PERSIST_RECORDINGS` | `true` | Enable/disable saving processed audio |
| `DRONE_RECORDING_DIR` | `frontendrecording` | Directory for persisted captures |
| `DRONE_TEMPLATE_PATH` | *(empty)* | Optional few-shot template bank (`server/drone/templates.json`) |
| `DRONE_TEMPLATE_THRESHOLD` | `0.75` | Minimum confidence for template predictions |

When template matching is enabled, each response also includes `templatePredictions` so you can inspect the direct similarities against your reference clips.

Every classification response now includes a `recordingPath` (when available) so you can verify which WAV file was captured for that detection.

### Prototype File Format
### Few-Shot Template Builder

If you only have a handful of labelled recordings (for example the four files under `server/train_data`), you can convert them into a template bank that the classifier will consult alongside the regular KNN prototypes:

```bash
cd server
go run ./cmd/template_tool -dir ./train_data -out ./drone/templates.json
export DRONE_TEMPLATE_PATH=server/drone/templates.json
export DRONE_TEMPLATE_THRESHOLD=0.7   # optional tweak
```

This approach stores one normalized feature vector per recording and performs cosine-similarity lookups during classification. It’s ideal for extremely small datasets or quick field calibration sessions.

### Mock Frontend Uploader

Need to simulate the browser client or quickly drop a batch of WAV recordings into the classifier (and confirm they were saved under `frontendrecording/`)? Use the helper CLI:

```bash
cd server
# send every WAV in train_data/
go run ./cmd/mock_frontend -dir ./train_data \
  -url http://localhost:5000/api/audio/classify

# send a single file with explicit coordinates
go run ./cmd/mock_frontend -file ./train_data/A_04.wav \
  -lat 37.7749 -lon -122.4194
```

Each upload goes through the exact HTTP endpoint the frontend uses, so the backend persists the audio (to `DRONE_RECORDING_DIR`, default `frontendrecording/`) and returns the usual classification payload for UI testing.

Prototypes are stored as JSON array:

```json
[
  {
    "id": "proto_dji_mavic_01",
    "label": "dji_mavic",
    "category": "drone",
    "description": "DJI Mavic 2 Pro quadcopter",
    "source": "dataset/dji_mavic/clip_01.wav",
    "features": [0.58, 0.04, 0.32, 0.21, 0.27, 0.36, 0.31, 0.72, 0.54, 0.12],
    "metadata": {
      "model": "DJI Mavic 2 Pro",
      "rotor_count": "4"
    }
  }
]
```

## Getting Started

### Prerequisites

- Go 1.21+
- Node 18+
- FFmpeg available on PATH
- (Optional) Python 3.10+ for dataset tooling

#### Installing FFmpeg

**Windows:**
1. Download FFmpeg from [https://ffmpeg.org/download.html](https://ffmpeg.org/download.html) or use a package manager:
   - **Using Chocolatey** (recommended):
     ```cmd
     choco install ffmpeg
     ```
   - **Using Scoop**:
     ```cmd
     scoop install ffmpeg
     ```
   - **Manual installation**:
     - Download the Windows build from [https://www.gyan.dev/ffmpeg/builds/](https://www.gyan.dev/ffmpeg/builds/)
     - Extract the ZIP file to a folder (e.g., `C:\ffmpeg`)
     - Add `C:\ffmpeg\bin` to your system PATH:
       - Open System Properties → Environment Variables
       - Edit the `Path` variable and add `C:\ffmpeg\bin`
       - Restart your terminal/PowerShell
     - Verify installation: `ffmpeg -version`

**macOS:**
```bash
brew install ffmpeg
```

**Linux:**
```bash
sudo apt-get update && sudo apt-get install ffmpeg
```

### 1. Setup Prototypes

Copy the example prototypes file:

```bash
cp server/drone/prototypes.example.json server/drone/prototypes.json
```

Or generate your own using the Python tooling:

```bash
cd ml
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python build_prototypes.py \
  --input /path/to/wav/dataset \
  --output ../server/drone/prototypes.json \
  --category-map category-map.yaml
```

### 2. Run Backend

**Unix/Linux/macOS:**
```bash
cd server
go run *.go serve -proto http -p 5000
```

**Windows:**
```cmd
cd server
go run . serve -proto http -p 5000
```

Note: On Windows, use `go run .` instead of `go run *.go` because Windows shells don't expand wildcards the same way.

### 3. Run Frontend

```bash
cd client
npm install
npm start
```

Visit `http://localhost:3000` and grant microphone permissions when prompted.

### 4. Docker (Alternative)

```bash
docker compose up --build
```

Access dashboard at `http://localhost:8080`

## Usage

1. **Start Listening**: Click the microphone button to begin recording
2. **Select Audio Source**: Choose microphone or system audio (desktop only)
3. **Record**: System automatically records for 20 seconds
4. **View Results**: Classification results appear in the results panel
5. **Upload Prototypes**: Use the upload form to add new drone signatures

## Technical Details

### Why These Features Work

Drone propellers produce distinct acoustic signatures:

- **Harmonic Content**: Rotor blades create periodic frequency patterns
- **Motor RPM**: Electric motors generate specific frequency components
- **Blade Passing Frequency**: `RPM × blade_count / 60` Hz
- **Resonance**: Propeller and frame resonances add characteristic frequencies

The 11 features capture these patterns:
- Spectral features identify frequency content and harmonics
- Temporal features capture amplitude and variability patterns
- Combined, they create a unique "fingerprint" for each drone type

### Performance Characteristics

- **Latency**: Typically 40-100ms for classification
- **Accuracy**: Depends on prototype library quality and diversity
- **Scalability**: KNN search is O(n) where n = number of prototypes
- **Memory**: All prototypes loaded in memory for fast access

## Offline Tooling

See [`ml/README.md`](ml/README.md) for guidance on building prototype libraries from raw WAV recordings. The Python tooling automatically:
- Extracts the same 11 features
- Normalizes feature vectors (L2 normalization)
- Exports to JSON format compatible with Go classifier

## License

MIT © 2025

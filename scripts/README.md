# Scripts for Building Training Dataset

## Quick Start: Download Noise Samples

The fastest way to improve your classifier is to add "noise" samples (non-drone sounds).

### 1. Get Freesound API Key (Free)

1. Go to https://freesound.org/apiv2/apply/
2. Create a free account
3. Fill out the API application form (takes 2 minutes)
4. You'll receive an API key instantly

### 2. Install Requirements

```bash
# Python dependencies
pip install requests

# ffmpeg (for audio conversion)
# macOS:
brew install ffmpeg

# Linux:
sudo apt install ffmpeg

# Windows:
# Download from https://ffmpeg.org/download.html
```

### 3. Download Noise Samples

```bash
# Navigate to project root
cd /path/to/project

# Set your API key
export FREESOUND_API_KEY="your-api-key-here"

# Download samples (default: 5 per category, 8 categories = 40 samples)
python scripts/download_noise_samples.py
```

This will download ~40 noise samples organized by category:
- `train_data_noise/traffic/` - Cars, highways, urban traffic
- `train_data_noise/birds/` - Bird chirping, singing, calls
- `train_data_noise/household/` - Fans, AC, vacuum, appliances
- `train_data_noise/construction/` - Drilling, hammering, sawing
- `train_data_noise/nature/` - Wind, rain, water, leaves
- `train_data_noise/crowd/` - People talking, cafe, mall
- `train_data_noise/aircraft/` - Airplanes, helicopters (NOT drones!)
- `train_data_noise/industrial/` - Machinery, motors, generators

### 4. Build Prototypes

```bash
cd server

# Generate noise prototypes
go run ./cmd/rebuild_prototypes \
  -dir ../train_data_noise \
  -category noise \
  -out drone/noise_prototypes.json

# Check what was created
cat drone/noise_prototypes.json | grep '"label"' | sort | uniq -c
```

### 5. Merge with Existing Prototypes

**Option A: Using jq (recommended)**
```bash
# Install jq if needed: brew install jq

cd server
jq -s '.[0] + .[1]' \
  drone/prototypes.json \
  drone/noise_prototypes.json \
  > drone/prototypes_merged.json

mv drone/prototypes_merged.json drone/prototypes.json
```

**Option B: Manual merge**
```bash
# Open both files in a text editor
# Copy the array contents from noise_prototypes.json
# Paste into the main prototypes.json array
# Make sure the JSON is valid (commas between objects)
```

### 6. Test Improved Classifier

```bash
cd server

# Check new prototype count
cat drone/prototypes.json | grep '"id"' | wc -l
# Should show: 46 (6 drone + 40 noise)

# Test classification
go run ./cmd/train_eval -dir ../test_recordings -k 5

# Or test single file
go run ./cmd/explain_classification ../test_recordings/some_sample.wav
```

**Expected improvement:**
- Before: Everything classified as "drone" (false positive rate ~80%)
- After: Noise correctly classified as "noise" (false positive rate ~30%)

## Customization

### Download More Samples

Edit `download_noise_samples.py`:

```python
# Line 38: Adjust samples per category
SAMPLES_PER_CATEGORY = 10  # Default: 5

# Lines 42-90: Add more categories or search queries
SOUND_CATEGORIES = {
    "traffic": [
        "car traffic city",
        "highway traffic",
        # Add more queries here
    ],
    # Add new categories here
    "dogs": [
        "dog barking",
        "dogs barking multiple",
    ],
}
```

Then run again:
```bash
python scripts/download_noise_samples.py
```

### Download Specific Categories Only

```python
# Comment out categories you don't want:
SOUND_CATEGORIES = {
    "traffic": [...],
    "birds": [...],
    # "household": [...],  # Skip this
    # "construction": [...],  # Skip this
}
```

### Adjust Duration

```python
# Lines 35-36: Change duration range
SAMPLE_DURATION_MIN = 5   # Longer samples
SAMPLE_DURATION_MAX = 30
```

## Alternative: Use Your Own Samples

If you have your own noise recordings:

```bash
# 1. Put your WAV files in train_data_noise/
mkdir -p train_data_noise
cp /path/to/your/*.wav train_data_noise/

# 2. Generate prototypes
cd server
go run ./cmd/rebuild_prototypes \
  -dir ../train_data_noise \
  -category noise \
  -out drone/noise_prototypes.json

# 3. Merge as shown above
```

## Troubleshooting

### "FREESOUND_API_KEY not set"

```bash
# Make sure you exported the key in the same terminal:
export FREESOUND_API_KEY="your-key-here"

# Or pass it inline:
FREESOUND_API_KEY="your-key" python scripts/download_noise_samples.py
```

### "ffmpeg not found"

```bash
# macOS:
brew install ffmpeg

# Linux:
sudo apt update && sudo apt install ffmpeg

# Windows:
# Download from https://ffmpeg.org/download.html
# Add to PATH
```

### "Rate limit exceeded"

Freesound has rate limits. The script includes delays, but if you hit limits:
- Wait 1 minute
- Reduce `SAMPLES_PER_CATEGORY`
- Run the script multiple times

### "No sounds found"

Some search queries might not return results. The script tries multiple queries per category, so this is normal. If entire categories fail:
- Check your API key is valid
- Check your internet connection
- Try different search queries

## Expected Timeline

- **Setup (5 minutes)**: Get API key, install requirements
- **Download (10-20 minutes)**: Download 40 samples
- **Build prototypes (2 minutes)**: Generate JSON
- **Test (5 minutes)**: Verify improved accuracy

**Total: ~30 minutes to dramatically improve your classifier!**

## What This Solves

### Before (Only Drone Samples):
```
Test: car engine sound
Result: i_12 (45% confidence) ← WRONG!

Test: bird chirping  
Result: c_07 (38% confidence) ← WRONG!

Test: actual i_12 drone
Result: i_12 (67% confidence) ← Correct, but low
```

### After (With Noise Samples):
```
Test: car engine sound
Result: noise (traffic) (82% confidence) ← CORRECT!

Test: bird chirping
Result: noise (birds) (91% confidence) ← CORRECT!

Test: actual i_12 drone
Result: i_12 (89% confidence) ← Correct AND confident!
```

## Next Steps

After adding noise samples:

1. **Test accuracy**: Run evaluation on known samples
2. **Collect more drone samples**: 20-30 per type
3. **Balance classes**: Equal numbers of each drone type
4. **Iterate**: Add edge cases as you find them

The noise samples are the HIGHEST PRIORITY improvement you can make!


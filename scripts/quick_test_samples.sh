#!/bin/bash
#
# Quick test: Generate synthetic noise samples for testing
# This is a FALLBACK if you can't use Freesound API
# These are simple synthetic sounds, not as good as real recordings
#

set -e

OUTPUT_DIR="train_data_noise_synthetic"
DURATION=5  # seconds

echo "================================================"
echo "Generating Synthetic Noise Samples"
echo "================================================"
echo ""
echo "⚠️  WARNING: These are synthetic sounds for TESTING only."
echo "    For production, use real recordings or Freesound API."
echo ""

# Check for ffmpeg
if ! command -v ffmpeg &> /dev/null; then
    echo "❌ ERROR: ffmpeg not found"
    echo ""
    echo "Install ffmpeg:"
    echo "  macOS:   brew install ffmpeg"
    echo "  Linux:   sudo apt install ffmpeg"
    exit 1
fi

# Check for sox (optional, better for synthetic sounds)
HAS_SOX=false
if command -v sox &> /dev/null; then
    HAS_SOX=true
    echo "✅ Found sox - will generate better quality synthetic sounds"
else
    echo "⚠️  sox not found - using basic ffmpeg generation"
    echo "   For better quality: brew install sox (macOS) or apt install sox (Linux)"
fi

echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo "📁 Output directory: $OUTPUT_DIR"
echo "⏱️  Duration: ${DURATION}s per sample"
echo ""

# Generate samples
SAMPLE_COUNT=0

# Traffic-like noise (pink noise with low-frequency emphasis)
echo "🚗 Generating traffic noise..."
if [ "$HAS_SOX" = true ]; then
    sox -n -r 44100 -c 1 "$OUTPUT_DIR/traffic_01_highway.wav" synth $DURATION pinknoise band 100 2000 gain -3
    sox -n -r 44100 -c 1 "$OUTPUT_DIR/traffic_02_city.wav" synth $DURATION pinknoise band 80 1500 tremolo 0.3 gain -3
    ((SAMPLE_COUNT+=2))
else
    ffmpeg -f lavfi -i "anoisesrc=d=${DURATION}:c=pink:r=44100:a=0.5" -ac 1 "$OUTPUT_DIR/traffic_01.wav" -y -loglevel error
    ((SAMPLE_COUNT+=1))
fi

# Birds (high-frequency chirps)
echo "🐦 Generating bird sounds..."
if [ "$HAS_SOX" = true ]; then
    sox -n -r 44100 -c 1 "$OUTPUT_DIR/birds_01_chirping.wav" synth $DURATION sine 2000-4000 tremolo 5 gain -5
    sox -n -r 44100 -c 1 "$OUTPUT_DIR/birds_02_singing.wav" synth $DURATION sine 1500-3500 tremolo 8 gain -6
    ((SAMPLE_COUNT+=2))
else
    ffmpeg -f lavfi -i "sine=frequency=2500:duration=${DURATION}:sample_rate=44100" -ac 1 "$OUTPUT_DIR/birds_01.wav" -y -loglevel error
    ((SAMPLE_COUNT+=1))
fi

# Household appliances (low hum)
echo "🏠 Generating household noise..."
if [ "$HAS_SOX" = true ]; then
    sox -n -r 44100 -c 1 "$OUTPUT_DIR/household_01_fan.wav" synth $DURATION pinknoise band 60 500 gain -5
    sox -n -r 44100 -c 1 "$OUTPUT_DIR/household_02_ac.wav" synth $DURATION sine 120 sine add 240 pinknoise band 100 800 gain -6
    ((SAMPLE_COUNT+=2))
else
    ffmpeg -f lavfi -i "sine=frequency=120:duration=${DURATION}:sample_rate=44100" -af "volume=0.3" -ac 1 "$OUTPUT_DIR/household_01.wav" -y -loglevel error
    ((SAMPLE_COUNT+=1))
fi

# Wind/nature (brown noise)
echo "🌳 Generating nature sounds..."
if [ "$HAS_SOX" = true ]; then
    sox -n -r 44100 -c 1 "$OUTPUT_DIR/nature_01_wind.wav" synth $DURATION brownnoise gain -3
    sox -n -r 44100 -c 1 "$OUTPUT_DIR/nature_02_rain.wav" synth $DURATION whitenoise band 500 5000 gain -8
    ((SAMPLE_COUNT+=2))
else
    ffmpeg -f lavfi -i "anoisesrc=d=${DURATION}:c=brown:r=44100:a=0.5" -ac 1 "$OUTPUT_DIR/nature_01.wav" -y -loglevel error
    ((SAMPLE_COUNT+=1))
fi

# Crowd (white noise with variations)
echo "👥 Generating crowd noise..."
if [ "$HAS_SOX" = true ]; then
    sox -n -r 44100 -c 1 "$OUTPUT_DIR/crowd_01_talking.wav" synth $DURATION whitenoise band 200 3000 tremolo 0.5 gain -5
    ((SAMPLE_COUNT+=1))
else
    ffmpeg -f lavfi -i "anoisesrc=d=${DURATION}:c=white:r=44100:a=0.3" -ac 1 "$OUTPUT_DIR/crowd_01.wav" -y -loglevel error
    ((SAMPLE_COUNT+=1))
fi

echo ""
echo "================================================"
echo "✅ Generated $SAMPLE_COUNT synthetic samples"
echo "================================================"
echo ""
echo "📁 Saved to: $OUTPUT_DIR/"
echo ""
echo "🎯 Next Steps:"
echo ""
echo "1. Build prototypes:"
echo "   cd server"
echo "   go run ./cmd/rebuild_prototypes \\"
echo "     -dir ../$OUTPUT_DIR \\"
echo "     -category noise \\"
echo "     -out drone/noise_prototypes.json"
echo ""
echo "2. Merge with existing:"
echo "   jq -s '.[0] + .[1]' \\"
echo "     drone/prototypes.json \\"
echo "     drone/noise_prototypes.json \\"
echo "     > drone/prototypes_merged.json"
echo "   mv drone/prototypes_merged.json drone/prototypes.json"
echo ""
echo "3. Test:"
echo "   go run ./cmd/train_eval -dir test_data -k 5"
echo ""
echo "⚠️  IMPORTANT: These are SYNTHETIC samples for testing."
echo "    For production use, download real recordings:"
echo "    python scripts/download_noise_samples.py"
echo ""


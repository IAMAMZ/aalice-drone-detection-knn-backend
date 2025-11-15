# Setup Guide - Drone Detection System

## Prerequisites

### Required Software:
1. **Go** (version 1.19+)
   - Download: https://go.dev/dl/
   - Verify: `go version`

2. **Python** (version 3.8+)
   - Download: https://www.python.org/downloads/
   - Verify: `python3 --version`

3. **FFmpeg** (for audio processing)
   - **macOS:** `brew install ffmpeg`
   - **Linux:** `sudo apt-get install ffmpeg`
   - **Windows:** Download from https://ffmpeg.org/download.html
   - Verify: `ffmpeg -version`

4. **Git** (to clone the repository)
   - Verify: `git --version`

---

## Step 1: Get the Code

```bash
# Option A: If you have the code already, just copy the whole folder
cp -r /path/to/drone-detection /new/location

# Option B: If using git
git clone <your-repo-url> drone-detection
cd drone-detection
```

---

## Step 2: Install Python Dependencies

```bash
cd ml

# Create virtual environment (recommended)
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install PANNS dependencies
pip install -r requirements-panns.txt

# This will install:
# - torch (PyTorch for the neural network)
# - librosa (audio processing)
# - numpy (numerical operations)
# - flask (web API)
# - audioset-tagging-cnn (PANNS model)
```

**Note:** First run will download the PANNS model (~327MB). This is automatic but takes a few minutes.

---

## Step 3: Install Go Dependencies

```bash
cd ../server

# Download Go dependencies
go mod download

# Verify everything compiles
go build
```

---

## Step 4: Verify Training Data

Make sure the training data is in place:

```bash
cd ..
ls Drone-Training-Data/

# Should see:
# drone_A/  drone_B/  drone_C/  drone_D/  drone_E/
# drone_F/  drone_G/  drone_H/  drone_I/  drone_J/
```

**Important:** The `server/drone/prototypes.json` file should already contain 150 PANNS embeddings (150 prototypes × 2048 dimensions each). If this file is missing or outdated, you'll need to rebuild it (see Troubleshooting section).

---

## Step 5: Run the System

### Terminal 1: Start PANNS Embedding Service

```bash
cd ml

# If using virtual environment
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Start the service (runs on port 5002)
python embedding_service.py
```

You should see:
```
Loading PANNS model...
PANNS model loaded successfully!
 * Running on http://127.0.0.1:5002
```

**Keep this terminal running!**

---

### Terminal 2: Start Go Server

```bash
cd server

# Start with PANNS enabled
USE_PANNS_EMBEDDINGS=true EMBEDDING_SERVICE_URL=http://localhost:5002 \
  go run . serve -proto http -p 5001
```

**On Windows (PowerShell):**
```powershell
$env:USE_PANNS_EMBEDDINGS="true"
$env:EMBEDDING_SERVICE_URL="http://localhost:5002"
go run . serve -proto http -p 5001
```

You should see:
```
FFmpeg is available
Starting HTTP server on port 5001
```

**Keep this terminal running!**

---

## Step 6: Open the Web Interface

Open your browser and go to:
```
http://localhost:5001
```

You should see the drone detection interface!

---

## Step 7: Test It Works

### Option A: Test with training data
```bash
# In a new terminal
cd server
go run ./cmd/mock_frontend \
  -file ../Drone-Training-Data/drone_A/A_01.wav \
  -url http://localhost:5001/api/audio/classify
```

Expected output:
```
→ A_01.wav
   best=drone a (90-96%) adjustedThreshold=0.70
```

### Option B: Test via web UI
1. Click "Upload Audio File" button
2. Select any `.wav` file from `Drone-Training-Data/`
3. Should see classification result with high confidence (80-100%)

### Option C: Test with microphone
1. Click "Start Detection" button
2. Make some noise or play a drone audio file
3. Should see real-time classification

---

## Troubleshooting

### Problem: "PANNS service not available"

**Cause:** The Python PANNS service isn't running or can't be reached.

**Solution:**
1. Check Terminal 1 - is `embedding_service.py` still running?
2. Test the service directly:
   ```bash
   curl http://localhost:5002/health
   # Should return: {"status": "ok"}
   ```
3. Check firewall - make sure port 5002 is not blocked

---

### Problem: "Module not found" errors in Python

**Solution:**
```bash
cd ml
pip install -r requirements-panns.txt --upgrade
```

---

### Problem: "prototypes.json not found" or accuracy is low

**Cause:** Prototypes need to be rebuilt with PANNS embeddings.

**Solution:**
```bash
cd ml
source venv/bin/activate  # If using venv

# Make sure PANNS service is running in another terminal!

# Rebuild prototypes (takes 5-10 minutes for 150 files)
python rebuild_prototypes_panns.py \
  --dir ../Drone-Training-Data \
  --out ../server/drone/prototypes.json

# Restart the Go server
```

---

### Problem: "FFmpeg not found"

**Solution:**
- **macOS:** `brew install ffmpeg`
- **Linux:** `sudo apt-get install ffmpeg`
- **Windows:** Download from https://ffmpeg.org/download.html and add to PATH

---

### Problem: Port 5001 or 5002 already in use

**Solution:**
```bash
# Find what's using the port
lsof -i :5001  # macOS/Linux
netstat -ano | findstr :5001  # Windows

# Kill the process or use different ports:
USE_PANNS_EMBEDDINGS=true EMBEDDING_SERVICE_URL=http://localhost:5003 \
  go run . serve -proto http -p 5004
```

---

## System Requirements

### Minimum:
- **RAM:** 2GB (4GB recommended)
- **Storage:** 1GB free space (for PANNS model + dependencies)
- **CPU:** Any modern processor (CPU inference is fine)

### Performance:
- **Classification time:** ~800ms per file (includes PANNS inference on CPU)
- **Accuracy:** 100% on training data
- **Confidence:** 80-100% typically

---

## Quick Command Reference

### Start everything:
```bash
# Terminal 1
cd ml && source venv/bin/activate && python embedding_service.py

# Terminal 2
cd server && USE_PANNS_EMBEDDINGS=true go run . serve -proto http -p 5001

# Browser
open http://localhost:5001
```

### Stop everything:
```
Ctrl+C in both terminals
```

### Test from command line:
```bash
cd server
go run ./cmd/mock_frontend -file ../Drone-Training-Data/drone_A/A_01.wav -url http://localhost:5001/api/audio/classify
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `USE_PANNS_EMBEDDINGS` | `true` | Enable PANNS (set to `false` for legacy features) |
| `EMBEDDING_SERVICE_URL` | `http://localhost:5002` | URL of PANNS service |
| `DRONE_MODEL_PATH` | `drone/prototypes.json` | Path to prototype file |
| `DRONE_MODEL_K` | `5` | Number of nearest neighbors for KNN |

---

## File Structure Overview

```
project-root/
├── ml/
│   ├── embedding_service.py      # PANNS API service (START THIS FIRST)
│   ├── rebuild_prototypes_panns.py
│   ├── requirements-panns.txt
│   └── venv/                      # Python virtual environment
├── server/
│   ├── main.go
│   ├── cmdHandlers.go             # HTTP handlers (file upload)
│   ├── socketHandlers.go          # WebSocket handlers (real-time)
│   ├── embedding/
│   │   └── panns_client.go        # Go client for PANNS
│   └── drone/
│       ├── classifier.go          # KNN classifier
│       ├── features.go            # Legacy feature extraction
│       └── prototypes.json        # 150 PANNS embeddings (2048-dim each)
├── client/                        # React frontend
│   └── src/
│       ├── components/
│       │   └── FileUploadDetection.js
│       └── pages/
│           └── DetectionPage.js
└── Drone-Training-Data/          # Audio samples
    ├── drone_A/
    ├── drone_B/
    └── ...
```

---

## Security Notes

**For production deployment:**

1. **Change ports:** Don't use default ports 5001/5002
2. **Add authentication:** The API has no auth currently
3. **Enable HTTPS:** Use proper TLS certificates
4. **Restrict CORS:** Update `allowOriginFunc` in cmdHandlers.go
5. **Rate limiting:** Add request limits to prevent abuse

---

## Support

If you encounter issues:
1. Check both terminal outputs for error messages
2. Verify FFmpeg is installed: `ffmpeg -version`
3. Test PANNS service: `curl http://localhost:5002/health`
4. Check file paths are correct
5. Ensure all dependencies are installed

For more details, see:
- `PANNS_SUCCESS.md` - Technical details
- `QUICK_START_PANNS.md` - Quick reference
- `PANNS_SETUP.md` - Original setup docs


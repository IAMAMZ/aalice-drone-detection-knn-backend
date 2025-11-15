# Generate Test Predictions CSV

This guide shows you how to generate `test_predictions.csv` without running the full project.

## Prerequisites

### Required
- **Go**: Programming language (install from [golang.org](https://golang.org/dl/))
- **Trained Model**: `server/drone/prototypes.json` (already in project)
- **Test Audio Files**: WAV format files in a test directory

### Check if Go is installed
```bash
go version
```

## Quick Start

### 1. Navigate to server directory
```bash
cd server
```

### 2. Run the test model command
```bash
go run ./cmd/test_model \
  -model drone/prototypes.json \
  -test-dir "../Test data" \
  -k 5 \
  -output-csv ../test_predictions.csv \
  -output-json ../test_predictions.json \
  -verbose
```

## Parameters 

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-model` | Path to trained model file | Required |
| `-test-dir` | Directory containing test WAV files | Required |
| `-k` | Number of nearest neighbors | 5 |
| `-output-csv` | Path to save CSV predictions | Optional |
| `-output-json` | Path to save JSON predictions | Optional |
| `-top-k` | Number of top predictions to include | 3 |
| `-verbose` | Enable detailed logging | false |

## Output Files

### CSV Format (`test_predictions.csv`)
```csv
filename,predicted_class,confidence,snr_db,processing_time_ms
Test01.wav,drone h,0.6085,-1.79,324.90
Test02.wav,drone c,0.5519,-0.57,227.60
```

### JSON Format (`test_predictions.json`)
Includes detailed predictions with top-k results, distances, and prototype information.

## Example with Custom Test Directory

If your test files are in a different location:

```bash
go run ./cmd/test_model \
  -model drone/prototypes.json \
  -test-dir "/path/to/your/test/files" \
  -k 5 \
  -output-csv ../test_predictions.csv
```

## Notes

- All test files must be in WAV format
- The model file (`prototypes.json`) must exist and be trained
- Processing time varies based on number of test files
- Higher `-k` values may provide better accuracy but slower processing


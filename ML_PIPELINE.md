# Drone Classification ML Pipeline

This document describes the complete machine learning pipeline for training, evaluating, and testing the drone classifier.

## Overview

The pipeline consists of three main stages:

1. **Training**: Build the model from labeled training data
2. **Evaluation**: Assess model performance on training data (self-check)
3. **Testing**: Make predictions on unlabeled test data

## Prerequisites

```bash
# Ensure you have Go 1.21+ installed
go version

# Navigate to server directory
cd server
```

## Data Organization

### Training Data Structure

```
Drone-Training-Data/
├── drone_A/
│   ├── A_01.wav
│   ├── A_02.wav
│   └── ...
├── drone_B/
│   ├── B_01.wav
│   └── ...
└── drone_X/
    └── ...
```

**Requirements:**
- Each subdirectory represents one class
- Subdirectory name becomes the class label
- Supports `.wav` and `.mp3` files
- Minimum 3-5 samples per class recommended
- 10-15+ samples per class for best results

### Test Data Structure

```
Test data/
├── Test01.wav
├── Test02.wav
└── ...
```

**Requirements:**
- Flat directory structure
- Unlabeled audio files
- Same format as training data (.wav or .mp3)

## Pipeline Stages

### Stage 1: Training

Build the classifier model from training data.

**Command:**
```bash
go run ./cmd/train_model \
  -train-dir ../Drone-Training-Data \
  -output drone/prototypes.json \
  -verbose
```

**Parameters:**
- `-train-dir`: Path to training data directory
- `-output`: Where to save the trained model (default: `drone/prototypes.json`)
- `-category`: Default category (default: `drone`)
- `-verbose`: Enable detailed logging

**Output:**
- Trained model saved to `drone/prototypes.json`
- Training statistics printed to console

**Example Output:**
```
=== Training Summary ===

Total training samples: 128
Successfully processed: 128 (100.0%)
Failed to process: 0

Class distribution:
  drone a             :  15 prototypes
  drone b             :  15 prototypes
  drone c             :  15 prototypes
  ...

Total training time: 12.34 seconds
Average time per sample: 96.41 ms

✓ Training complete!
```

---

### Stage 2: Evaluation

Evaluate the trained model on the training data to check for overfitting and measure accuracy.

**Command:**
```bash
go run ./cmd/evaluate_model \
  -model drone/prototypes.json \
  -train-dir ../Drone-Training-Data \
  -k 5 \
  -report evaluation_report.json \
  -verbose
```

**Parameters:**
- `-model`: Path to trained model
- `-train-dir`: Path to training data (same as training stage)
- `-k`: Number of nearest neighbors (default: 5)
- `-report`: Where to save evaluation report JSON (empty to skip)
- `-verbose`: Enable detailed logging

**Output:**
- Evaluation metrics printed to console
- Detailed report saved to JSON file (if specified)

**Example Output:**
```
=== EVALUATION RESULTS ===

Overall Accuracy: 85.94% (110/128 correct)
Average Confidence: 68.45%
Processing Time: 45.23 seconds

Per-Class Performance:
--------------------------------------------------------------
Class                Accuracy  Confidence     Samples
--------------------------------------------------------------
drone_a                93.3%      72.4%          15   ✓
drone_b                86.7%      69.1%          15   ✓
drone_c                80.0%      65.3%          15   ✓
...

Confusion Matrix:
--------------------------------------------------------------
Actual \ Pred  drone_a drone_b drone_c ...
--------------------------------------------------------------
drone_a             14       1       .
drone_b              2      13       .
...

VERDICT: ✓ GOOD
Accuracy: 85.94%, Confidence: 68.45%
Recommendation: Model works well. Consider adding more diverse training data.
```

**Interpretation:**

| Accuracy | Assessment | Action |
|----------|------------|--------|
| ≥ 90% | Excellent | Production-ready |
| 80-90% | Good | Consider more training data |
| 70-80% | Fair | Add more diverse samples |
| < 70% | Poor | Review data quality, add more samples |

---

### Stage 3: Testing

Make predictions on unlabeled test data.

**Command:**
```bash
go run ./cmd/test_model \
  -model drone/prototypes.json \
  -test-dir "../Test data" \
  -k 5 \
  -output-csv test_predictions.csv \
  -output-json test_predictions.json \
  -top-k 3 \
  -verbose
```

**Parameters:**
- `-model`: Path to trained model
- `-test-dir`: Path to test data directory
- `-k`: Number of nearest neighbors (default: 5)
- `-output-csv`: Save predictions as CSV
- `-output-json`: Save predictions as JSON
- `-top-k`: Number of top predictions to include (default: 3)
- `-verbose`: Enable detailed logging

**Output:**
- Predictions printed to console
- CSV file with predictions
- JSON file with detailed predictions

**Example CSV Output** (`test_predictions.csv`):
```csv
filename,predicted_class,confidence,snr_db,processing_time_ms
Test01.wav,drone_a,0.7234,12.5,45.23
Test02.wav,drone_c,0.8123,15.2,43.11
Test03.wav,drone_b,0.6891,8.7,47.56
...
```

**Example JSON Output** (`test_predictions.json`):
```json
{
  "timestamp": "2024-11-15T10:30:00Z",
  "model_path": "drone/prototypes.json",
  "test_data_dir": "Test data",
  "total_samples": 20,
  "avg_confidence": 0.7234,
  "avg_processing_ms": 45.67,
  "predictions": [
    {
      "filename": "Test01.wav",
      "predicted_class": "drone_a",
      "confidence": 0.7234,
      "snr_db": 12.5,
      "processing_time_ms": 45.23,
      "top_predictions": [
        {
          "label": "drone_a",
          "confidence": 0.7234,
          "category": "drone"
        },
        {
          "label": "drone_b",
          "confidence": 0.1891,
          "category": "drone"
        },
        {
          "label": "drone_c",
          "confidence": 0.0875,
          "category": "drone"
        }
      ]
    },
    ...
  ]
}
```

---

## Complete Pipeline Example

Run all three stages in sequence:

```bash
#!/bin/bash
# pipeline.sh - Complete ML pipeline

cd server

echo "Stage 1: Training..."
go run ./cmd/train_model \
  -train-dir ../Drone-Training-Data \
  -output drone/prototypes.json

echo ""
echo "Stage 2: Evaluation..."
go run ./cmd/evaluate_model \
  -model drone/prototypes.json \
  -train-dir ../Drone-Training-Data \
  -k 5 \
  -report evaluation_report.json

echo ""
echo "Stage 3: Testing..."
go run ./cmd/test_model \
  -model drone/prototypes.json \
  -test-dir "../Test data" \
  -k 5 \
  -output-csv test_predictions.csv \
  -output-json test_predictions.json

echo ""
echo "Pipeline complete! Check outputs:"
echo "  - Model: drone/prototypes.json"
echo "  - Evaluation: evaluation_report.json"
echo "  - Predictions: test_predictions.csv, test_predictions.json"
```

Make it executable and run:
```bash
chmod +x pipeline.sh
./pipeline.sh
```

---

## Hyperparameter Tuning

### K (Number of Neighbors)

The K parameter controls how many nearest neighbors are considered.

**How to tune:**
```bash
# Try different K values
for k in 3 5 7 9; do
  echo "Testing K=$k"
  go run ./cmd/evaluate_model -k $k -model drone/prototypes.json -train-dir ../Drone-Training-Data
done
```

**Guidelines:**
- **K=3**: Use for small datasets (< 50 samples)
- **K=5**: Good default for most cases
- **K=7-9**: Use for large datasets (> 100 samples)
- Higher K = smoother decision boundaries, more robust to noise
- Lower K = more sensitive to local patterns

---

## Troubleshooting

### Low Accuracy (< 70%)

**Possible causes:**
1. Insufficient training data
2. Poor quality audio samples
3. High overlap between classes

**Solutions:**
- Add more diverse training samples (15+ per class)
- Ensure good audio quality (SNR > 10 dB)
- Check if classes are truly distinguishable
- Add preprocessing (already enabled by default)

### High Training Accuracy but Low Test Accuracy

**Cause:** Overfitting

**Solutions:**
- Add more diverse training samples
- Ensure test data matches training data distribution
- Check for data leakage (test samples in training)

### Inconsistent Predictions

**Causes:**
1. Noisy audio (low SNR)
2. Insufficient training samples
3. Overlapping acoustic signatures

**Solutions:**
- Filter low-quality samples (SNR < 5 dB)
- Add more training samples per class
- Review misclassifications to identify patterns

---

## Best Practices

### 1. Data Collection
- ✅ Record in consistent conditions
- ✅ Include variety (distances, angles, noise levels)
- ✅ Minimum 10-15 samples per class
- ✅ Balance classes (similar number of samples)

### 2. Training
- ✅ Always run evaluation after training
- ✅ Save evaluation reports for comparison
- ✅ Version control your models (git)
- ✅ Document data sources and collection process

### 3. Testing
- ✅ Keep test data completely separate
- ✅ Never use test data for training
- ✅ Save predictions for analysis
- ✅ Review low-confidence predictions manually

### 4. Model Updates
- ✅ Retrain when adding new classes
- ✅ Retrain when adding ≥20% more data
- ✅ Compare new model vs old on same test set
- ✅ A/B test in production before full deployment

---

## File Outputs Summary

| File | Generated By | Purpose |
|------|--------------|---------|
| `drone/prototypes.json` | `train_model` | Trained classifier model |
| `evaluation_report.json` | `evaluate_model` | Detailed evaluation metrics |
| `test_predictions.csv` | `test_model` | Test predictions (CSV format) |
| `test_predictions.json` | `test_model` | Test predictions (JSON with details) |

---

## Quick Reference

```bash
# Train
go run ./cmd/train_model -train-dir ../Drone-Training-Data

# Evaluate
go run ./cmd/evaluate_model -model drone/prototypes.json -train-dir ../Drone-Training-Data

# Test
go run ./cmd/test_model -model drone/prototypes.json -test-dir "../Test data"

# All-in-one
./pipeline.sh
```

---

## Questions?

For issues or questions, refer to:
- `BUGFIX_SUMMARY.md` - Technical details about the feature scaling fix
- `FINAL_RESULTS.md` - Performance metrics and analysis
- `NEXT_STEPS.md` - Improvement recommendations


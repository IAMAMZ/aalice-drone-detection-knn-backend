#!/bin/bash
# pipeline.sh - Complete ML pipeline for drone classification
# This script trains, evaluates, and tests the drone classifier model

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TRAIN_DIR="Drone-Training-Data"
TEST_DIR="Test data"
MODEL_PATH="server/drone/prototypes.json"
EVAL_REPORT="evaluation_report.json"
TEST_CSV="test_predictions.csv"
TEST_JSON="test_predictions.json"
K_VALUE=5

# Print banner
echo -e "${BLUE}======================================================================"
echo "Drone Classification ML Pipeline"
echo -e "======================================================================${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v go &> /dev/null; then
    echo -e "${RED}ERROR: Go is not installed${NC}"
    exit 1
fi

if [ ! -d "$TRAIN_DIR" ]; then
    echo -e "${RED}ERROR: Training directory not found: $TRAIN_DIR${NC}"
    exit 1
fi

if [ ! -d "$TEST_DIR" ]; then
    echo -e "${YELLOW}WARNING: Test directory not found: $TEST_DIR${NC}"
    echo "Skipping test stage"
    SKIP_TEST=true
fi

cd server
echo -e "${GREEN}✓ Prerequisites OK${NC}"
echo ""

# Stage 1: Training
echo -e "${BLUE}======================================================================"
echo "Stage 1: Training Model"
echo -e "======================================================================${NC}"
echo ""

go run ./cmd/train_model \
  -train-dir "../${TRAIN_DIR}" \
  -output drone/prototypes.json

if [ $? -ne 0 ]; then
    echo -e "${RED}ERROR: Training failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Training complete${NC}"
echo ""

# Stage 2: Evaluation
echo -e "${BLUE}======================================================================"
echo "Stage 2: Evaluating Model"
echo -e "======================================================================${NC}"
echo ""

go run ./cmd/evaluate_model \
  -model drone/prototypes.json \
  -train-dir "../${TRAIN_DIR}" \
  -k ${K_VALUE} \
  -report "../${EVAL_REPORT}"

if [ $? -ne 0 ]; then
    echo -e "${RED}ERROR: Evaluation failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Evaluation complete${NC}"
echo ""

# Stage 3: Testing (if test data exists)
if [ "$SKIP_TEST" != "true" ]; then
    echo -e "${BLUE}======================================================================"
    echo "Stage 3: Testing Model"
    echo -e "======================================================================${NC}"
    echo ""

    go run ./cmd/test_model \
      -model drone/prototypes.json \
      -test-dir "../${TEST_DIR}" \
      -k ${K_VALUE} \
      -output-csv "../${TEST_CSV}" \
      -output-json "../${TEST_JSON}"

    if [ $? -ne 0 ]; then
        echo -e "${RED}ERROR: Testing failed${NC}"
        exit 1
    fi

    echo ""
    echo -e "${GREEN}✓ Testing complete${NC}"
    echo ""
fi

cd ..

# Summary
echo -e "${BLUE}======================================================================"
echo "Pipeline Complete!"
echo -e "======================================================================${NC}"
echo ""
echo "Output files:"
echo "  Model:       ${MODEL_PATH}"
echo "  Evaluation:  ${EVAL_REPORT}"

if [ "$SKIP_TEST" != "true" ]; then
    echo "  Predictions: ${TEST_CSV}"
    echo "               ${TEST_JSON}"
fi

echo ""
echo -e "${GREEN}✓ All stages completed successfully!${NC}"
echo ""
echo "Next steps:"
echo "  1. Review evaluation report for accuracy metrics"
echo "  2. Check test predictions for correctness"
echo "  3. If accuracy < 80%, add more training data"
echo "  4. Run 'go run server/main.go serve -p 5000' to start API server"
echo ""


# Drone Prototype Builder

This directory hosts optional tooling used to curate the acoustic prototypes that power the realtime classifier.

## 1. Install the Python toolchain

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

The scripts expect FFmpeg to be available on your `PATH` for reliable resampling.

## 2. Prepare your dataset

Organise wave files by label:

```
<dataset-root>/
  ├── dji_mavic/
  │    ├── clip_01.wav
  │    └── clip_02.wav
  ├── parrot_anafi/
  │    └── takeoff.wav
  ├── urban_noise/
  │    └── truck_idle.wav
  └── wind_noise/
       └── gust.wav
```

Optional label metadata (category, friendly description, etc.) can be captured in a YAML file following `category-map.example.yaml`.

## 3. Generate prototypes

```bash
python build_prototypes.py \
  --input /path/to/dataset \
  --output ../server/drone/prototypes.json \
  --category-map category-map.yaml
```

The script extracts a lightweight spectral feature vector for every file, normalises the result, and writes a JSON artefact that the Go backend consumes at runtime.

Regenerate this file whenever you add new drone recordings or rebalance the dataset.

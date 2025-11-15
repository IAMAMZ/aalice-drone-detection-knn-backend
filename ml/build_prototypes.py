#!/usr/bin/env python3
"""Build prototype embeddings for the drone classifier."""

from __future__ import annotations

import argparse
import json
import uuid
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Iterable, List, Optional

import librosa
import numpy as np
import soundfile as sf
from tqdm import tqdm


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate prototype embeddings for drone audio")
    parser.add_argument("--input", required=True, type=Path, help="Root directory containing labelled .wav files")
    parser.add_argument("--output", required=True, type=Path, help="Output JSON path for prototypes")
    parser.add_argument("--category-map", type=Path, default=None, help="Optional YAML/JSON describing category metadata")
    parser.add_argument("--sample-rate", type=int, default=44_100, help="Target sample rate when loading audio")
    return parser.parse_args()


@dataclass
class LabelMetadata:
    category: str = "drone"
    description: Optional[str] = None
    extra: Dict[str, str] = None


def load_metadata(path: Optional[Path]) -> Dict[str, LabelMetadata]:
    if path is None:
        return {}

    import yaml  # pylint: disable=import-outside-toplevel

    with path.open("r", encoding="utf-8") as handle:
        config = yaml.safe_load(handle)

    result: Dict[str, LabelMetadata] = {}
    for label, payload in config.get("labels", {}).items():
        category = payload.get("category", "drone")
        description = payload.get("description")
        extras = {k: str(v) for k, v in payload.items() if k not in {"category", "description"}}
        result[label] = LabelMetadata(category=category, description=description, extra=extras)
    return result


def list_audio_files(root: Path) -> Iterable[Path]:
    for path in sorted(root.rglob("*.wav")):
        if path.is_file():
            yield path


def compute_harmonic_features(
    magnitude: np.ndarray, freqs: np.ndarray, fundamental_freq: float, sample_rate: int
) -> tuple[float, float, float]:
    """Compute harmonic features matching the Go implementation."""
    if len(magnitude) == 0 or fundamental_freq <= 0:
        return 0.0, 0.0, 0.0

    # Calculate total energy
    total_energy = float(np.sum(magnitude ** 2))
    if total_energy == 0:
        return 0.0, 0.0, 0.0

    # Find peaks
    peaks = []
    avg_mag = float(np.mean(magnitude))
    for i in range(1, len(magnitude) - 1):
        if magnitude[i] > magnitude[i - 1] and magnitude[i] > magnitude[i + 1]:
            if magnitude[i] > avg_mag * 1.2:
                peaks.append(i)

    if len(peaks) == 0:
        return 0.0, 0.0, 0.0

    # Calculate frequency resolution
    freq_resolution = float(sample_rate) / float(len(magnitude) * 2)

    # Find harmonics of the fundamental frequency
    max_harmonic = 10
    harmonic_energy = 0.0
    harmonic_magnitudes = []
    tolerance = fundamental_freq * 0.1  # 10% tolerance

    for h in range(1, max_harmonic + 1):
        target_freq = fundamental_freq * float(h)
        if target_freq >= sample_rate / 2:
            break

        # Find the bin closest to the target harmonic frequency
        target_bin = int(target_freq / freq_resolution)
        if target_bin >= len(magnitude):
            break

        # Search in a small window around the expected harmonic
        search_window = max(1, min(10, int(tolerance / freq_resolution)))
        start_bin = max(0, target_bin - search_window)
        end_bin = min(len(magnitude) - 1, target_bin + search_window)

        # Find maximum in the search window
        max_mag = float(np.max(magnitude[start_bin : end_bin + 1]))

        # Harmonic must be at least 1.5x the average magnitude
        if max_mag > avg_mag * 1.5:
            harmonic_energy += max_mag * max_mag
            harmonic_magnitudes.append(max_mag)

    # Calculate harmonic ratio
    harmonic_ratio = harmonic_energy / total_energy if total_energy > 0 else 0.0

    # Harmonic count (normalized to 0-1 range)
    harmonic_count = min(1.0, float(len(harmonic_magnitudes)) / 10.0)

    # Harmonic strength (average magnitude of harmonics, normalized)
    harmonic_strength = 0.0
    if len(harmonic_magnitudes) > 0:
        avg_harmonic_mag = float(np.mean(harmonic_magnitudes))
        max_possible_mag = float(np.max(magnitude))
        if max_possible_mag > 0:
            harmonic_strength = avg_harmonic_mag / max_possible_mag

    return harmonic_ratio, harmonic_count, harmonic_strength


def compute_feature_vector(waveform: np.ndarray, sample_rate: int) -> List[float]:
    if waveform.ndim > 1:
        waveform = librosa.to_mono(waveform.T)

    waveform = librosa.util.normalize(waveform)
    rms = float(np.sqrt(np.mean(waveform**2)))
    zcr = float(np.mean(librosa.feature.zero_crossing_rate(waveform)))
    variance = float(np.var(waveform))

    n_fft = 2048
    hop_length = n_fft // 2
    stft = librosa.stft(waveform, n_fft=n_fft, hop_length=hop_length)
    magnitude = np.abs(stft)
    freq_axis = librosa.fft_frequencies(sr=sample_rate, n_fft=n_fft)
    avg_spectrum = np.mean(magnitude, axis=1)

    spectral_centroid = float(np.sum(freq_axis * avg_spectrum) / (np.sum(avg_spectrum) + 1e-12))
    
    # Re-implementing Go's version of these features
    
    # Spectral Bandwidth
    deviation = freq_axis - spectral_centroid
    spectral_bandwidth = float(np.sqrt(np.sum(avg_spectrum * deviation * deviation) / (np.sum(avg_spectrum) + 1e-12)))

    # Spectral Rolloff
    total_energy = np.sum(avg_spectrum)
    target_energy = 0.85 * total_energy
    cumulative_energy = np.cumsum(avg_spectrum)
    rolloff_index = np.searchsorted(cumulative_energy, target_energy)
    spectral_rolloff = float(freq_axis[rolloff_index]) if rolloff_index < len(freq_axis) else float(freq_axis[-1])

    # Spectral Flatness
    log_mag = np.log(avg_spectrum + 1e-12)
    geometric_mean = np.exp(np.mean(log_mag))
    arithmetic_mean = np.mean(avg_spectrum)
    spectral_flatness = float(geometric_mean / (arithmetic_mean + 1e-12))
    
    # Spectral Entropy
    prob = avg_spectrum / (np.sum(avg_spectrum) + 1e-12)
    entropy = float(-np.sum(prob * np.log2(prob + 1e-12)) / np.log2(len(prob)))

    crest_factor = float(np.max(avg_spectrum) / (np.mean(avg_spectrum) + 1e-12))

    dominant_index = int(np.argmax(avg_spectrum))
    dominant_frequency = float(freq_axis[dominant_index])
    
    # Temporal features from Go implementation
    # These require operating on the original waveform
    temporal_centroid_val = np.sum(np.arange(len(waveform)) * (waveform**2)) / (np.sum(waveform**2) + 1e-12) / len(waveform)

    # Simplified onset rate for Python
    onset_env = librosa.onset.onset_detect(y=waveform, sr=sample_rate, units='time')
    onset_rate_val = len(onset_env) / (len(waveform) / sample_rate)
    onset_rate_norm = min(1.0, onset_rate_val / 20.0) # Normalize by max 20 onsets/sec

    # Amplitude modulation depth
    env = np.abs(librosa.effects.preemphasis(waveform))
    mean_env = np.mean(env)
    std_env = np.std(env)
    am_depth_val = min(1.0, std_env / (mean_env + 1e-9))
    
    # Spectral Shape features from Go
    # Skewness
    third_moment = np.sum(avg_spectrum * ((freq_axis - spectral_centroid) ** 3)) / (np.sum(avg_spectrum) + 1e-12)
    skewness_val = np.tanh(third_moment / ((spectral_bandwidth ** 3) + 1e-12))

    # Kurtosis
    fourth_moment = np.sum(avg_spectrum * ((freq_axis - spectral_centroid) ** 4)) / (np.sum(avg_spectrum) + 1e-12)
    kurtosis_val = max(0, (fourth_moment / ((spectral_bandwidth ** 4) + 1e-12)) / 3.0)

    # Peak Prominence
    sorted_peaks = np.sort(avg_spectrum)
    top_peaks_avg = np.mean(sorted_peaks[-3:])
    mean_spectrum = np.mean(avg_spectrum)
    peak_prominence_val = (top_peaks_avg - mean_spectrum) / (top_peaks_avg + mean_spectrum + 1e-9)
    peak_prominence_val = max(0, min(1, peak_prominence_val))


    # Harmonic features (matching Go implementation)
    # Need raw Hz value for harmonic calculation
    harmonic_ratio, harmonic_count, harmonic_strength = compute_harmonic_features(
        avg_spectrum, freq_axis, dominant_frequency, sample_rate
    )

    # Normalize frequency-based features to 0-1 range to prevent scale mismatch
    # Frequency features are in Hz (0 to sample_rate/2), normalize by Nyquist frequency
    nyquist_freq = sample_rate / 2.0
    if nyquist_freq > 0:
        spectral_centroid = np.clip(spectral_centroid / nyquist_freq, 0, 1)
        spectral_bandwidth = np.clip(spectral_bandwidth / nyquist_freq, 0, 1)
        spectral_rolloff = np.clip(spectral_rolloff / nyquist_freq, 0, 1)
        dominant_frequency = np.clip(dominant_frequency / nyquist_freq, 0, 1)

    vector = np.array(
        [
            rms,
            zcr,
            spectral_centroid,
            spectral_bandwidth,
            spectral_rolloff,
            spectral_flatness,
            dominant_frequency,
            crest_factor,
            entropy,
            variance,
            temporal_centroid_val,
            onset_rate_norm,
            am_depth_val,
            skewness_val,
            kurtosis_val,
            peak_prominence_val,
            harmonic_ratio,
            harmonic_count,
            harmonic_strength,
        ],
        dtype=float,
    )

    norm = float(np.linalg.norm(vector))
    if norm > 0:
        vector /= norm

    return vector.round(6).tolist()


def build_prototypes(args: argparse.Namespace) -> None:
    metadata = load_metadata(args.category_map)

    input_root = args.input.resolve()
    files = list(list_audio_files(input_root))
    if not files:
        raise RuntimeError(f"No wav files found under {input_root}")

    prototypes = []
    for wav_path in tqdm(files, desc="Extracting features"):
        try:
            waveform, sr = sf.read(wav_path)
        except RuntimeError:
            waveform, sr = librosa.load(wav_path, sr=args.sample_rate, mono=True)

        if sr != args.sample_rate:
            waveform = librosa.resample(
                waveform.T if isinstance(waveform, np.ndarray) and waveform.ndim > 1 else waveform,
                orig_sr=sr,
                target_sr=args.sample_rate,
            )
            sr = args.sample_rate

        label = wav_path.parent.name
        label_meta = metadata.get(label, LabelMetadata())
        if label_meta.extra is None:
            label_meta.extra = {}
        if label_meta.description and "description" not in label_meta.extra:
            label_meta.extra["description"] = label_meta.description

        features = compute_feature_vector(np.asarray(waveform, dtype=float), sr)
        proto_id = f"proto_{label}_{uuid.uuid4().hex[:8]}"

        prototypes.append(
            {
                "id": proto_id,
                "label": label,
                "category": label_meta.category,
                "description": label_meta.description,
                "source": str(wav_path),
                "features": features,
                "metadata": label_meta.extra,
            }
        )

    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w", encoding="utf-8") as handle:
        json.dump(prototypes, handle, indent=2)

    print(f"Wrote {len(prototypes)} prototypes to {args.output}")


def main() -> None:
    args = parse_args()
    build_prototypes(args)


if __name__ == "__main__":
    main()

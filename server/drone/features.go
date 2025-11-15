package drone

// Feature Extraction Pipeline
//
// This package implements audio feature extraction for drone acoustic signature analysis.
// The system extracts 19 spectral and temporal features from audio samples:
//
// Temporal Features:
//   - Energy (RMS): Root mean square amplitude, measures overall signal strength
//   - Zero Crossing Rate: Frequency of sign changes, indicates pitch/noise characteristics
//   - Variance: Signal variability, helps distinguish steady tones from noise
//
// Temporal Stability Features:
//   - Temporal Centroid: Location of energy within the window (normalised 0-1)
//   - Onset Rate: Frequency of amplitude onsets, indicates impulsive activity
//   - Amplitude Modulation Depth: Variation of the amplitude envelope
//
// Spectral Features (derived from FFT):
//   - Spectral Centroid: "Brightness" of sound, weighted average frequency
//   - Spectral Bandwidth: Spread of frequencies around the centroid
//   - Spectral Rolloff: Frequency below which 85% of energy is contained
//   - Spectral Flatness: Ratio of geometric to arithmetic mean, measures noisiness
//   - Spectral Crest Factor: Peak-to-average ratio, indicates tonal vs noisy content
//   - Spectral Entropy: Measure of randomness in frequency distribution
//   - Dominant Frequency: Frequency bin with maximum magnitude
//
// Spectral Shape Features:
//   - Spectral Skewness: Asymmetry of the frequency distribution
//   - Spectral Kurtosis: Peakedness of the frequency distribution
//   - Peak Prominence: Contrast between the strongest peaks and average spectrum level
//
// Harmonic Features (critical for drone detection):
//   - Harmonic Ratio: Ratio of harmonic energy to total energy
//   - Harmonic Count: Number of significant harmonic peaks
//   - Harmonic Strength: Average magnitude of harmonic components
//
// Processing Steps:
// 1. Compute FFT: Convert time-domain signal to frequency domain using Fast Fourier Transform
// 2. Apply Hann Window: Reduce spectral leakage before FFT
// 3. Extract Magnitude Spectrum: Calculate magnitude of complex FFT results
// 4. Compute Features: Calculate each feature from the magnitude spectrum
// 5. Normalize: Vector is normalized to unit length for distance-based classification
//
// These features form a compact 19-dimensional descriptor that captures the acoustic
// signature of drone propellers, which typically have distinct spectral characteristics
// including harmonic content, rotor blade frequencies, and motor noise patterns.

import (
	"errors"
	"math"
	"math/cmplx"
	"sort"

	"song-recognition/shazam"
)

// ExtractFeatureVector derives a compact descriptor for an audio waveform.
func ExtractFeatureVector(samples []float64, sampleRate int) ([]float64, error) {
	if len(samples) == 0 {
		return nil, errors.New("no samples provided")
	}
	if sampleRate <= 0 {
		return nil, errors.New("invalid sample rate")
	}

	energy := rootMeanSquare(samples)
	zcr := zeroCrossingRate(samples)
	variance := signalVariance(samples)

	spectrum, freqs := computeSpectrum(samples, sampleRate)
	centroid := spectralCentroid(spectrum, freqs)
	bandwidth := spectralBandwidth(spectrum, freqs, centroid)
	rolloff := spectralRolloff(spectrum, freqs, 0.85)
	flatness := spectralFlatness(spectrum)
	crest := spectralCrestFactor(spectrum)
	entropy := spectralEntropy(spectrum)
	dominant := dominantFrequency(spectrum, freqs)

	temporalCentre := temporalCentroid(samples, sampleRate)
	onsetRateNorm := onsetRate(samples, sampleRate)
	amDepth := amplitudeModulationDepth(samples)
	// Calculate skewness and kurtosis using raw Hz values (they're normalized internally)
	skewness := spectralSkewness(spectrum, freqs, centroid, bandwidth)
	kurtosis := spectralKurtosis(spectrum, freqs, centroid, bandwidth)
	peakProminence := spectralPeakProminence(spectrum)

	// Harmonic features (critical for drone detection)
	// Only compute if dominant frequency is valid (needs raw Hz value)
	var harmonicRatio, harmonicCount, harmonicStrength float64
	if dominant > 0 {
		harmonicRatio, harmonicCount, harmonicStrength = harmonicFeatures(spectrum, freqs, dominant, sampleRate)
	}

	// Normalize frequency-based features to 0-1 range AFTER all calculations that need raw Hz values
	// This prevents scale mismatch where Hz values (0-22050) dominate normalized features (0-1)
	// Frequency features are in Hz (0 to sampleRate/2), normalize by Nyquist frequency
	nyquistFreq := float64(sampleRate) / 2.0
	if nyquistFreq > 0 {
		centroid = clamp01(centroid / nyquistFreq)
		bandwidth = clamp01(bandwidth / nyquistFreq)
		rolloff = clamp01(rolloff / nyquistFreq)
		dominant = clamp01(dominant / nyquistFreq)
	}

	// Normalize other large-scale features to prevent them from dominating distance calculations
	// Crest factor typically ranges from 1-200, normalize by dividing by expected max
	crest = clamp01(crest / 100.0) // Most audio has crest < 100
	// Kurtosis typically ranges from -3 to 10+, normalize to 0-1
	kurtosis = clamp01((kurtosis + 3.0) / 13.0) // Shift and scale to 0-1 range

	return []float64{
		energy,
		zcr,
		centroid,
		bandwidth,
		rolloff,
		flatness,
		dominant,
		crest,
		entropy,
		variance,
		temporalCentre,
		onsetRateNorm,
		amDepth,
		skewness,
		kurtosis,
		peakProminence,
		harmonicRatio,
		harmonicCount,
		harmonicStrength,
	}, nil
}

func rootMeanSquare(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, v := range samples {
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(samples)))
}

func zeroCrossingRate(samples []float64) float64 {
	if len(samples) <= 1 {
		return 0
	}
	var count float64
	for i := 1; i < len(samples); i++ {
		if samples[i-1] == 0 || samples[i] == 0 {
			continue
		}
		if (samples[i-1] > 0) != (samples[i] > 0) {
			count++
		}
	}
	return count / float64(len(samples)-1)
}

func signalVariance(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, v := range samples {
		sum += v
	}
	mean := sum / float64(len(samples))
	var variance float64
	for _, v := range samples {
		diff := v - mean
		variance += diff * diff
	}
	return variance / float64(len(samples))
}

func computeSpectrum(samples []float64, sampleRate int) ([]float64, []float64) {
	fftSize := nextPowerOfTwo(len(samples))
	buffer := make([]float64, fftSize)
	copy(buffer, samples)
	applyHannWindow(buffer)

	fft := shazam.FFT(buffer)
	binCount := fftSize / 2
	magnitude := make([]float64, binCount)
	freqs := make([]float64, binCount)

	for i := 0; i < binCount; i++ {
		mag := cmplx.Abs(fft[i])
		magnitude[i] = mag
		freqs[i] = float64(i) * float64(sampleRate) / float64(fftSize)
	}

	return magnitude, freqs
}

func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	power := 1
	for power < n {
		power <<= 1
	}
	return power
}

func applyHannWindow(buffer []float64) {
	length := len(buffer)
	if length <= 1 {
		return
	}
	for i := range buffer {
		buffer[i] *= 0.5 * (1 - math.Cos((2*math.Pi*float64(i))/float64(length-1)))
	}
}

func spectralCentroid(magnitude, freqs []float64) float64 {
	var weightedSum float64
	var total float64
	for i := range magnitude {
		weightedSum += magnitude[i] * freqs[i]
		total += magnitude[i]
	}
	if total == 0 {
		return 0
	}
	return weightedSum / total
}

func spectralBandwidth(magnitude, freqs []float64, centroid float64) float64 {
	var variance float64
	var total float64
	for i := range magnitude {
		deviation := freqs[i] - centroid
		variance += magnitude[i] * deviation * deviation
		total += magnitude[i]
	}
	if total == 0 {
		return 0
	}
	return math.Sqrt(variance / total)
}

func spectralRolloff(magnitude, freqs []float64, threshold float64) float64 {
	if len(magnitude) == 0 {
		return 0
	}
	if threshold <= 0 || threshold >= 1 {
		threshold = 0.85
	}

	var cumulative float64
	var total float64
	for _, mag := range magnitude {
		total += mag
	}
	if total == 0 {
		return freqs[len(freqs)-1]
	}

	target := threshold * total
	for i, mag := range magnitude {
		cumulative += mag
		if cumulative >= target {
			return freqs[i]
		}
	}

	return freqs[len(freqs)-1]
}

func spectralFlatness(magnitude []float64) float64 {
	if len(magnitude) == 0 {
		return 0
	}
	const eps = 1e-12
	var logSum float64
	var geomCount float64
	var arithmetic float64

	for _, mag := range magnitude {
		value := mag + eps
		logSum += math.Log(value)
		arithmetic += value
		geomCount++
	}

	geoMean := math.Exp(logSum / geomCount)
	ariMean := arithmetic / geomCount

	if ariMean == 0 {
		return 0
	}
	return geoMean / ariMean
}

func spectralCrestFactor(magnitude []float64) float64 {
	if len(magnitude) == 0 {
		return 0
	}
	maxVal := magnitude[0]
	var sum float64
	for _, mag := range magnitude {
		if mag > maxVal {
			maxVal = mag
		}
		sum += mag
	}
	mean := sum / float64(len(magnitude))
	if mean == 0 {
		return 0
	}
	return maxVal / mean
}

func spectralEntropy(magnitude []float64) float64 {
	if len(magnitude) == 0 {
		return 0
	}
	var powerSum float64
	for _, mag := range magnitude {
		powerSum += mag * mag
	}
	if powerSum == 0 {
		return 0
	}

	probabilities := make([]float64, len(magnitude))
	for i, mag := range magnitude {
		probabilities[i] = (mag * mag) / powerSum
	}

	var entropy float64
	for _, p := range probabilities {
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy / math.Log2(float64(len(magnitude)))
}

func dominantFrequency(magnitude, freqs []float64) float64 {
	if len(magnitude) == 0 {
		return 0
	}
	idx := 0
	maxVal := magnitude[0]
	for i, mag := range magnitude {
		if mag > maxVal {
			maxVal = mag
			idx = i
		}
	}
	return freqs[idx]
}

// harmonicFeatures extracts harmonic-related features from the spectrum.
// Drones produce strong harmonics due to propeller rotation, making these features
// highly discriminative for drone detection.
//
// Returns:
//   - harmonicRatio: Ratio of harmonic energy to total energy (0-1)
//   - harmonicCount: Number of significant harmonic peaks detected
//   - harmonicStrength: Average magnitude of harmonic components
func harmonicFeatures(magnitude, freqs []float64, fundamentalFreq float64, sampleRate int) (harmonicRatio, harmonicCount, harmonicStrength float64) {
	if len(magnitude) == 0 || fundamentalFreq <= 0 {
		return 0, 0, 0
	}

	// Calculate total energy and average magnitude in one pass (optimization)
	var totalEnergy float64
	var sumMag float64
	for _, mag := range magnitude {
		totalEnergy += mag * mag
		sumMag += mag
	}
	if totalEnergy == 0 {
		return 0, 0, 0
	}
	avgMag := sumMag / float64(len(magnitude))

	// Calculate frequency resolution
	freqResolution := float64(sampleRate) / float64(len(magnitude)*2)

	// Find max magnitude for normalization (optimization)
	maxPossibleMag := 0.0
	for _, mag := range magnitude {
		if mag > maxPossibleMag {
			maxPossibleMag = mag
		}
	}

	// Find harmonics of the fundamental frequency
	// Check up to 10 harmonics (reasonable for drone propellers)
	maxHarmonic := 10
	harmonicEnergy := 0.0
	harmonicMagnitudes := []float64{}
	tolerance := fundamentalFreq * 0.1 // 10% tolerance for harmonic detection

	// Pre-calculate search window size (optimization)
	searchWindowSize := int(tolerance / freqResolution)
	if searchWindowSize < 1 {
		searchWindowSize = 1
	}
	if searchWindowSize > 10 {
		searchWindowSize = 10
	}

	for h := 1; h <= maxHarmonic; h++ {
		targetFreq := fundamentalFreq * float64(h)
		if targetFreq >= float64(sampleRate)/2 {
			break
		}

		// Find the bin closest to the target harmonic frequency
		targetBin := int(targetFreq / freqResolution)
		if targetBin >= len(magnitude) {
			break
		}

		// Search in a small window around the expected harmonic
		startBin := targetBin - searchWindowSize
		endBin := targetBin + searchWindowSize
		if startBin < 0 {
			startBin = 0
		}
		if endBin >= len(magnitude) {
			endBin = len(magnitude) - 1
		}

		// Find maximum in the search window
		maxMag := 0.0
		for i := startBin; i <= endBin; i++ {
			if magnitude[i] > maxMag {
				maxMag = magnitude[i]
			}
		}

		// Harmonic must be at least 1.5x the average magnitude
		if maxMag > avgMag*1.5 {
			harmonicEnergy += maxMag * maxMag
			harmonicMagnitudes = append(harmonicMagnitudes, maxMag)
		}
	}

	// Calculate harmonic ratio
	harmonicRatio = harmonicEnergy / totalEnergy

	// Harmonic count (normalized to 0-1 range, assuming max 10 harmonics)
	harmonicCount = float64(len(harmonicMagnitudes)) / 10.0
	if harmonicCount > 1.0 {
		harmonicCount = 1.0
	}

	// Harmonic strength (average magnitude of harmonics, normalized)
	if len(harmonicMagnitudes) > 0 && maxPossibleMag > 0 {
		var sum float64
		for _, mag := range harmonicMagnitudes {
			sum += mag
		}
		harmonicStrength = (sum / float64(len(harmonicMagnitudes))) / maxPossibleMag
	}

	return harmonicRatio, harmonicCount, harmonicStrength
}

// NormaliseVector rescales a vector into unit length to aid distance computation.
func NormaliseVector(vector []float64) []float64 {
	var sumSquares float64
	for _, v := range vector {
		sumSquares += v * v
	}
	if sumSquares == 0 {
		return append([]float64{}, vector...)
	}
	factor := 1 / math.Sqrt(sumSquares)
	result := make([]float64, len(vector))
	for i, v := range vector {
		result[i] = v * factor
	}
	return result
}

// NormaliseVectorInPlace rescales the slice in place.
func NormaliseVectorInPlace(vector []float64) {
	var sumSquares float64
	for _, v := range vector {
		sumSquares += v * v
	}
	if sumSquares == 0 {
		return
	}
	factor := 1 / math.Sqrt(sumSquares)
	for i := range vector {
		vector[i] *= factor
	}
}

// SortFeatureVector ensures deterministic ordering (useful for logs and fixtures).
func SortFeatureVector(vector []float64) []float64 {
	result := append([]float64(nil), vector...)
	sort.Float64s(result)
	return result
}

// clamp01 clamps a value to [0, 1] range
func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// temporalCentroid returns the temporal center of mass of the signal (normalised 0-1).
func temporalCentroid(samples []float64, sampleRate int) float64 {
	if len(samples) == 0 || sampleRate <= 0 {
		return 0
	}

	var energySum float64
	var weightedSum float64
	for i, sample := range samples {
		energy := sample * sample
		energySum += energy
		weightedSum += energy * float64(i)
	}

	if energySum == 0 {
		return 0
	}

	centroidIndex := weightedSum / energySum
	return centroidIndex / float64(len(samples))
}

// onsetRate estimates the rate of amplitude onsets, normalised to 0-1 with cap.
func onsetRate(samples []float64, sampleRate int) float64 {
	if len(samples) < 2 || sampleRate <= 0 {
		return 0
	}

	absVals := make([]float64, len(samples))
	var sumAbs float64
	for i, s := range samples {
		val := math.Abs(s)
		absVals[i] = val
		sumAbs += val
	}

	mean := sumAbs / float64(len(absVals))
	var variance float64
	for _, v := range absVals {
		diff := v - mean
		variance += diff * diff
	}
	std := 0.0
	if len(absVals) > 0 {
		std = math.Sqrt(variance / float64(len(absVals)))
	}
	threshold := mean + std

	onsetCount := 0
	for i := 1; i < len(absVals); i++ {
		if absVals[i-1] < threshold && absVals[i] >= threshold {
			onsetCount++
		}
	}

	duration := float64(len(samples)) / float64(sampleRate)
	if duration <= 0 {
		return 0
	}

	rate := float64(onsetCount) / duration
	const maxRate = 20.0 // normalise assuming max 20 significant onsets per second
	if rate > maxRate {
		rate = maxRate
	}
	return rate / maxRate
}

// amplitudeModulationDepth returns the ratio of amplitude variability to mean level (0-1).
func amplitudeModulationDepth(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}

	var sumAbs float64
	for _, s := range samples {
		sumAbs += math.Abs(s)
	}
	mean := sumAbs / float64(len(samples))
	if mean == 0 {
		return 0
	}

	var variance float64
	for _, s := range samples {
		diff := math.Abs(s) - mean
		variance += diff * diff
	}
	std := math.Sqrt(variance / float64(len(samples)))

	depth := std / (mean + 1e-9)
	if depth > 1.0 {
		return 1.0
	}
	return depth
}

// spectralSkewness measures asymmetry of frequency distribution.
func spectralSkewness(magnitude, freqs []float64, centroid, bandwidth float64) float64 {
	if len(magnitude) == 0 || bandwidth == 0 {
		return 0
	}

	var total float64
	var thirdMoment float64
	for i := range magnitude {
		weight := magnitude[i]
		diff := freqs[i] - centroid
		thirdMoment += weight * diff * diff * diff
		total += weight
	}
	if total == 0 {
		return 0
	}

	value := (thirdMoment / total) / (math.Pow(bandwidth, 3) + 1e-12)
	// squash to manageable range (-1, 1)
	return math.Tanh(value)
}

// spectralKurtosis measures peakedness of frequency distribution.
func spectralKurtosis(magnitude, freqs []float64, centroid, bandwidth float64) float64 {
	if len(magnitude) == 0 || bandwidth == 0 {
		return 0
	}

	var total float64
	var fourthMoment float64
	for i := range magnitude {
		weight := magnitude[i]
		diff := freqs[i] - centroid
		fourthMoment += weight * diff * diff * diff * diff
		total += weight
	}
	if total == 0 {
		return 0
	}

	value := (fourthMoment / total) / (math.Pow(bandwidth, 4) + 1e-12)
	// normalise relative to Gaussian kurtosis (3)
	return math.Max(0, value/3.0)
}

// spectralPeakProminence measures contrast between dominant peaks and average spectrum level.
func spectralPeakProminence(magnitude []float64) float64 {
	if len(magnitude) == 0 {
		return 0
	}

	var sum float64
	for _, mag := range magnitude {
		sum += mag
	}
	mean := sum / float64(len(magnitude))
	if mean == 0 {
		return 0
	}

	peaks := append([]float64(nil), magnitude...)
	sort.Float64s(peaks)
	topCount := 3
	if len(peaks) < topCount {
		topCount = len(peaks)
	}

	var topSum float64
	for i := 0; i < topCount; i++ {
		topSum += peaks[len(peaks)-1-i]
	}
	topAvg := topSum / float64(topCount)

	prominence := (topAvg - mean) / (topAvg + mean + 1e-9)
	if prominence < 0 {
		return 0
	}
	if prominence > 1 {
		return 1
	}
	return prominence
}

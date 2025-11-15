package drone

// Audio Pre-processing and Noise Reduction
//
// This package provides audio preprocessing functions to improve detection accuracy
// in noisy environments. It includes:
//
// 1. High-pass filter: Removes low-frequency noise (<50Hz)
// 2. Band-pass filter: Focuses on drone frequency range (100-5000 Hz)
// 3. Automatic Gain Control (AGC): Normalizes audio levels
// 4. Spectral subtraction: Reduces background noise
// 5. SNR estimation: Measures signal-to-noise ratio

import (
	"math"
)

// PreprocessingConfig holds configuration for audio preprocessing
type PreprocessingConfig struct {
	EnableHighPass       bool
	HighPassCutoff       float64 // Hz, default 50
	EnableBandPass       bool
	BandPassLow          float64 // Hz, default 100
	BandPassHigh         float64 // Hz, default 5000
	EnableAGC            bool
	AGCTargetLevel       float64 // Target RMS level, default 0.3
	EnableNoiseReduction bool
	NoiseReductionAlpha  float64 // Spectral subtraction factor, default 0.1
}

// DefaultPreprocessingConfig returns a sensible default configuration
func DefaultPreprocessingConfig() PreprocessingConfig {
	return PreprocessingConfig{
		EnableHighPass:       true,
		HighPassCutoff:       50.0,
		EnableBandPass:       true,
		BandPassLow:          100.0,
		BandPassHigh:         5000.0,
		EnableAGC:            true,
		AGCTargetLevel:       0.3,
		EnableNoiseReduction: false, // Disabled by default, requires noise estimation
		NoiseReductionAlpha:  0.1,
	}
}

// PreprocessAudio applies configured preprocessing steps to audio samples
func PreprocessAudio(samples []float64, sampleRate int, config PreprocessingConfig) []float64 {
	if len(samples) == 0 {
		return samples
	}

	result := make([]float64, len(samples))
	copy(result, samples)

	// Step 1: High-pass filter to remove low-frequency noise
	if config.EnableHighPass {
		result = HighPassFilter(result, sampleRate, config.HighPassCutoff)
	}

	// Step 2: Band-pass filter to focus on drone frequencies
	if config.EnableBandPass {
		result = BandPassFilter(result, sampleRate, config.BandPassLow, config.BandPassHigh)
	}

	// Step 3: Automatic Gain Control
	if config.EnableAGC {
		result = ApplyAGC(result, config.AGCTargetLevel)
	}

	// Step 4: Spectral subtraction (if enabled and noise estimate available)
	// Note: This requires noise estimation which is complex, so we'll do a simple version
	if config.EnableNoiseReduction {
		result = SimpleNoiseReduction(result, sampleRate, config.NoiseReductionAlpha)
	}

	return result
}

// HighPassFilter removes frequencies below cutoff using a first-order IIR filter
func HighPassFilter(samples []float64, sampleRate int, cutoffHz float64) []float64 {
	if cutoffHz <= 0 || cutoffHz >= float64(sampleRate)/2 {
		return samples
	}

	rc := 1.0 / (2 * math.Pi * cutoffHz)
	dt := 1.0 / float64(sampleRate)
	alpha := rc / (rc + dt)

	filtered := make([]float64, len(samples))
	var prevOutput float64

	for i, x := range samples {
		if i == 0 {
			filtered[i] = x
		} else {
			filtered[i] = alpha * (prevOutput + x - samples[i-1])
		}
		prevOutput = filtered[i]
	}

	return filtered
}

// LowPassFilter removes frequencies above cutoff using a first-order IIR filter
func LowPassFilter(samples []float64, sampleRate int, cutoffHz float64) []float64 {
	if cutoffHz <= 0 || cutoffHz >= float64(sampleRate)/2 {
		return samples
	}

	rc := 1.0 / (2 * math.Pi * cutoffHz)
	dt := 1.0 / float64(sampleRate)
	alpha := dt / (rc + dt)

	filtered := make([]float64, len(samples))
	var prevOutput float64

	for i, x := range samples {
		if i == 0 {
			filtered[i] = x * alpha
		} else {
			filtered[i] = alpha*x + (1-alpha)*prevOutput
		}
		prevOutput = filtered[i]
	}

	return filtered
}

// BandPassFilter applies both high-pass and low-pass filters
func BandPassFilter(samples []float64, sampleRate int, lowHz, highHz float64) []float64 {
	result := HighPassFilter(samples, sampleRate, lowHz)
	result = LowPassFilter(result, sampleRate, highHz)
	return result
}

// ApplyAGC normalizes audio levels using Automatic Gain Control
func ApplyAGC(samples []float64, targetRMS float64) []float64 {
	if len(samples) == 0 {
		return samples
	}

	// Calculate current RMS
	var sumSquares float64
	for _, s := range samples {
		sumSquares += s * s
	}
	currentRMS := math.Sqrt(sumSquares / float64(len(samples)))

	if currentRMS == 0 || math.Abs(currentRMS-targetRMS) < 1e-6 {
		return samples
	}

	// Calculate gain factor
	gain := targetRMS / currentRMS

	// Apply gain with soft limiting to prevent clipping
	result := make([]float64, len(samples))
	for i, s := range samples {
		amplified := s * gain
		// Soft limiter: tanh provides smooth limiting
		if math.Abs(amplified) > 0.95 {
			result[i] = math.Tanh(amplified) * 0.95
		} else {
			result[i] = amplified
		}
	}

	return result
}

// SimpleNoiseReduction applies basic spectral subtraction
// This is a simplified version - full implementation would require noise estimation
func SimpleNoiseReduction(samples []float64, sampleRate int, alpha float64) []float64 {
	if len(samples) < 1024 {
		return samples // Too short for meaningful processing
	}

	// Estimate noise from first 10% of signal (assuming it's relatively quiet)
	noiseLength := len(samples) / 10
	if noiseLength < 512 {
		noiseLength = 512
	}

	noiseEstimate := estimateNoiseFloor(samples[:noiseLength], sampleRate)

	// Apply spectral subtraction in frequency domain
	// For simplicity, we'll do a basic time-domain approach
	// Full implementation would use FFT-based spectral subtraction
	result := make([]float64, len(samples))
	noiseThreshold := noiseEstimate * (1.0 + alpha)

	for i, s := range samples {
		if math.Abs(s) > noiseThreshold {
			// Reduce noise component
			result[i] = s - math.Copysign(noiseEstimate*alpha, s)
		} else {
			// Below threshold, reduce more aggressively
			result[i] = s * (1.0 - alpha*2.0)
		}
	}

	return result
}

// estimateNoiseFloor estimates the noise floor from a sample segment
func estimateNoiseFloor(samples []float64, sampleRate int) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	// Calculate RMS of the segment
	var sumSquares float64
	for _, s := range samples {
		sumSquares += s * s
	}

	return math.Sqrt(sumSquares / float64(len(samples)))
}

// EstimateSNR estimates signal-to-noise ratio in dB
func EstimateSNR(samples []float64) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	// Estimate noise from first 10% (assuming quiet period)
	noiseLength := len(samples) / 10
	if noiseLength < 512 {
		noiseLength = 512
	}
	if noiseLength > len(samples) {
		noiseLength = len(samples)
	}

	noisePower := estimateNoiseFloor(samples[:noiseLength], 44100)
	noisePower = noisePower * noisePower

	// Estimate signal power from entire sample
	var signalPower float64
	for _, s := range samples {
		signalPower += s * s
	}
	signalPower /= float64(len(samples))

	if noisePower == 0 {
		return 100.0 // Very high SNR if no noise detected
	}

	snr := signalPower / noisePower
	if snr <= 0 {
		return -100.0 // Very low SNR
	}

	// Convert to dB
	return 10.0 * math.Log10(snr)
}

// AdaptiveThreshold calculates confidence threshold based on SNR
func AdaptiveThreshold(baseThreshold float64, snrDb float64) float64 {
	// Lower SNR = higher threshold (more conservative)
	// Higher SNR = can use lower threshold (more sensitive)

	// SNR ranges:
	// < 10 dB: Very noisy, increase threshold by 0.15
	// 10-20 dB: Moderate noise, increase threshold by 0.10
	// 20-30 dB: Good SNR, increase threshold by 0.05
	// > 30 dB: Excellent SNR, use base threshold

	var adjustment float64
	if snrDb < 10 {
		adjustment = 0.15
	} else if snrDb < 20 {
		adjustment = 0.10
	} else if snrDb < 30 {
		adjustment = 0.05
	} else {
		adjustment = 0.0
	}

	adjustedThreshold := baseThreshold + adjustment
	// Clamp between 0.5 and 0.9
	if adjustedThreshold < 0.5 {
		return 0.5
	}
	if adjustedThreshold > 0.9 {
		return 0.9
	}

	return adjustedThreshold
}

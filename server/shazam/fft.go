package shazam

// Fast Fourier Transform (FFT)
//
// This package implements the Cooley-Tukey FFT algorithm, which efficiently converts
// a time-domain signal into its frequency-domain representation.
//
// How FFT Works:
//
// 1. Purpose:
//    - Converts audio samples (amplitude over time) into frequency components
//    - Reveals which frequencies are present and their magnitudes
//    - Essential for spectral analysis and feature extraction
//
// 2. Algorithm (Cooley-Tukey Radix-2):
//    - Divide-and-conquer approach: recursively splits signal in half
//    - Even-indexed samples → one half, odd-indexed → other half
//    - Recursively compute FFT of each half
//    - Combine results using twiddle factors (complex exponentials)
//
// 3. Twiddle Factors:
//    - W_N^k = e^(-2πik/N) = cos(-2πk/N) + i*sin(-2πk/N)
//    - Rotate frequency components to combine even/odd halves
//    - Creates frequency bins representing different frequency ranges
//
// 4. Output:
//    - Array of complex numbers representing frequency spectrum
//    - Magnitude = strength of frequency component
//    - Phase = timing relationship of frequency component
//    - Used to compute spectral features (centroid, bandwidth, etc.)
//
// The FFT is the core mathematical operation that enables:
// - Feature extraction (spectral centroid, rolloff, etc.)
// - Spectrogram generation (time-frequency analysis)
// - Peak detection (finding distinctive frequencies)
// - All audio fingerprinting and classification tasks
//
// For better understanding, refer to: https://www.youtube.com/watch?v=spUNpyF58BY

import (
	"math"
)

func FFT(input []float64) []complex128 {
	complexArray := make([]complex128, len(input))
	for i, v := range input {
		complexArray[i] = complex(v, 0)
	}

	fftResult := make([]complex128, len(complexArray))
	copy(fftResult, complexArray)
	return recursiveFFT(fftResult)
}

func recursiveFFT(complexArray []complex128) []complex128 {
	N := len(complexArray)
	if N <= 1 {
		return complexArray
	}

	even := make([]complex128, N/2)
	odd := make([]complex128, N/2)
	for i := 0; i < N/2; i++ {
		even[i] = complexArray[2*i]
		odd[i] = complexArray[2*i+1]
	}

	even = recursiveFFT(even)
	odd = recursiveFFT(odd)

	fftResult := make([]complex128, N)
	for k := 0; k < N/2; k++ {
		t := complex(math.Cos(-2*math.Pi*float64(k)/float64(N)), math.Sin(-2*math.Pi*float64(k)/float64(N)))
		fftResult[k] = even[k] + t*odd[k]
		fftResult[k+N/2] = even[k] - t*odd[k]
	}

	return fftResult
}

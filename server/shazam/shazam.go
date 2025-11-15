//go:build !js && !wasm
// +build !js,!wasm

package shazam

// Audio Matching Algorithm
//
// This package implements the matching logic for audio fingerprinting. It finds
// songs in the database that match an input audio sample.
//
// Matching Process:
//
// 1. Fingerprint Generation:
//    - Input audio is converted to spectrogram (time-frequency representation)
//    - Peaks are extracted from the spectrogram
//    - Fingerprints are generated from peak pairs (see fingerprint.go)
//
// 2. Database Lookup:
//    - For each fingerprint address, query database for matching songs
//    - Database returns all songs that have fingerprints at that address
//    - Each match includes: (song_id, anchor_time_in_song)
//
// 3. Temporal Alignment:
//    - For each candidate song, collect all matching timestamps
//    - Analyze relative timing between matches:
//      * Compute time differences between pairs of matches
//      * Compare sample time differences with song time differences
//      * If differences align (within 100ms tolerance), increment score
//
// 4. Scoring:
//    - Score = number of temporally aligned peak pairs
//    - Higher score = more consistent timing pattern = better match
//    - Songs are ranked by score (highest first)
//
// This matching algorithm is robust because:
// - It doesn't require exact time alignment (handles partial matches)
// - Temporal consistency filters out false positives
// - Works even when audio is noisy or distorted
//
// Note: This matching system was originally designed for song recognition but
// the underlying fingerprinting technique is also useful for drone acoustic
// signature identification.

import (
	"fmt"
	"math"
	"song-recognition/db"
	"song-recognition/utils"
	"sort"
	"time"
)

type Match struct {
	SongID     uint32
	SongTitle  string
	SongArtist string
	YouTubeID  string // Deprecated: kept for compatibility but not used in drone detection system
	Timestamp  uint32
	Score      float64
}

// FindMatches analyzes the audio sample to find matching songs in the database.
func FindMatches(audioSample []float64, audioDuration float64, sampleRate int) ([]Match, time.Duration, error) {
	startTime := time.Now()

	spectrogram, err := Spectrogram(audioSample, sampleRate)
	if err != nil {
		return nil, time.Since(startTime), fmt.Errorf("failed to get spectrogram of samples: %v", err)
	}

	peaks := ExtractPeaks(spectrogram, audioDuration)
	sampleFingerprint := Fingerprint(peaks, utils.GenerateUniqueID())

	sampleFingerprintMap := make(map[uint32]uint32)
	for address, couple := range sampleFingerprint {
		sampleFingerprintMap[address] = couple.AnchorTimeMs
	}

	matches, _, err := FindMatchesFGP(sampleFingerprintMap)

	return matches, time.Since(startTime), nil
}

// FindMatchesFGP uses the sample fingerprint to find matching songs in the database.
func FindMatchesFGP(sampleFingerprint map[uint32]uint32) ([]Match, time.Duration, error) {
	startTime := time.Now()
	logger := utils.GetLogger()

	addresses := make([]uint32, 0, len(sampleFingerprint))
	for address := range sampleFingerprint {
		addresses = append(addresses, address)
	}

	db, err := db.NewDBClient()
	if err != nil {
		return nil, time.Since(startTime), err
	}
	defer db.Close()

	m, err := db.GetCouples(addresses)
	if err != nil {
		return nil, time.Since(startTime), err
	}

	matches := map[uint32][][2]uint32{}        // songID -> [(sampleTime, dbTime)]
	timestamps := map[uint32]uint32{}          // songID -> earliest timestamp
	targetZones := map[uint32]map[uint32]int{} // songID -> timestamp -> count

	for address, couples := range m {
		for _, couple := range couples {
			matches[couple.SongID] = append(
				matches[couple.SongID],
				[2]uint32{sampleFingerprint[address], couple.AnchorTimeMs},
			)

			if existingTime, ok := timestamps[couple.SongID]; !ok || couple.AnchorTimeMs < existingTime {
				timestamps[couple.SongID] = couple.AnchorTimeMs
			}

			if _, ok := targetZones[couple.SongID]; !ok {
				targetZones[couple.SongID] = make(map[uint32]int)
			}
			targetZones[couple.SongID][couple.AnchorTimeMs]++
		}
	}

	// matches = filterMatches(10, matches, targetZones)

	scores := analyzeRelativeTiming(matches)

	var matchList []Match

	for songID, points := range scores {
		song, songExists, err := db.GetSongByID(songID)
		if !songExists {
			logger.Info(fmt.Sprintf("song with ID (%v) doesn't exist", songID))
			continue
		}
		if err != nil {
			logger.Info(fmt.Sprintf("failed to get song by ID (%v): %v", songID, err))
			continue
		}

		match := Match{songID, song.Title, song.Artist, song.YouTubeID, timestamps[songID], points}
		matchList = append(matchList, match)
	}

	sort.Slice(matchList, func(i, j int) bool {
		return matchList[i].Score > matchList[j].Score
	})

	return matchList, time.Since(startTime), nil
}

// filterMatches filters out matches that don't have enough
// target zones to meet the specified threshold
func filterMatches(
	threshold int,
	matches map[uint32][][2]uint32,
	targetZones map[uint32]map[uint32]int) map[uint32][][2]uint32 {

	// Filter out non target zones.
	// When a target zone has less than `targetZoneSize` anchor times, it is not considered a target zone.
	for songID, anchorTimes := range targetZones {
		for anchorTime, count := range anchorTimes {
			if count < targetZoneSize {
				delete(targetZones[songID], anchorTime)
			}
		}
	}

	filteredMatches := map[uint32][][2]uint32{}
	for songID, zones := range targetZones {
		if len(zones) >= threshold {
			filteredMatches[songID] = matches[songID]
		}
	}

	return filteredMatches
}

// analyzeRelativeTiming calculates a score for each song based on the
// relative timing between the song and the sample's anchor times.
func analyzeRelativeTiming(matches map[uint32][][2]uint32) map[uint32]float64 {
	scores := make(map[uint32]float64)
	for songID, times := range matches {
		count := 0
		for i := 0; i < len(times); i++ {
			for j := i + 1; j < len(times); j++ {
				sampleDiff := math.Abs(float64(times[i][0] - times[j][0]))
				dbDiff := math.Abs(float64(times[i][1] - times[j][1]))
				if math.Abs(sampleDiff-dbDiff) < 100 { // Allow some tolerance
					count++
				}
			}
		}
		scores[songID] = float64(count)
	}
	return scores
}

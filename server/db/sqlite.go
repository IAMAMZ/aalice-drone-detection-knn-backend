package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"song-recognition/models"
	"song-recognition/utils"
	"strings"

	_ "github.com/mattn/go-sqlite3" // SQLite driver registration
)

type SQLiteClient struct {
	db *sql.DB
}

func NewSQLiteClient(dataSourceName string) (*SQLiteClient, error) {
	// Extract the file path before query parameters
	dbPath := dataSourceName
	if idx := strings.Index(dataSourceName, "?"); idx != -1 {
		dbPath = dataSourceName[:idx]
	}

	// Create the directory if it doesn't exist (cross-platform)
	dbDir := filepath.Dir(dbPath)
	if dbDir != "." && dbDir != "" {
		if err := utils.CreateFolder(dbDir); err != nil {
			return nil, fmt.Errorf("error creating database directory: %s", err)
		}
	}

	// Add busy timeout param to DSN (milliseconds)
	if !strings.Contains(dataSourceName, "_busy_timeout") {
		if strings.Contains(dataSourceName, "?") {
			dataSourceName += "&_busy_timeout=5000" // 5 seconds
		} else {
			dataSourceName += "?_busy_timeout=5000"
		}
	}

	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("error connecting to SQLite: %s", err)
	}

	err = createTables(db)
	if err != nil {
		return nil, fmt.Errorf("error creating tables: %s", err)
	}

	return &SQLiteClient{db: db}, nil
}

// createTables creates the required tables if they don't exist
func createTables(db *sql.DB) error {
	createSongsTable := `
    CREATE TABLE IF NOT EXISTS songs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        artist TEXT NOT NULL,
        ytID TEXT,
        key TEXT NOT NULL UNIQUE
    );
    `

	createFingerprintsTable := `
    CREATE TABLE IF NOT EXISTS fingerprints (
        address INTEGER NOT NULL,
        anchorTimeMs INTEGER NOT NULL,
        songID INTEGER NOT NULL,
        PRIMARY KEY (address, anchorTimeMs, songID)
    );
    `

	createDetectionsTable := `
    CREATE TABLE IF NOT EXISTS detections (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        latitude REAL,
        longitude REAL,
        is_drone INTEGER NOT NULL DEFAULT 0,
        primary_type TEXT,
        primary_label TEXT,
        primary_category TEXT,
        confidence REAL NOT NULL DEFAULT 0,
        snr_db REAL,
        latency_ms REAL NOT NULL DEFAULT 0,
        predictions TEXT NOT NULL,
        metadata TEXT
    );
    CREATE INDEX IF NOT EXISTS idx_detections_timestamp ON detections(timestamp);
    CREATE INDEX IF NOT EXISTS idx_detections_location ON detections(latitude, longitude);
    `

	_, err := db.Exec(createSongsTable)
	if err != nil {
		return fmt.Errorf("error creating songs table: %s", err)
	}

	_, err = db.Exec(createFingerprintsTable)
	if err != nil {
		return fmt.Errorf("error creating fingerprints table: %s", err)
	}

	_, err = db.Exec(createDetectionsTable)
	if err != nil {
		return fmt.Errorf("error creating detections table: %s", err)
	}

	return nil
}

func (db *SQLiteClient) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}

func (db *SQLiteClient) StoreFingerprints(fingerprints map[uint32]models.Couple) error {
	tx, err := db.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %s", err)
	}

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO fingerprints (address, anchorTimeMs, songID) VALUES (?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error preparing statement: %s", err)
	}
	defer stmt.Close()

	for address, couple := range fingerprints {
		if _, err := stmt.Exec(address, couple.AnchorTimeMs, couple.SongID); err != nil {
			tx.Rollback()
			return fmt.Errorf("error executing statement: %s", err)
		}
	}

	return tx.Commit()
}

func (db *SQLiteClient) GetCouples(addresses []uint32) (map[uint32][]models.Couple, error) {
	couples := make(map[uint32][]models.Couple)

	for _, address := range addresses {
		rows, err := db.db.Query("SELECT anchorTimeMs, songID FROM fingerprints WHERE address = ?", address)
		if err != nil {
			return nil, fmt.Errorf("error querying database: %s", err)
		}

		var docCouples []models.Couple
		for rows.Next() {
			var couple models.Couple
			if err := rows.Scan(&couple.AnchorTimeMs, &couple.SongID); err != nil {
				rows.Close() // close before returning error
				return nil, fmt.Errorf("error scanning row: %s", err)
			}
			docCouples = append(docCouples, couple)
		}

		rows.Close() // close explicitly after reading

		couples[address] = docCouples
	}

	return couples, nil
}

func (db *SQLiteClient) TotalSongs() (int, error) {
	var count int
	err := db.db.QueryRow("SELECT COUNT(*) FROM songs").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting songs: %s", err)
	}
	return count, nil
}

func (db *SQLiteClient) RegisterSong(songTitle, songArtist, ytID string) (uint32, error) {
	tx, err := db.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("error starting transaction: %s", err)
	}

	stmt, err := tx.Prepare("INSERT INTO songs (id, title, artist, ytID, key) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("error preparing statement: %s", err)
	}
	defer stmt.Close()

	songID := utils.GenerateUniqueID()
	songKey := utils.GenerateSongKey(songTitle, songArtist)
	if _, err := stmt.Exec(songID, songTitle, songArtist, ytID, songKey); err != nil {
		tx.Rollback()
		// Check for constraint violation by examining error message (cross-platform compatible)
		errMsg := err.Error()
		if strings.Contains(errMsg, "UNIQUE constraint") || strings.Contains(errMsg, "constraint failed") {
			return 0, fmt.Errorf("song with ytID or key already exists: %v", err)
		}
		return 0, fmt.Errorf("failed to register song: %v", err)
	}

	return songID, tx.Commit()
}

var sqlitefilterKeys = "id | ytID | key"

// GetSong retrieves a song by filter key
func (s *SQLiteClient) GetSong(filterKey string, value interface{}) (Song, bool, error) {

	if !strings.Contains(sqlitefilterKeys, filterKey) {
		return Song{}, false, fmt.Errorf("invalid filter key")
	}

	query := fmt.Sprintf("SELECT title, artist, ytID FROM songs WHERE %s = ?", filterKey)

	row := s.db.QueryRow(query, value)

	var song Song
	err := row.Scan(&song.Title, &song.Artist, &song.YouTubeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return Song{}, false, nil
		}
		return Song{}, false, fmt.Errorf("failed to retrieve song: %s", err)
	}

	return song, true, nil
}

func (db *SQLiteClient) GetSongByID(songID uint32) (Song, bool, error) {
	return db.GetSong("id", songID)
}

func (db *SQLiteClient) GetSongByYTID(ytID string) (Song, bool, error) {
	return db.GetSong("ytID", ytID)
}

func (db *SQLiteClient) GetSongByKey(key string) (Song, bool, error) {
	return db.GetSong("key", key)
}

// DeleteSongByID deletes a song by ID
func (db *SQLiteClient) DeleteSongByID(songID uint32) error {
	_, err := db.db.Exec("DELETE FROM songs WHERE id = ?", songID)
	if err != nil {
		return fmt.Errorf("failed to delete song: %v", err)
	}
	return nil
}

// DeleteCollection deletes a collection (table) from the database
func (db *SQLiteClient) DeleteCollection(collectionName string) error {
	_, err := db.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", collectionName))
	if err != nil {
		return fmt.Errorf("error deleting collection: %v", err)
	}
	return nil
}

// StoreDetection stores a detection in the database
func (db *SQLiteClient) StoreDetection(detection *models.Detection) error {
	predictionsJSON, err := json.Marshal(detection.Predictions)
	if err != nil {
		return fmt.Errorf("error marshaling predictions: %s", err)
	}

	var metadataJSON *string
	if detection.Metadata != nil {
		metadataBytes, err := json.Marshal(detection.Metadata)
		if err != nil {
			return fmt.Errorf("error marshaling metadata: %s", err)
		}
		metadataStr := string(metadataBytes)
		metadataJSON = &metadataStr
	}

	isDroneInt := 0
	if detection.IsDrone {
		isDroneInt = 1
	}

	_, err = db.db.Exec(`
		INSERT INTO detections (
			timestamp, latitude, longitude, is_drone, primary_type, 
			primary_label, primary_category, confidence, snr_db, 
			latency_ms, predictions, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		detection.Timestamp,
		detection.Latitude,
		detection.Longitude,
		isDroneInt,
		detection.PrimaryType,
		detection.PrimaryLabel,
		detection.PrimaryCategory,
		detection.Confidence,
		detection.SNRDb,
		detection.LatencyMs,
		string(predictionsJSON),
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("error storing detection: %s", err)
	}
	return nil
}

// GetAllDetections retrieves all detections from the database
func (db *SQLiteClient) GetAllDetections() ([]models.Detection, error) {
	rows, err := db.db.Query(`
		SELECT id, timestamp, latitude, longitude, is_drone, primary_type,
		       primary_label, primary_category, confidence, snr_db, latency_ms,
		       predictions, metadata
		FROM detections
		ORDER BY timestamp DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying detections: %s", err)
	}
	defer rows.Close()

	var detections []models.Detection
	for rows.Next() {
		var d models.Detection
		var isDroneInt int
		var predictionsJSON string
		var metadataJSON *string

		err := rows.Scan(
			&d.ID,
			&d.Timestamp,
			&d.Latitude,
			&d.Longitude,
			&isDroneInt,
			&d.PrimaryType,
			&d.PrimaryLabel,
			&d.PrimaryCategory,
			&d.Confidence,
			&d.SNRDb,
			&d.LatencyMs,
			&predictionsJSON,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning detection: %s", err)
		}

		d.IsDrone = isDroneInt == 1
		d.Predictions = json.RawMessage(predictionsJSON)

		if metadataJSON != nil {
			err = json.Unmarshal([]byte(*metadataJSON), &d.Metadata)
			if err != nil {
				return nil, fmt.Errorf("error unmarshaling metadata: %s", err)
			}
		}

		detections = append(detections, d)
	}

	return detections, nil
}

// GetDetectionsByLocation retrieves detections within a radius of a location
func (db *SQLiteClient) GetDetectionsByLocation(lat, lng float64, radiusKm float64) ([]models.Detection, error) {
	// Using Haversine formula approximation for SQLite
	// This is a simplified version - for production, consider using PostGIS or similar
	rows, err := db.db.Query(`
		SELECT id, timestamp, latitude, longitude, is_drone, primary_type,
		       primary_label, primary_category, confidence, snr_db, latency_ms,
		       predictions, metadata
		FROM detections
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		  AND ABS(latitude - ?) < ? AND ABS(longitude - ?) < ?
		ORDER BY timestamp DESC
	`, lat, radiusKm/111.0, lng, radiusKm/(111.0*math.Cos(lat*math.Pi/180.0)))
	if err != nil {
		return nil, fmt.Errorf("error querying detections by location: %s", err)
	}
	defer rows.Close()

	var detections []models.Detection
	for rows.Next() {
		var d models.Detection
		var isDroneInt int
		var predictionsJSON string
		var metadataJSON *string

		err := rows.Scan(
			&d.ID,
			&d.Timestamp,
			&d.Latitude,
			&d.Longitude,
			&isDroneInt,
			&d.PrimaryType,
			&d.PrimaryLabel,
			&d.PrimaryCategory,
			&d.Confidence,
			&d.SNRDb,
			&d.LatencyMs,
			&predictionsJSON,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning detection: %s", err)
		}

		d.IsDrone = isDroneInt == 1
		d.Predictions = json.RawMessage(predictionsJSON)

		if metadataJSON != nil {
			err = json.Unmarshal([]byte(*metadataJSON), &d.Metadata)
			if err != nil {
				return nil, fmt.Errorf("error unmarshaling metadata: %s", err)
			}
		}

		detections = append(detections, d)
	}

	return detections, nil
}

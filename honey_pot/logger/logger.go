package logger

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"honeypot-go/config"
	"honeypot-go/models"

	_ "github.com/mattn/go-sqlite3"
)

// RequestLogger dual-writes structured attack telemetry to SQLite and JSONL.
type RequestLogger struct {
	db       *sql.DB
	jsonPath string
	mu       sync.Mutex
}

// NewRequestLogger opens (or creates) the SQLite database and JSONL file.
func NewRequestLogger() (*RequestLogger, error) {
	dbPath := config.AppConfig.LogDBPath
	jsonPath := config.AppConfig.LogJSONPath

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db log dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0755); err != nil {
		return nil, fmt.Errorf("create json log dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS requests (
        id TEXT PRIMARY KEY,
        timestamp DATETIME NOT NULL,
        ip TEXT NOT NULL,
        user_agent TEXT,
        method TEXT NOT NULL,
        endpoint TEXT NOT NULL,
        headers TEXT,
        body TEXT,
        response_code INTEGER,
        response_time INTEGER,
        ml_label TEXT,
        confidence REAL
    );
    CREATE INDEX IF NOT EXISTS idx_requests_ip ON requests(ip);
    CREATE INDEX IF NOT EXISTS idx_requests_label ON requests(ml_label);
    CREATE INDEX IF NOT EXISTS idx_requests_ts ON requests(timestamp);
    `

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	log.Printf("[logger] SQLite → %s | JSONL → %s", dbPath, jsonPath)
	return &RequestLogger{db: db, jsonPath: jsonPath}, nil
}

// LogRequest persists a request event to SQLite and appends a JSON line.
func (l *RequestLogger) LogRequest(logEntry models.RequestLog) error {
	headersJSON, err := json.Marshal(logEntry.Headers)
	if err != nil {
		headersJSON = []byte("{}")
	}

	query := `INSERT INTO requests (
        id, timestamp, ip, user_agent, method, endpoint,
        headers, body, response_code, response_time, ml_label, confidence
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = l.db.Exec(query,
		logEntry.ID,
		logEntry.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z"),
		logEntry.IP,
		logEntry.UserAgent,
		logEntry.Method,
		logEntry.Endpoint,
		string(headersJSON),
		logEntry.Body,
		logEntry.ResponseCode,
		logEntry.ResponseTime,
		logEntry.MLLabel,
		logEntry.Confidence,
	)
	if err != nil {
		log.Printf("[logger] sqlite insert failed: %v", err)
		return err
	}

	if err := l.appendToJSONFile(logEntry); err != nil {
		log.Printf("[logger] jsonl append failed: %v", err)
		// SQLite succeeded; surface JSON failure but do not roll back.
		return err
	}
	return nil
}

func (l *RequestLogger) appendToJSONFile(logEntry models.RequestLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.OpenFile(l.jsonPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open jsonl: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(logEntry)
	if err != nil {
		return fmt.Errorf("marshal jsonl: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write jsonl: %w", err)
	}
	return nil
}

// GetRequestsByIP returns recent request summaries for a source IP.
func (l *RequestLogger) GetRequestsByIP(ip string) ([]models.RequestLog, error) {
	query := `SELECT timestamp, method, endpoint, ml_label, confidence, response_code
              FROM requests WHERE ip = ? ORDER BY timestamp DESC LIMIT 100`
	rows, err := l.db.Query(query, ip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []models.RequestLog
	for rows.Next() {
		var req models.RequestLog
		var ts string
		if err := rows.Scan(&ts, &req.Method, &req.Endpoint, &req.MLLabel, &req.Confidence, &req.ResponseCode); err != nil {
			continue
		}
		req.IP = ip
		requests = append(requests, req)
	}
	return requests, nil
}

// Close releases the database handle.
func (l *RequestLogger) Close() error {
	return l.db.Close()
}

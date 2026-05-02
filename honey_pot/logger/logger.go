package logger

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"

	"honeypot-go/config"
	"honeypot-go/models"

	_ "github.com/mattn/go-sqlite3"
)

// Using SQLite for structured querying and better attacker profiling
type RequestLogger struct {
	db *sql.DB
}

func NewRequestLogger() (*RequestLogger, error) {
	// Ensure logs directory exists
	if err := os.MkdirAll("./logs", 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", config.AppConfig.LogDBPath)
	if err != nil {
		return nil, err
	}

	// Create table if not exists
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS requests (
        id TEXT PRIMARY KEY,
        timestamp DATETIME,
        ip TEXT,
        user_agent TEXT,
        method TEXT,
        endpoint TEXT,
        headers TEXT,
        body TEXT,
        response_code INTEGER,
        response_time INTEGER,
        ml_label TEXT,
        confidence REAL
    );`

	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, err
	}

	return &RequestLogger{db: db}, nil
}

func (l *RequestLogger) LogRequest(logEntry models.RequestLog) error {
	headersJSON, _ := json.Marshal(logEntry.Headers)

	query := `INSERT INTO requests (
        id, timestamp, ip, user_agent, method, endpoint, 
        headers, body, response_code, response_time, ml_label, confidence
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := l.db.Exec(query,
		logEntry.ID,
		logEntry.Timestamp,
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
		log.Printf("Failed to log request: %v", err)
		return err
	}

	// Also write to JSON file for easy parsing
	l.appendToJSONFile(logEntry)
	return nil
}

func (l *RequestLogger) appendToJSONFile(logEntry models.RequestLog) {
	file, err := os.OpenFile("./logs/requests.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open JSON log file: %v", err)
		return
	}
	defer file.Close()

	data, _ := json.Marshal(logEntry)
	file.WriteString(string(data) + "\n")
}

func (l *RequestLogger) GetRequestsByIP(ip string) ([]models.RequestLog, error) {
	query := `SELECT timestamp, method, endpoint, ml_label FROM requests WHERE ip = ? ORDER BY timestamp DESC LIMIT 100`
	rows, err := l.db.Query(query, ip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []models.RequestLog
	for rows.Next() {
		var req models.RequestLog
		err := rows.Scan(&req.Timestamp, &req.Method, &req.Endpoint, &req.MLLabel)
		if err != nil {
			continue
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func (l *RequestLogger) Close() error {
	return l.db.Close()
}

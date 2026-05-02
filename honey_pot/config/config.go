package config

import (
	"time"
)

type Config struct {
	ServerPort     string
	MLServiceURL   string
	LogDBPath      string
	RequestTimeout time.Duration
	MinDelay       time.Duration
	MaxDelay       time.Duration
	AttackDelay    time.Duration
}

var AppConfig = Config{
	ServerPort:     ":8080",
	MLServiceURL:   "http://localhost:5000/predict",
	LogDBPath:      "./logs/honeypot.db",
	RequestTimeout: 5 * time.Second,
	MinDelay:       100 * time.Millisecond,
	MaxDelay:       800 * time.Millisecond,
	AttackDelay:    2 * time.Second,
}

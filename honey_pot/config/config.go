package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime settings for the Go honeypot service.
// Values can be overridden via environment variables or a .env file.
type Config struct {
	ServerHost     string
	ServerPort     string
	MLServiceURL   string
	LogDBPath      string
	LogJSONPath    string
	RequestTimeout time.Duration
	MinDelay       time.Duration
	MaxDelay       time.Duration
	AttackDelay    time.Duration
	// Fallback used when the C++ inference service is unreachable.
	FallbackLabel      string
	FallbackConfidence float64
}

// AppConfig is the process-wide configuration populated at init.
var AppConfig = Config{
	ServerHost:         "0.0.0.0",
	ServerPort:         ":8080",
	MLServiceURL:       "http://localhost:5000/predict",
	LogDBPath:          "./logs/honeypot.db",
	LogJSONPath:        "./logs/requests.json",
	RequestTimeout:     5 * time.Second,
	MinDelay:           100 * time.Millisecond,
	MaxDelay:           800 * time.Millisecond,
	AttackDelay:        2 * time.Second,
	FallbackLabel:      "normal",
	FallbackConfidence: 0.5,
}

func init() {
	loadDotEnv(".env")
	loadDotEnv("config/.env")
	ApplyEnvOverrides()
}

// ApplyEnvOverrides reads known environment variables into AppConfig.
func ApplyEnvOverrides() {
	if v := os.Getenv("HONEYPOT_HOST"); v != "" {
		AppConfig.ServerHost = v
	}
	if v := os.Getenv("HONEYPOT_PORT"); v != "" {
		if !strings.HasPrefix(v, ":") {
			v = ":" + v
		}
		AppConfig.ServerPort = v
	}
	// Combined bind address override, e.g. "0.0.0.0:8080" or ":9090"
	if v := os.Getenv("HONEYPOT_ADDR"); v != "" {
		AppConfig.ServerPort = normalizeAddr(v)
	}
	if v := os.Getenv("ML_SERVICE_URL"); v != "" {
		AppConfig.MLServiceURL = v
	}
	if v := os.Getenv("LOG_DB_PATH"); v != "" {
		AppConfig.LogDBPath = v
	}
	if v := os.Getenv("LOG_JSON_PATH"); v != "" {
		AppConfig.LogJSONPath = v
	}
	if v := os.Getenv("REQUEST_TIMEOUT_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			AppConfig.RequestTimeout = time.Duration(ms) * time.Millisecond
		}
	}
	if v := os.Getenv("MIN_DELAY_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
			AppConfig.MinDelay = time.Duration(ms) * time.Millisecond
		}
	}
	if v := os.Getenv("MAX_DELAY_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
			AppConfig.MaxDelay = time.Duration(ms) * time.Millisecond
		}
	}
	if v := os.Getenv("ATTACK_DELAY_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
			AppConfig.AttackDelay = time.Duration(ms) * time.Millisecond
		}
	}
	if v := os.Getenv("FALLBACK_LABEL"); v != "" {
		AppConfig.FallbackLabel = strings.ToLower(v)
	}
	if v := os.Getenv("FALLBACK_CONFIDENCE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
			AppConfig.FallbackConfidence = f
		}
	}
}

// ListenAddr returns host+port suitable for http.Server.Addr.
func ListenAddr() string {
	port := AppConfig.ServerPort
	if strings.Contains(port, ":") && !strings.HasPrefix(port, ":") {
		return port // already "host:port"
	}
	host := AppConfig.ServerHost
	if host == "" || host == "0.0.0.0" {
		return port
	}
	return host + port
}

func normalizeAddr(v string) string {
	if strings.Contains(v, ":") {
		return v
	}
	return ":" + v
}

// loadDotEnv loads KEY=VALUE pairs from a file into the process environment
// without overwriting variables that are already set.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, val); err != nil {
				log.Printf("[config] failed to set %s from %s: %v", key, path, err)
			}
		}
	}
}

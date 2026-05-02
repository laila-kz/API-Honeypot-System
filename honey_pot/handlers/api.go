package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"honeypot-go/client"
	decision "honeypot-go/decisions"
	features "honeypot-go/feature"
	"honeypot-go/logger"
	"honeypot-go/models"

	"github.com/google/uuid"
)

type APIHandler struct {
	logger      *logger.RequestLogger
	featureExt  *features.FeatureExtractor
	mlClient    *client.MLClient
	decisionEng *decision.DecisionEngine
}

func NewAPIHandler() (*APIHandler, error) {
	l, err := logger.NewRequestLogger()
	if err != nil {
		return nil, err
	}

	return &APIHandler{
		logger:      l,
		featureExt:  features.NewFeatureExtractor(),
		mlClient:    client.NewMLClient(),
		decisionEng: decision.NewDecisionEngine(),
	}, nil
}

func (h *APIHandler) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extract client IP (considering proxies)
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ips := strings.Split(forwarded, ",")
			if len(ips) > 0 {
				ip = strings.TrimSpace(ips[0])
			}
		}

		// Read request body (if any)
		var body string
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				body = string(bodyBytes)
				// Restore body for further processing
				r.Body = io.NopCloser(strings.NewReader(body))
			}
		}

		// Extract features
		features := h.featureExt.ExtractFeatures(r, ip)

		// Get ML prediction
		mlResp, err := h.mlClient.Predict(features)
		if err != nil {
			log.Printf("ML prediction error: %v", err)
			mlResp = &models.MLResponse{Label: "normal", Confidence: 0.5}
		}

		// Get decision based on prediction
		statusCode, responseBody, delay := h.decisionEng.GetResponse(mlResp.Label, mlResp.Confidence)

		// Simulate latency (attacker profiling)
		time.Sleep(delay)

		// Write response
		w.Header().Set("Content-Type", "application/json")

		// Add fake admin hints for attackers (deception)
		if mlResp.Label == "attack" || mlResp.Label == "suspicious" {
			w.Header().Set("X-Debug-Path", "/admin/config.php")
			w.Header().Set("X-Backend-Server", "internal-10.0.0.5")
		}

		w.WriteHeader(statusCode)
		w.Write([]byte(responseBody))

		// Log the request
		logEntry := models.RequestLog{
			ID:           uuid.New().String(),
			Timestamp:    time.Now(),
			IP:           ip,
			UserAgent:    r.UserAgent(),
			Method:       r.Method,
			Endpoint:     r.URL.Path,
			Headers:      make(map[string]string),
			Body:         body,
			ResponseCode: statusCode,
			ResponseTime: time.Since(start).Milliseconds(),
			MLLabel:      mlResp.Label,
			Confidence:   mlResp.Confidence,
		}

		// Sanitize headers (remove sensitive ones)
		for key, values := range r.Header {
			if key != "Authorization" && key != "Cookie" {
				logEntry.Headers[key] = strings.Join(values, ",")
			} else {
				logEntry.Headers[key] = "[REDACTED]"
			}
		}

		if err := h.logger.LogRequest(logEntry); err != nil {
			log.Printf("Failed to log request: %v", err)
		}

		// Extra logging for attacks
		if mlResp.Label == "attack" {
			log.Printf("[ATTACK DETECTED] IP: %s, Endpoint: %s, Confidence: %.2f",
				ip, r.URL.Path, mlResp.Confidence)
		}
	}
}

func (h *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.Middleware(func(w http.ResponseWriter, r *http.Request) {
		// Handler logic is in middleware
	})(w, r)
}

func (h *APIHandler) Admin(w http.ResponseWriter, r *http.Request) {
	h.Middleware(func(w http.ResponseWriter, r *http.Request) {})(w, r)
}

func (h *APIHandler) Users(w http.ResponseWriter, r *http.Request) {
	h.Middleware(func(w http.ResponseWriter, r *http.Request) {})(w, r)
}

func (h *APIHandler) Auth(w http.ResponseWriter, r *http.Request) {
	h.Middleware(func(w http.ResponseWriter, r *http.Request) {})(w, r)
}

func (h *APIHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	h.Middleware(func(w http.ResponseWriter, r *http.Request) {})(w, r)
}

func (h *APIHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	h.Middleware(func(w http.ResponseWriter, r *http.Request) {})(w, r)
}

// Log viewer endpoint (bonus)
func (h *APIHandler) ViewLogs(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "IP parameter required", http.StatusBadRequest)
		return
	}

	logs, err := h.logger.GetRequestsByIP(ip)
	if err != nil {
		http.Error(w, "Failed to retrieve logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ip":       ip,
		"requests": logs,
		"total":    len(logs),
	})
}

func (h *APIHandler) Close() error {
	return h.logger.Close()
}

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

// APIHandler wires feature extraction, ML inference, deception, and logging.
type APIHandler struct {
	logger      *logger.RequestLogger
	featureExt  *features.FeatureExtractor
	mlClient    *client.MLClient
	decisionEng *decision.DecisionEngine
}

// NewAPIHandler constructs the honeypot request pipeline.
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

// Middleware runs the full classify → deceive → log pipeline for decoy routes.
func (h *APIHandler) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ips := strings.Split(forwarded, ",")
			if len(ips) > 0 {
				ip = strings.TrimSpace(ips[0])
			}
		}

		var body string
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
			if err == nil {
				body = string(bodyBytes)
				r.Body = io.NopCloser(strings.NewReader(body))
			}
		}

		feat := h.featureExt.ExtractFeatures(r, ip)

		mlResp, err := h.mlClient.Predict(feat)
		if err != nil {
			// Client already returns a safe fallback; keep serving decoy responses.
			log.Printf("[handlers] ML fallback active: %v", err)
			if mlResp == nil {
				mlResp = client.FallbackPrediction()
			}
		}

		statusCode, responseBody, delay := h.decisionEng.GetResponse(mlResp.Label, mlResp.Confidence)
		time.Sleep(delay)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", uuid.New().String())
		w.Header().Set("Server", "nginx/1.18.0")

		if mlResp.Label == "attack" || mlResp.Label == "suspicious" {
			w.Header().Set("X-Debug-Path", "/admin/config.php")
			w.Header().Set("X-Backend-Server", "internal-10.0.0.5")
			w.Header().Set("X-Powered-By", "PHP/7.4.3")
		}

		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(responseBody))

		logEntry := models.RequestLog{
			ID:           uuid.New().String(),
			Timestamp:    time.Now().UTC(),
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

		for key, values := range r.Header {
			lk := strings.ToLower(key)
			if lk == "authorization" || lk == "cookie" || lk == "x-api-key" {
				logEntry.Headers[key] = "[REDACTED]"
			} else {
				logEntry.Headers[key] = strings.Join(values, ",")
			}
		}

		if err := h.logger.LogRequest(logEntry); err != nil {
			log.Printf("[handlers] failed to log request: %v", err)
		}

		if mlResp.Label == "attack" {
			log.Printf("[ATTACK DETECTED] IP=%s endpoint=%s confidence=%.2f",
				ip, r.URL.Path, mlResp.Confidence)
		}

		if next != nil {
			next(w, r)
		}
	}
}

// Login handles /login decoy traffic.
func (h *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.Middleware(nil)(w, r)
}

// Admin handles /admin decoy traffic.
func (h *APIHandler) Admin(w http.ResponseWriter, r *http.Request) {
	h.Middleware(nil)(w, r)
}

// Users handles /api/v1/users decoy traffic.
func (h *APIHandler) Users(w http.ResponseWriter, r *http.Request) {
	h.Middleware(nil)(w, r)
}

// Auth handles /api/v1/auth decoy traffic.
func (h *APIHandler) Auth(w http.ResponseWriter, r *http.Request) {
	h.Middleware(nil)(w, r)
}

// ResetPassword handles /reset-password decoy traffic.
func (h *APIHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	h.Middleware(nil)(w, r)
}

// Dashboard handles /dashboard decoy traffic.
func (h *APIHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	h.Middleware(nil)(w, r)
}

// NotFound classifies unknown routes then returns a decoy deprecation 404.
func (h *APIHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip = strings.TrimSpace(ips[0])
		}
	}

	var body string
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err == nil {
			body = string(bodyBytes)
		}
	}

	feat := h.featureExt.ExtractFeatures(r, ip)
	mlResp, err := h.mlClient.Predict(feat)
	if err != nil || mlResp == nil {
		mlResp = client.FallbackPrediction()
	}

	_, _, delay := h.decisionEng.GetResponse(mlResp.Label, mlResp.Confidence)
	time.Sleep(delay)

	statusCode := http.StatusNotFound
	responseBody := `{"error":"Endpoint not found","message":"API version deprecated","docs":"/api/v1/swagger.json"}`

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "nginx/1.18.0")
	if mlResp.Label == "attack" || mlResp.Label == "suspicious" {
		w.Header().Set("X-Debug-Path", "/admin/config.php")
		w.Header().Set("X-Backend-Server", "internal-10.0.0.5")
	}
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(responseBody))

	logEntry := models.RequestLog{
		ID:           uuid.New().String(),
		Timestamp:    time.Now().UTC(),
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
	for key, values := range r.Header {
		lk := strings.ToLower(key)
		if lk == "authorization" || lk == "cookie" || lk == "x-api-key" {
			logEntry.Headers[key] = "[REDACTED]"
		} else {
			logEntry.Headers[key] = strings.Join(values, ",")
		}
	}
	_ = h.logger.LogRequest(logEntry)
}

// ViewLogs returns recent request summaries for an IP (operator convenience).
func (h *APIHandler) ViewLogs(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, `{"error":"ip query parameter required"}`, http.StatusBadRequest)
		return
	}

	logs, err := h.logger.GetRequestsByIP(ip)
	if err != nil {
		http.Error(w, `{"error":"failed to retrieve logs"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ip":       ip,
		"requests": logs,
		"total":    len(logs),
	})
}

// Close flushes logger resources.
func (h *APIHandler) Close() error {
	return h.logger.Close()
}

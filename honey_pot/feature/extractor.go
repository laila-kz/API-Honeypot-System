package features

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"honeypot-go/models"
)

type FeatureExtractor struct {
	mu              sync.RWMutex
	requestCounts   map[string]int             // IP -> count
	endpointCounts  map[string]int             // Endpoint -> count
	ipEndpoints     map[string]map[string]bool // IP -> set of endpoints
	lastRequestTime map[string]time.Time       // IP -> last request time
}

func NewFeatureExtractor() *FeatureExtractor {
	return &FeatureExtractor{
		requestCounts:   make(map[string]int),
		endpointCounts:  make(map[string]int),
		ipEndpoints:     make(map[string]map[string]bool),
		lastRequestTime: make(map[string]time.Time),
	}
}

func (fe *FeatureExtractor) ExtractFeatures(r *http.Request, ip string) models.FeatureVector {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	// Update counters
	fe.requestCounts[ip]++
	fe.endpointCounts[r.URL.Path]++

	if fe.ipEndpoints[ip] == nil {
		fe.ipEndpoints[ip] = make(map[string]bool)
	}
	fe.ipEndpoints[ip][r.URL.Path] = true

	// Calculate features
	now := time.Now()
	lastReq := fe.lastRequestTime[ip]
	timeDiff := 0.0
	if !lastReq.IsZero() {
		timeDiff = now.Sub(lastReq).Seconds()
	}
	fe.lastRequestTime[ip] = now

	// 1. Request rate per IP (normalized 0..1)
	// Use raw count normalized against a reasonable cap (100 requests)
	requestRate := float64(fe.requestCounts[ip])
	if requestRate > 100 {
		requestRate = 100 // Cap for normalization
	}

	// 2. Endpoint frequency (how many times this endpoint was hit)
	endpointFreq := float64(fe.endpointCounts[r.URL.Path]) / 100.0
	if endpointFreq > 1 {
		endpointFreq = 1
	}

	// 3. Unique endpoints per IP
	uniqueEndpoints := float64(len(fe.ipEndpoints[ip])) / 10.0
	if uniqueEndpoints > 1 {
		uniqueEndpoints = 1
	}

	// 4. Payload size (normalized to bytes)
	var payloadSize float64
	if r.Body != nil {
		// Estimate body size from Content-Length header
		if r.ContentLength > 0 {
			payloadSize = float64(r.ContentLength) / 10000.0
			if payloadSize > 1 {
				payloadSize = 1
			}
		} else {
			payloadSize = 0
		}
	}

	// 5. Header anomaly score
	headerAnomaly := fe.calculateHeaderAnomaly(r)

	// 6. Time between requests (normalized 0..1, 1 = long gap)
	timeBetween := 1.0
	if timeDiff > 0 {
		timeBetween = timeDiff / 60.0
		if timeBetween < 0 {
			timeBetween = 0
		}
		if timeBetween > 1 {
			timeBetween = 1
		}
	}

	// 7. Method distribution (GET ratio)
	methodDist := 0.5 // Default 50/50
	if r.Method == "GET" {
		methodDist = 1.0
	} else if r.Method == "POST" {
		methodDist = 0.0
	}

	return models.FeatureVector{
		RequestRatePerIP:     requestRate / 100,
		EndpointFrequency:    endpointFreq,
		UniqueEndpointsPerIP: uniqueEndpoints,
		PayloadSize:          payloadSize,
		HeaderAnomalyScore:   headerAnomaly,
		TimeBetweenRequests:  timeBetween,
		MethodDistribution:   methodDist,
	}
}

func (fe *FeatureExtractor) calculateHeaderAnomaly(r *http.Request) float64 {
	anomalies := 0.0
	total := 0.0

	// Check for unusual user-agents
	ua := r.UserAgent()
	if strings.Contains(ua, "sqlmap") || strings.Contains(ua, "nmap") ||
		strings.Contains(ua, "nikto") || (strings.Contains(ua, "curl") && !strings.Contains(ua, "Postman")) {
		anomalies += 1.0
	}
	total += 1.0

	// Check for suspicious headers (often used in attacks)
	suspiciousHeaders := []string{"X-Forwarded-For", "X-Originating-IP", "X-Remote-IP"}
	for _, header := range suspiciousHeaders {
		if r.Header.Get(header) != "" {
			anomalies += 0.5
		}
		total += 0.5
	}

	// Check for missing common headers
	if r.Header.Get("Accept") == "" {
		anomalies += 0.3
	}
	if r.Header.Get("Accept-Language") == "" {
		anomalies += 0.2
	}
	total += 0.5

	if total == 0 {
		return 0
	}
	score := anomalies / total
	if score > 1 {
		score = 1
	}
	return score
}

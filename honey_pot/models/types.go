package models

import (
	"time"
)

type RequestLog struct {
	ID           string            `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	IP           string            `json:"ip"`
	UserAgent    string            `json:"user_agent"`
	Method       string            `json:"method"`
	Endpoint     string            `json:"endpoint"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body,omitempty"`
	ResponseCode int               `json:"response_code"`
	ResponseTime int64             `json:"response_time_ms"`
	MLLabel      string            `json:"ml_label"`
	Confidence   float64           `json:"confidence"`
}

type FeatureVector struct {
	RequestRatePerIP     float64 `json:"request_rate_per_ip"`
	EndpointFrequency    float64 `json:"endpoint_frequency"`
	UniqueEndpointsPerIP float64 `json:"unique_endpoints_per_ip"`
	PayloadSize          float64 `json:"payload_size"`
	HeaderAnomalyScore   float64 `json:"header_anomaly_score"`
	TimeBetweenRequests  float64 `json:"time_between_requests"`
	MethodDistribution   float64 `json:"method_distribution"`
}

type MLRequest struct {
	Features FeatureVector `json:"features"`
}

type MLResponse struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

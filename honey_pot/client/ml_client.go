package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"honeypot-go/config"
	"honeypot-go/models"
)

// MLClient calls the C++ ONNX inference service over HTTP.
type MLClient struct {
	client *http.Client
	url    string
}

// NewMLClient constructs a client using AppConfig timeouts and endpoint.
func NewMLClient() *MLClient {
	return &MLClient{
		client: &http.Client{
			Timeout: config.AppConfig.RequestTimeout,
		},
		url: config.AppConfig.MLServiceURL,
	}
}

// FallbackPrediction returns the configured safe default when inference fails.
func FallbackPrediction() *models.MLResponse {
	return &models.MLResponse{
		Label:      config.AppConfig.FallbackLabel,
		Confidence: config.AppConfig.FallbackConfidence,
	}
}

// Predict sends a feature vector to the ML service.
// On timeout, network error, non-2xx, or decode failure it returns the
// configured fallback prediction along with a descriptive error so callers
// can log the incident while still serving a deceptive response.
func (mc *MLClient) Predict(features models.FeatureVector) (*models.MLResponse, error) {
	fallback := FallbackPrediction()

	reqBody := models.MLRequest{Features: features}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fallback, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := mc.client.Post(mc.url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[ml-client] inference unreachable (%s): %v — using fallback label=%s confidence=%.2f",
			mc.url, err, fallback.Label, fallback.Confidence)
		return fallback, fmt.Errorf("ml service unavailable: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fallback, fmt.Errorf("failed to read ml response body: %w", readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[ml-client] non-2xx status %d — using fallback", resp.StatusCode)
		return fallback, fmt.Errorf("ml service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var mlResp models.MLResponse
	if err := json.Unmarshal(bodyBytes, &mlResp); err != nil {
		log.Printf("[ml-client] decode failure — using fallback: %v", err)
		return fallback, fmt.Errorf("failed to decode response: %w", err)
	}

	mlResp.Label = strings.ToLower(strings.TrimSpace(mlResp.Label))
	if mlResp.Label == "" {
		return fallback, fmt.Errorf("ml response missing label")
	}
	if mlResp.Label != "normal" && mlResp.Label != "suspicious" && mlResp.Label != "attack" {
		log.Printf("[ml-client] unexpected label %q — using fallback", mlResp.Label)
		return fallback, fmt.Errorf("unexpected ml label: %s", mlResp.Label)
	}
	if mlResp.Confidence < 0 || mlResp.Confidence > 1 {
		mlResp.Confidence = fallback.Confidence
	}

	return &mlResp, nil
}

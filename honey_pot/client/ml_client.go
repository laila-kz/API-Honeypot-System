package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"honeypot-go/config"
	"honeypot-go/models"
)

type MLClient struct {
	client *http.Client
	url    string
}

func NewMLClient() *MLClient {
	return &MLClient{
		client: &http.Client{
			Timeout: config.AppConfig.RequestTimeout,
		},
		url: config.AppConfig.MLServiceURL,
	}
}

func (mc *MLClient) Predict(features models.FeatureVector) (*models.MLResponse, error) {
	reqBody := models.MLRequest{Features: features}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := mc.client.Post(mc.url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Service unavailable: return default and error so callers can decide
		return &models.MLResponse{Label: "normal", Confidence: 0.5}, fmt.Errorf("ml service unavailable: %w", err)
	}
	defer resp.Body.Close()

	// Require HTTP 2xx
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &models.MLResponse{Label: "normal", Confidence: 0.5}, fmt.Errorf("ml service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var mlResp models.MLResponse
	if err := json.NewDecoder(resp.Body).Decode(&mlResp); err != nil {
		return &models.MLResponse{Label: "normal", Confidence: 0.5}, fmt.Errorf("failed to decode response: %w", err)
	}

	// Validate required fields
	if mlResp.Label == "" {
		return &models.MLResponse{Label: "normal", Confidence: 0.5}, fmt.Errorf("ml response missing label")
	}

	return &mlResp, nil
}

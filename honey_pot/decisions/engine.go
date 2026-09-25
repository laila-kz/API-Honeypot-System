package decision

import (
	"math/rand"
	"net/http"
	"time"

	"honeypot-go/config"
)

// DecisionEngine maps ML labels to deceptive HTTP responses and latency.
type DecisionEngine struct{}

// NewDecisionEngine creates a deception/latency decision engine.
func NewDecisionEngine() *DecisionEngine {
	return &DecisionEngine{}
}

// GetResponse selects status, body, and intentional delay for a threat label.
func (de *DecisionEngine) GetResponse(label string, confidence float64) (statusCode int, body string, delay time.Duration) {
	_ = confidence // reserved for future confidence-weighted delays
	switch label {
	case "attack":
		return de.attackResponse()
	case "suspicious":
		return de.suspiciousResponse()
	default:
		return de.normalResponse()
	}
}

func (de *DecisionEngine) attackResponse() (int, string, time.Duration) {
	responses := []struct {
		status int
		body   string
	}{
		{500, `{"error": "database connection failed: SQL ERROR (Syntax error near 'union')", "details": "Query: SELECT * FROM users WHERE id = '%s'"}`},
		{403, `{"error": "Access denied", "message": "Invalid credentials or IP blocked", "remaining_attempts": 3}`},
		{503, `{"error": "Service temporarily unavailable", "reason": "Backend MySQL connection pool exhausted"}`},
		{400, `{"error": "Bad Request", "sql_injection_detected": true, "query_parsed": "SELECT * FROM admin_users"}`},
	}

	chosen := responses[rand.Intn(len(responses))]
	base := config.AppConfig.AttackDelay
	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
	return chosen.status, chosen.body, base + jitter
}

func (de *DecisionEngine) suspiciousResponse() (int, string, time.Duration) {
	min := config.AppConfig.MinDelay
	max := config.AppConfig.MaxDelay
	if max < min {
		max = min
	}
	span := int((max - min).Milliseconds())
	delay := min
	if span > 0 {
		delay += time.Duration(rand.Intn(span)) * time.Millisecond
	}

	bodies := []string{
		`{"status": "success", "data": {"message": "Authenticated"}}`,
		`{"status": "success", "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"}`,
		`{"status": "pending", "message": "Request queued for processing"}`,
	}

	return http.StatusOK, bodies[rand.Intn(len(bodies))], delay
}

func (de *DecisionEngine) normalResponse() (int, string, time.Duration) {
	min := config.AppConfig.MinDelay
	span := int((config.AppConfig.MaxDelay - min).Milliseconds())
	if span < 0 {
		span = 0
	}
	// Keep normal traffic snappy: use lower half of the configured delay band.
	half := span / 2
	delay := min
	if half > 0 {
		delay += time.Duration(rand.Intn(half)) * time.Millisecond
	}

	bodies := []string{
		`{"status": "success", "data": {"users": [{"id": 1, "name": "John Doe", "email": "john@example.com"}, {"id": 2, "name": "Jane Smith", "email": "jane@example.com"}]}}`,
		`{"status": "success", "message": "Login successful", "session_id": "abc123def456", "expires_in": 3600}`,
		`{"status": "success", "data": {"dashboard": {"stats": {"requests": 1234, "users": 567, "errors": 89}}}}`,
		`{"status": "success", "message": "Password reset email sent to registered address"}`,
	}

	return http.StatusOK, bodies[rand.Intn(len(bodies))], delay
}

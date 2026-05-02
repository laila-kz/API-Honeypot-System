package decision

import (
	"math/rand"
	"net/http"
	"time"
)

type DecisionEngine struct{}

func NewDecisionEngine() *DecisionEngine {
	return &DecisionEngine{}
}

func (de *DecisionEngine) GetResponse(label string, confidence float64) (statusCode int, body string, delay time.Duration) {
	switch label {
	case "attack":
		return de.attackResponse()
	case "suspicious":
		return de.suspiciousResponse()
	default: // normal
		return de.normalResponse()
	}
}

func (de *DecisionEngine) attackResponse() (int, string, time.Duration) {
	// Simulate fake vulnerabilities and misleading errors
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
	// Longer delay for attackers to waste their time
	delay := 2*time.Second + time.Duration(rand.Intn(1000))*time.Millisecond

	return chosen.status, chosen.body, delay
}

func (de *DecisionEngine) suspiciousResponse() (int, string, time.Duration) {
	// Slight delay for suspicious traffic
	delay := 500*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond

	bodies := []string{
		`{"status": "success", "data": {"message": "Authenticated"}}`,
		`{"status": "success", "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"}`,
		`{"status": "pending", "message": "Request queued for processing"}`,
	}

	return http.StatusOK, bodies[rand.Intn(len(bodies))], delay
}

func (de *DecisionEngine) normalResponse() (int, string, time.Duration) {
	// Normal response with fake data
	delay := time.Duration(rand.Intn(300)+100) * time.Millisecond

	bodies := []string{
		`{"status": "success", "data": {"users": [{"id": 1, "name": "John Doe", "email": "john@example.com"}, {"id": 2, "name": "Jane Smith", "email": "jane@example.com"}]}}`,
		`{"status": "success", "message": "Login successful", "session_id": "abc123def456", "expires_in": 3600}`,
		`{"status": "success", "data": {"dashboard": {"stats": {"requests": 1234, "users": 567, "errors": 89}}}}`,
		`{"status": "success", "message": "Password reset email sent to registered address"}`,
	}

	return http.StatusOK, bodies[rand.Intn(len(bodies))], delay
}

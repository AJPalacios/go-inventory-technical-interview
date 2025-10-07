package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/AJPalacios/inventory/internal/domain"
)

// MockLogger para tests
type MockLogger struct{}

func (m *MockLogger) Debug(msg string, fields ...map[string]interface{})            {}
func (m *MockLogger) Info(msg string, fields ...map[string]interface{})             {}
func (m *MockLogger) Warn(msg string, fields ...map[string]interface{})             {}
func (m *MockLogger) Error(msg string, err error, fields ...map[string]interface{}) {}
func (m *MockLogger) With(fields map[string]interface{}) domain.Logger              { return m }

// MockMetricsProvider para tests
type MockMetricsProvider struct{}

func (m *MockMetricsProvider) IncrementCounter(name string, labels map[string]string) {}
func (m *MockMetricsProvider) RecordDuration(name string, duration time.Duration, labels map[string]string) {
}

func TestRootHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/health", nil)

	rootHealthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Body.String(), "inventory-api")
	assert.Contains(t, w.Body.String(), "1.0.0")
}

func TestCorsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	middleware := corsMiddleware()
	middleware(c)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
}

func TestCorsMiddleware_OptionsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("OPTIONS", "/test", nil)

	middleware := corsMiddleware()
	middleware(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with existing X-Request-ID header
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Request-ID", "test-request-id")

	middleware := requestIDMiddleware()
	middleware(c)

	assert.Equal(t, "test-request-id", w.Header().Get("X-Request-ID"))
	assert.Equal(t, "test-request-id", c.GetString("request_id"))
}

func TestRequestIDMiddleware_GenerateID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test without existing X-Request-ID header
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	middleware := requestIDMiddleware()
	middleware(c)

	requestID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
	assert.Contains(t, requestID, "req_")
	assert.Equal(t, requestID, c.GetString("request_id"))
}

func TestMetricsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockMetrics := &MockMetricsProvider{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Writer.WriteHeader(http.StatusOK)

	middleware := metricsMiddleware(mockMetrics)
	middleware(c)

	// Test completes without errors
	assert.Equal(t, http.StatusOK, c.Writer.Status())
}

func TestMetricsMiddleware_NilProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	middleware := metricsMiddleware(nil)
	middleware(c)

	// Should not panic with nil provider
	assert.Equal(t, http.StatusOK, c.Writer.Status())
}

func TestGinLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLogger := &MockLogger{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test?param=value", nil)
	c.Set("request_id", "test-request-123")

	middleware := ginLogger(mockLogger)
	middleware(c)

	// Test completes without errors
	assert.True(t, true) // Test passes if no panic occurs
}

func TestRouterConfig(t *testing.T) {
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsProvider{}

	config := RouterConfig{
		InventoryHandler: nil, // Can be nil for this test
		Logger:           mockLogger,
		MetricsProvider:  mockMetrics,
	}

	assert.Equal(t, mockLogger, config.Logger)
	assert.Equal(t, mockMetrics, config.MetricsProvider)
}

func TestSetupRoutes_Basic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLogger := &MockLogger{}
	mockMetrics := &MockMetricsProvider{}

	engine := gin.New()
	config := RouterConfig{
		InventoryHandler: nil, // Handler can be nil for basic route setup test
		Logger:           mockLogger,
		MetricsProvider:  mockMetrics,
	}

	// Should not panic
	SetupRoutes(engine, config)

	// Verify that routes were added
	routes := engine.Routes()
	assert.Greater(t, len(routes), 0)

	// Check that health endpoint exists
	found := false
	for _, route := range routes {
		if route.Path == "/health" && route.Method == "GET" {
			found = true
			break
		}
	}
	assert.True(t, found, "Health endpoint should be registered")
}

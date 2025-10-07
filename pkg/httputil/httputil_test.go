package httputil

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestGetOrGenerateRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns existing request ID from header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		existingID := uuid.New().String()
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("X-Request-ID", existingID)

		result := GetOrGenerateRequestID(c)

		if result != existingID {
			t.Errorf("GetOrGenerateRequestID() = %v, want %v", result, existingID)
		}
	})

	t.Run("generates new request ID when header is missing", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		result := GetOrGenerateRequestID(c)

		// Verify it's a valid UUID
		if _, err := uuid.Parse(result); err != nil {
			t.Errorf("GetOrGenerateRequestID() returned invalid UUID: %v", err)
		}

		// Should set the header in response
		if w.Header().Get("X-Request-ID") != result {
			t.Errorf("Response header X-Request-ID = %v, want %v", w.Header().Get("X-Request-ID"), result)
		}
	})

	t.Run("generates different IDs for multiple calls", func(t *testing.T) {
		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest("GET", "/", nil)

		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request = httptest.NewRequest("GET", "/", nil)

		id1 := GetOrGenerateRequestID(c1)
		id2 := GetOrGenerateRequestID(c2)

		if id1 == id2 {
			t.Error("GetOrGenerateRequestID() should generate unique IDs for different requests")
		}
	})
}

// Package httputil provides common HTTP utilities.
//
// This package contains reusable HTTP-related functions for handling
// common patterns like request IDs, headers, and response formatting.
package httputil

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetOrGenerateRequestID retrieves the request ID from the header or generates a new one.
//
// This function checks for the X-Request-ID header and returns it if present.
// If not found, it generates a new UUID and sets it in the response header.
// This is useful for request tracing and idempotency.
func GetOrGenerateRequestID(c *gin.Context) string {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
		c.Header("X-Request-ID", requestID)
	}
	return requestID
}

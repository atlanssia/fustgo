package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/atlanssia/fustgo/internal/logger"
)

// LoggerMiddleware logs HTTP requests
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		requestID := c.GetString("request_id")

		// Log request
		logger.Info("[%s] %s %s - Status: %d - Latency: %v - IP: %s - RequestID: %s",
			method, path, c.Request.Proto, statusCode, latency, clientIP, requestID)

		// Log errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.Error("Request error: %v", err.Err)
			}
		}
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID is provided in header
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate new request ID
			requestID = uuid.New().String()
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// AuthMiddleware provides basic authentication (placeholder)
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement authentication logic
		// For now, just pass through
		c.Next()
	}
}

// RateLimitMiddleware provides rate limiting (placeholder)
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement rate limiting
		// For now, just pass through
		c.Next()
	}
}

// ErrorHandlerMiddleware handles errors
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			// Determine status code
			statusCode := c.Writer.Status()
			if statusCode == 200 {
				statusCode = 500 // Default to 500 if not set
			}

			// Return error response
			c.JSON(statusCode, gin.H{
				"error": err.Error(),
				"request_id": c.GetString("request_id"),
			})
		}
	}
}

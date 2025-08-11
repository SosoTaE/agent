package services

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// RateLimiter implements a simple rate limiter
type RateLimiter struct {
	mu                sync.Mutex
	requestsPerMinute int
	lastRequests      []time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rpm int) *RateLimiter {
	return &RateLimiter{
		requestsPerMinute: rpm,
		lastRequests:      make([]time.Time, 0),
	}
}

// Wait blocks until a request can be made within rate limits
func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Minute)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0)
	for _, t := range r.lastRequests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	r.lastRequests = validRequests

	// Check if we need to wait
	if len(r.lastRequests) >= r.requestsPerMinute {
		// Calculate wait time
		oldestRequest := r.lastRequests[0]
		waitUntil := oldestRequest.Add(time.Minute)
		waitDuration := waitUntil.Sub(now)

		if waitDuration > 0 {
			slog.Info("Rate limit reached, waiting...",
				"waitSeconds", waitDuration.Seconds(),
				"rpm", r.requestsPerMinute,
			)

			select {
			case <-time.After(waitDuration):
				// Continue after wait
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// Add current request
	r.lastRequests = append(r.lastRequests, now)
	return nil
}

// Global rate limiter for Voyage API (3 RPM for free tier)
var voyageRateLimiter = NewRateLimiter(3)

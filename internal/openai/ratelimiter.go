package openai

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting for API calls
type RateLimiter struct {
	requestsPerMinute int
	requestsPerDay    int
	
	// Minute-level tracking
	minuteTokens     int
	minuteLastRefill time.Time
	minuteMux        sync.Mutex
	
	// Day-level tracking
	dayTokens     int
	dayLastRefill time.Time
	dayMux        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute, requestsPerDay int) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		requestsPerMinute: requestsPerMinute,
		requestsPerDay:    requestsPerDay,
		minuteTokens:      requestsPerMinute,
		minuteLastRefill:  now,
		dayTokens:         requestsPerDay,
		dayLastRefill:     now,
	}
}

// Wait blocks until a request can be made according to rate limits
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		// Check if we can proceed
		if rl.canProceed() {
			rl.consumeToken()
			return nil
		}
		
		// Calculate wait time
		waitTime := rl.getWaitTime()
		if waitTime <= 0 {
			continue
		}
		
		// Wait or return if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue to next iteration
		}
	}
}

// canProceed checks if we have tokens available
func (rl *RateLimiter) canProceed() bool {
	rl.refillTokens()
	
	rl.minuteMux.Lock()
	minuteOk := rl.minuteTokens > 0
	rl.minuteMux.Unlock()
	
	rl.dayMux.Lock()
	dayOk := rl.dayTokens > 0
	rl.dayMux.Unlock()
	
	return minuteOk && dayOk
}

// consumeToken consumes one token from both buckets
func (rl *RateLimiter) consumeToken() {
	rl.minuteMux.Lock()
	if rl.minuteTokens > 0 {
		rl.minuteTokens--
	}
	rl.minuteMux.Unlock()
	
	rl.dayMux.Lock()
	if rl.dayTokens > 0 {
		rl.dayTokens--
	}
	rl.dayMux.Unlock()
}

// refillTokens refills token buckets based on elapsed time
func (rl *RateLimiter) refillTokens() {
	now := time.Now()
	
	// Refill minute bucket
	rl.minuteMux.Lock()
	if now.Sub(rl.minuteLastRefill) >= time.Minute {
		rl.minuteTokens = rl.requestsPerMinute
		rl.minuteLastRefill = now
	}
	rl.minuteMux.Unlock()
	
	// Refill day bucket
	rl.dayMux.Lock()
	if now.Sub(rl.dayLastRefill) >= 24*time.Hour {
		rl.dayTokens = rl.requestsPerDay
		rl.dayLastRefill = now
	}
	rl.dayMux.Unlock()
}

// getWaitTime calculates how long to wait before next attempt
func (rl *RateLimiter) getWaitTime() time.Duration {
	now := time.Now()
	
	rl.minuteMux.Lock()
	minuteWait := time.Duration(0)
	if rl.minuteTokens <= 0 {
		minuteWait = time.Minute - now.Sub(rl.minuteLastRefill)
	}
	rl.minuteMux.Unlock()
	
	rl.dayMux.Lock()
	dayWait := time.Duration(0)
	if rl.dayTokens <= 0 {
		dayWait = 24*time.Hour - now.Sub(rl.dayLastRefill)
	}
	rl.dayMux.Unlock()
	
	// Return the maximum wait time needed
	if dayWait > minuteWait {
		return dayWait
	}
	return minuteWait
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() (minuteTokens, dayTokens int) {
	rl.refillTokens()
	
	rl.minuteMux.Lock()
	minuteTokens = rl.minuteTokens
	rl.minuteMux.Unlock()
	
	rl.dayMux.Lock()
	dayTokens = rl.dayTokens
	rl.dayMux.Unlock()
	
	return
}

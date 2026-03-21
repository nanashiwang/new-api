package common

import (
	"testing"
	"time"
)

func TestInMemoryRateLimiterInitIdempotent(t *testing.T) {
	var limiter InMemoryRateLimiter

	limiter.Init(time.Minute)
	if limiter.store == nil {
		t.Fatal("store should be initialized after Init")
	}
	if !limiter.started {
		t.Fatal("cleanup goroutine should be started")
	}
	if limiter.expirationDuration != time.Minute {
		t.Fatalf("expirationDuration = %s, want %s", limiter.expirationDuration, time.Minute)
	}

	// Calling Init again should keep the longest retention window.
	limiter.Init(30 * time.Second)
	if limiter.expirationDuration != time.Minute {
		t.Fatalf("expirationDuration shrank to %s", limiter.expirationDuration)
	}
	limiter.Init(time.Hour)
	if limiter.store == nil {
		t.Fatal("store should still be initialized after second Init")
	}
	if limiter.expirationDuration != time.Hour {
		t.Fatalf("expirationDuration = %s, want %s", limiter.expirationDuration, time.Hour)
	}
}

func TestInMemoryRateLimiterRequest(t *testing.T) {
	var limiter InMemoryRateLimiter
	limiter.Init(time.Minute)

	// Allow 2 requests per 1 second window
	if !limiter.Request("key1", 2, 1) {
		t.Fatal("first request should be allowed")
	}
	if !limiter.Request("key1", 2, 1) {
		t.Fatal("second request should be allowed")
	}
	if limiter.Request("key1", 2, 1) {
		t.Fatal("third request should be denied within window")
	}
}

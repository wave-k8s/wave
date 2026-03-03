/*
Copyright 2018 Pusher Ltd. and Wave Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

import (
	"context"
	"math"
	"testing"
	"time"

	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/types"
)

func TestUpdateThrottler_InfiniteRate(t *testing.T) {
	// Test that infinite rate disables throttling
	ut := NewUpdateThrottler(rate.Limit(math.Inf(1)), 1)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}
	ctx := context.Background()

	// Multiple rapid updates should all succeed immediately with no delay
	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := ut.Wait(ctx, name); err != nil {
			t.Errorf("Wait %d failed: %v", i+1, err)
		}
	}
	elapsed := time.Since(start)

	// Should complete almost instantly (allow 100ms for overhead)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Infinite rate should not cause delays, took %v", elapsed)
	}
}

func TestUpdateThrottler_RateLimiting(t *testing.T) {
	// Test basic rate limiting: 10 updates per second (rate = 10.0)
	// With burst=1, first update is immediate, second should be delayed
	ut := NewUpdateThrottler(rate.Limit(10.0), 1)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}
	ctx := context.Background()

	// First update should be immediate (uses burst token)
	start := time.Now()
	if err := ut.Wait(ctx, name); err != nil {
		t.Fatalf("First wait failed: %v", err)
	}
	firstElapsed := time.Since(start)
	if firstElapsed > 10*time.Millisecond {
		t.Errorf("First update should be immediate, took %v", firstElapsed)
	}

	// Second update should be delayed by ~100ms (1/10 second)
	start = time.Now()
	if err := ut.Wait(ctx, name); err != nil {
		t.Fatalf("Second wait failed: %v", err)
	}
	secondElapsed := time.Since(start)
	if secondElapsed < 50*time.Millisecond {
		t.Errorf("Second update should be delayed, but took only %v", secondElapsed)
	}
}

func TestUpdateThrottler_BurstAllowance(t *testing.T) {
	// Test burst: rate=1.0 (1 per second), burst=3 allows 3 immediate updates
	ut := NewUpdateThrottler(rate.Limit(1.0), 3)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}
	ctx := context.Background()

	// First 3 updates should be immediate (use burst tokens)
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := ut.Wait(ctx, name); err != nil {
			t.Fatalf("Wait %d failed: %v", i+1, err)
		}
	}
	burstElapsed := time.Since(start)
	if burstElapsed > 50*time.Millisecond {
		t.Errorf("First 3 updates should be immediate (burst), took %v", burstElapsed)
	}

	// 4th update should be delayed by ~1 second
	start = time.Now()
	if err := ut.Wait(ctx, name); err != nil {
		t.Fatalf("4th wait failed: %v", err)
	}
	fourthElapsed := time.Since(start)
	if fourthElapsed < 500*time.Millisecond {
		t.Errorf("4th update should be delayed by ~1 second, but took only %v", fourthElapsed)
	}
}

func TestUpdateThrottler_MultipleInstances(t *testing.T) {
	// Test that different instances share the same global rate limiter
	ut := NewUpdateThrottler(rate.Limit(10.0), 1)
	name1 := types.NamespacedName{Namespace: "test", Name: "deployment1"}
	name2 := types.NamespacedName{Namespace: "test", Name: "deployment2"}
	ctx := context.Background()

	// First instance uses the burst token
	start := time.Now()
	if err := ut.Wait(ctx, name1); err != nil {
		t.Fatalf("Wait for name1 failed: %v", err)
	}
	firstElapsed := time.Since(start)
	if firstElapsed > 10*time.Millisecond {
		t.Errorf("First update should be immediate (uses burst), took %v", firstElapsed)
	}

	// Second instance should be delayed since burst token was used
	start = time.Now()
	if err := ut.Wait(ctx, name2); err != nil {
		t.Fatalf("Wait for name2 failed: %v", err)
	}
	secondElapsed := time.Since(start)
	if secondElapsed < 50*time.Millisecond {
		t.Errorf("Second update should be delayed (global rate limit), took only %v", secondElapsed)
	}
}

func TestUpdateThrottler_ContextCancellation(t *testing.T) {
	// Test that context cancellation stops waiting
	ut := NewUpdateThrottler(rate.Limit(0.1), 1) // Very slow: 1 update per 10 seconds
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}

	// Use up the burst token
	if err := ut.Wait(context.Background(), name); err != nil {
		t.Fatalf("Initial wait failed: %v", err)
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This wait should be cancelled by context timeout
	start := time.Now()
	err := ut.Wait(ctx, name)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("Should fail quickly on context cancellation, took %v", elapsed)
	}
}

func TestUpdateThrottler_DefaultValues(t *testing.T) {
	// Test that invalid rate/burst values use defaults

	// Zero rate should use default
	ut := NewUpdateThrottler(0, 1)
	if ut.rate != DefaultUpdateRate {
		t.Errorf("Expected default rate %v, got %v", DefaultUpdateRate, ut.rate)
	}

	// Negative rate should use default
	ut = NewUpdateThrottler(-1, 1)
	if ut.rate != DefaultUpdateRate {
		t.Errorf("Expected default rate %v, got %v", DefaultUpdateRate, ut.rate)
	}

	// Zero burst should use default
	ut = NewUpdateThrottler(1.0, 0)
	if ut.burst != DefaultUpdateBurst {
		t.Errorf("Expected default burst %d, got %d", DefaultUpdateBurst, ut.burst)
	}

	// Negative burst should use default
	ut = NewUpdateThrottler(1.0, -1)
	if ut.burst != DefaultUpdateBurst {
		t.Errorf("Expected default burst %d, got %d", DefaultUpdateBurst, ut.burst)
	}

	// Valid values should be preserved
	ut = NewUpdateThrottler(5.0, 10)
	if ut.rate != 5.0 {
		t.Errorf("Expected rate 5.0, got %v", ut.rate)
	}
	if ut.burst != 10 {
		t.Errorf("Expected burst 10, got %d", ut.burst)
	}
}

func TestUpdateThrottler_PreventChurn(t *testing.T) {
	// Test the original issue #182: rapid updates causing deployment churn
	// Rate of 0.1 = 1 update per 10 seconds
	ut := NewUpdateThrottler(rate.Limit(0.1), 1)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}
	ctx := context.Background()

	// First update should be immediate
	start := time.Now()
	if err := ut.Wait(ctx, name); err != nil {
		t.Fatalf("First wait failed: %v", err)
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Error("First update should be immediate")
	}

	// Simulate rapid reconciliations (buggy controller scenario)
	// Second update should be delayed by ~10 seconds
	// We'll use a timeout to avoid actually waiting 10 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := ut.Wait(ctx, name)
	if err == nil {
		t.Error("Expected rate limiting to delay second update")
	}
}

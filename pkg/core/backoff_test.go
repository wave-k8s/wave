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
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

const testInterval = 100 * time.Millisecond

func TestUpdateThrottler_FirstUpdate(t *testing.T) {
	ut := NewUpdateThrottler(testInterval)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}

	shouldUpdate, requeueAfter := ut.ShouldUpdate(name)

	if !shouldUpdate {
		t.Error("First update should be allowed")
	}
	if requeueAfter != 0 {
		t.Errorf("First update should not require requeue, got %v", requeueAfter)
	}

	// Verify state was created
	_, exists := ut.GetLastUpdateTime(name)
	if !exists {
		t.Error("State should exist after first update")
	}
}

func TestUpdateThrottler_SecondUpdateThrottled(t *testing.T) {
	ut := NewUpdateThrottler(testInterval)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}

	// First update
	shouldUpdate, _ := ut.ShouldUpdate(name)
	if !shouldUpdate {
		t.Fatal("First update should be allowed")
	}

	// Immediate second update should be blocked
	shouldUpdate, requeueAfter := ut.ShouldUpdate(name)
	if shouldUpdate {
		t.Error("Second immediate update should be blocked by throttling")
	}
	if requeueAfter <= 0 {
		t.Error("Should return positive requeue duration")
	}
	if requeueAfter > testInterval {
		t.Errorf("Requeue duration should be <= %v, got %v", testInterval, requeueAfter)
	}
}

func TestUpdateThrottler_AllUpdatesSubjectToThrottling(t *testing.T) {
	ut := NewUpdateThrottler(testInterval)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}

	// First update
	ut.ShouldUpdate(name)

	// Multiple rapid updates should all be blocked
	for i := 0; i < 5; i++ {
		shouldUpdate, requeueAfter := ut.ShouldUpdate(name)
		if shouldUpdate {
			t.Errorf("Rapid update %d should be blocked by throttling", i+2)
		}
		if requeueAfter <= 0 {
			t.Errorf("Update %d should have positive requeue duration", i+2)
		}
	}
}

func TestUpdateThrottler_RapidUpdatesAreThrottled(t *testing.T) {
	ut := NewUpdateThrottler(testInterval)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}

	// Simulate the issue #182 scenario: rapid secret changes causing rapid reconciliations
	// First update allowed
	shouldUpdate, _ := ut.ShouldUpdate(name)
	if !shouldUpdate {
		t.Fatal("First update should be allowed")
	}

	// Simulate multiple rapid reconciliations (e.g., buggy controller updating secrets)
	for i := 0; i < 5; i++ {
		shouldUpdate, requeueAfter := ut.ShouldUpdate(name)
		if shouldUpdate {
			t.Errorf("Rapid update %d should be blocked by throttling", i+2)
		}
		if requeueAfter <= 0 {
			t.Errorf("Update %d should have positive requeue duration", i+2)
		}
	}

	// After throttle period, update should be allowed
	ut.states[name].lastUpdateTime = time.Now().Add(-testInterval - time.Millisecond)
	shouldUpdate, requeueAfter := ut.ShouldUpdate(name)
	if !shouldUpdate {
		t.Error("Update should be allowed after throttle period")
	}
	if requeueAfter != 0 {
		t.Error("Should not require requeue after throttle period")
	}
}

func TestUpdateThrottler_AfterInterval(t *testing.T) {
	ut := NewUpdateThrottler(testInterval)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}

	// First update
	shouldUpdate, _ := ut.ShouldUpdate(name)
	if !shouldUpdate {
		t.Fatal("First update should be allowed")
	}

	// Wait for the interval to pass
	time.Sleep(testInterval + 10*time.Millisecond)

	// Second update should now be allowed
	shouldUpdate, requeueAfter := ut.ShouldUpdate(name)
	if !shouldUpdate {
		t.Error("Update should be allowed after interval has passed")
	}
	if requeueAfter != 0 {
		t.Errorf("Should not require requeue, got %v", requeueAfter)
	}
}

func TestUpdateThrottler_Reset(t *testing.T) {
	ut := NewUpdateThrottler(testInterval)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}

	// Create some state
	ut.ShouldUpdate(name)

	// Verify state exists
	_, exists := ut.GetLastUpdateTime(name)
	if !exists {
		t.Fatal("State should exist")
	}

	// Reset
	ut.Reset(name)

	// Verify state is gone
	_, exists = ut.GetLastUpdateTime(name)
	if exists {
		t.Error("State should not exist after reset")
	}
}

func TestUpdateThrottler_RecordUpdate(t *testing.T) {
	ut := NewUpdateThrottler(testInterval)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}

	// Create initial state
	ut.ShouldUpdate(name)

	// Get initial time
	initialTime, _ := ut.GetLastUpdateTime(name)

	// Small delay
	time.Sleep(10 * time.Millisecond)

	// Record update
	ut.RecordUpdate(name)

	// Get updated time
	updatedTime, exists := ut.GetLastUpdateTime(name)
	if !exists {
		t.Fatal("State should still exist")
	}

	if !updatedTime.After(initialTime) {
		t.Error("Update time should be after initial time")
	}
}

func TestUpdateThrottler_MultipleInstances(t *testing.T) {
	ut := NewUpdateThrottler(testInterval)
	name1 := types.NamespacedName{Namespace: "test", Name: "deployment1"}
	name2 := types.NamespacedName{Namespace: "test", Name: "deployment2"}

	// Update both instances
	shouldUpdate1, _ := ut.ShouldUpdate(name1)
	shouldUpdate2, _ := ut.ShouldUpdate(name2)

	if !shouldUpdate1 || !shouldUpdate2 {
		t.Error("Both first updates should be allowed")
	}

	// Verify both have independent state
	_, exists1 := ut.GetLastUpdateTime(name1)
	_, exists2 := ut.GetLastUpdateTime(name2)

	if !exists1 || !exists2 {
		t.Error("Both instances should have state")
	}

	// Block second update for name1
	shouldUpdate1, requeueAfter1 := ut.ShouldUpdate(name1)
	if shouldUpdate1 {
		t.Error("Second update for name1 should be blocked")
	}
	if requeueAfter1 <= 0 {
		t.Error("Should have positive requeue duration for name1")
	}

	// name2 should still be independent
	shouldUpdate2, requeueAfter2 := ut.ShouldUpdate(name2)
	if shouldUpdate2 {
		t.Error("Second update for name2 should also be blocked")
	}
	if requeueAfter2 <= 0 {
		t.Error("Should have positive requeue duration for name2")
	}
}

func TestUpdateThrottler_DefaultInterval(t *testing.T) {
	// Test with zero interval (should use default)
	ut := NewUpdateThrottler(0)
	if ut.minUpdateInterval != DefaultMinUpdateInterval {
		t.Errorf("Expected default interval %v, got %v", DefaultMinUpdateInterval, ut.minUpdateInterval)
	}

	// Test with negative interval (should use default)
	ut = NewUpdateThrottler(-1 * time.Second)
	if ut.minUpdateInterval != DefaultMinUpdateInterval {
		t.Errorf("Expected default interval %v, got %v", DefaultMinUpdateInterval, ut.minUpdateInterval)
	}

	// Test with valid interval
	customInterval := 5 * time.Second
	ut = NewUpdateThrottler(customInterval)
	if ut.minUpdateInterval != customInterval {
		t.Errorf("Expected custom interval %v, got %v", customInterval, ut.minUpdateInterval)
	}
}

func TestUpdateThrottler_ConsistentThrottling(t *testing.T) {
	ut := NewUpdateThrottler(testInterval)
	name := types.NamespacedName{Namespace: "test", Name: "deployment"}

	// First update
	ut.ShouldUpdate(name)

	// Check that throttling remains consistent regardless of config hash changes
	// This addresses issue #182 where changing hashes shouldn't bypass throttling
	shouldUpdate, _ := ut.ShouldUpdate(name)
	if shouldUpdate {
		t.Error("Update should be throttled regardless of hash changes")
	}

	// After interval, update should be allowed
	ut.states[name].lastUpdateTime = time.Now().Add(-testInterval - time.Millisecond)
	shouldUpdate, _ = ut.ShouldUpdate(name)
	if !shouldUpdate {
		t.Error("Update should be allowed after interval")
	}
}

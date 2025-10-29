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
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

const (
	// DefaultMinUpdateInterval is the default minimum time between updates (10 seconds)
	DefaultMinUpdateInterval = 10 * time.Second
)

// updateState tracks the last update time for a single instance
type updateState struct {
	lastUpdateTime time.Time
}

// UpdateThrottler manages update throttling for multiple instances
type UpdateThrottler struct {
	states            map[types.NamespacedName]*updateState
	mutex             sync.RWMutex
	minUpdateInterval time.Duration
}

// NewUpdateThrottler creates a new UpdateThrottler with the specified minimum interval
func NewUpdateThrottler(minUpdateInterval time.Duration) *UpdateThrottler {
	if minUpdateInterval <= 0 {
		minUpdateInterval = DefaultMinUpdateInterval
	}
	return &UpdateThrottler{
		states:            make(map[types.NamespacedName]*updateState),
		minUpdateInterval: minUpdateInterval,
	}
}

// ShouldUpdate checks if an instance should be updated based on the minimum interval
// Returns (shouldUpdate, requeueAfter)
func (ut *UpdateThrottler) ShouldUpdate(name types.NamespacedName) (bool, time.Duration) {
	ut.mutex.Lock()
	defer ut.mutex.Unlock()

	state, exists := ut.states[name]
	now := time.Now()

	// First update or no state
	if !exists {
		ut.states[name] = &updateState{
			lastUpdateTime: now,
		}
		return true, 0
	}

	// Check if enough time has passed since last update
	elapsed := now.Sub(state.lastUpdateTime)

	if elapsed < ut.minUpdateInterval {
		// Still within minimum interval - delay update
		remaining := ut.minUpdateInterval - elapsed
		return false, remaining
	}

	// Minimum interval has passed, allow update
	state.lastUpdateTime = now
	return true, 0
}

// RecordUpdate records a successful update (updates the timestamp)
func (ut *UpdateThrottler) RecordUpdate(name types.NamespacedName) {
	ut.mutex.Lock()
	defer ut.mutex.Unlock()

	if state, exists := ut.states[name]; exists {
		state.lastUpdateTime = time.Now()
	}
}

// Reset removes the throttling state for an instance
func (ut *UpdateThrottler) Reset(name types.NamespacedName) {
	ut.mutex.Lock()
	defer ut.mutex.Unlock()

	delete(ut.states, name)
}

// GetLastUpdateTime returns the last update time for debugging/testing
func (ut *UpdateThrottler) GetLastUpdateTime(name types.NamespacedName) (lastUpdate time.Time, exists bool) {
	ut.mutex.RLock()
	defer ut.mutex.RUnlock()

	if state, ok := ut.states[name]; ok {
		return state.lastUpdateTime, true
	}
	return time.Time{}, false
}

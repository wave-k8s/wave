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

	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// DefaultUpdateRate is the default rate of updates per second (1.0 = 1 update per second)
	DefaultUpdateRate = 1.0
	// DefaultUpdateBurst is the default burst size (allows 100 immediate updates)
	DefaultUpdateBurst = 100
)

// UpdateThrottler manages global rate-limited updates using token bucket algorithm
// All deployments, statefulsets, and daemonsets share the same rate limiter
type UpdateThrottler struct {
	limiter *rate.Limiter
	rate    rate.Limit
	burst   int
}

// NewUpdateThrottler creates a new UpdateThrottler with the specified rate and burst
// rate: number of updates per second globally (default 1.0 = 1 update per second)
// burst: maximum number of updates that can happen in quick succession (default 100)
// If rate <= 0, uses DefaultUpdateRate (1.0). If rate is Inf, throttling is disabled.
// If burst <= 0, uses DefaultUpdateBurst (100)
func NewUpdateThrottler(r rate.Limit, burst int) *UpdateThrottler {
	if r <= 0 && !math.IsInf(float64(r), 1) {
		r = DefaultUpdateRate
	}
	if burst <= 0 {
		burst = DefaultUpdateBurst
	}
	return &UpdateThrottler{
		limiter: rate.NewLimiter(r, burst),
		rate:    r,
		burst:   burst,
	}
}

// Wait blocks until the global rate limiter allows an update
// This stalls the operator pipeline to enforce rate limiting across all instances
func (ut *UpdateThrottler) Wait(ctx context.Context, name types.NamespacedName) error {
	// If rate limiting is disabled (infinite rate), return immediately
	if math.IsInf(float64(ut.rate), 1) {
		return nil
	}

	// Block until global rate limiter allows the operation
	return ut.limiter.Wait(ctx)
}

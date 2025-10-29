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

package controller

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Config holds configuration options for controllers
type Config struct {
	MinUpdateInterval time.Duration
}

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, Config) error

// AddToManager adds all Controllers to the Manager with the given configuration
func AddToManager(m manager.Manager, cfg Config) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, cfg); err != nil {
			return err
		}
	}
	return nil
}

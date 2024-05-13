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

package daemonset

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/wave-k8s/wave/pkg/core"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
)

var _ = Describe("DaemonSet controller Suite", func() {
	core.ControllerTestSuite(
		&t, &cfg, &m,
		&requestsStart, &requests,
		func() *appsv1.DaemonSet {
			return utils.ExampleDaemonSet.DeepCopy()
		},
	)
})

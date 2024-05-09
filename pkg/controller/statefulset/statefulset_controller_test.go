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

package statefulset

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wave-k8s/wave/pkg/core"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("StatefulSet controller Suite", func() {
	core.ControllerTestSuite(
		&t, &cfg,
		func() *appsv1.StatefulSet {
			return utils.ExampleStatefulSet.DeepCopy()
		},
		func(mgr manager.Manager) (context.CancelFunc, chan reconcile.Request, chan reconcile.Request) {
			var recFn reconcile.Reconciler
			r := newReconciler(mgr)
			recFn, requestsStart, requests := SetupTestReconcile(r)
			Expect(add(mgr, recFn, r.handler)).NotTo(HaveOccurred())

			// register mutating pod webhook
			err := AddStatefulSetWebhook(mgr)
			Expect(err).ToNot(HaveOccurred())

			testCtx, testCancel = context.WithCancel(context.Background())
			go Run(testCtx, mgr)
			return testCancel, requestsStart, requests
		},
	)
})

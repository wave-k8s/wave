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
	"log"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/wave-k8s/wave/pkg/apis"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var cfg *rest.Config

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wave Controller Suite")
}

var t *envtest.Environment

var testCtx, testCancel = context.WithCancel(context.Background())

var _ = BeforeSuite(func() {
	failurePolicy := admissionv1.Ignore
	sideEffects := admissionv1.SideEffectClassNone
	webhookPath := "/mutate-apps-v1-statefulset"
	webhookInstallOptions := envtest.WebhookInstallOptions{
		MutatingWebhooks: []*admissionv1.MutatingWebhookConfiguration{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "statefulset-operator",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "MutatingWebhookConfiguration",
					APIVersion: "admissionregistration.k8s.io/v1",
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name:                    "statefulsets.wave.pusher.com",
						AdmissionReviewVersions: []string{"v1"},
						FailurePolicy:           &failurePolicy,
						ClientConfig: admissionv1.WebhookClientConfig{
							Service: &admissionv1.ServiceReference{
								Path: &webhookPath,
							},
						},
						Rules: []admissionv1.RuleWithOperations{
							{
								Operations: []admissionv1.OperationType{
									admissionv1.Create,
									admissionv1.Update,
								},
								Rule: admissionv1.Rule{
									APIGroups:   []string{"apps"},
									APIVersions: []string{"v1"},
									Resources:   []string{"statefulsets"},
								},
							},
						},
						SideEffects: &sideEffects,
					},
				},
			},
		},
	}
	t = &envtest.Environment{
		WebhookInstallOptions: webhookInstallOptions,
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crds")},
	}
	apis.AddToScheme(scheme.Scheme)

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	var err error
	if cfg, err = t.Start(); err != nil {
		log.Fatal(err)
	}

})

var _ = AfterSuite(func() {
	t.Stop()
})

// SetupTestReconcile returns a reconcile.Reconcile implementation that delegates to inner and
// writes the request to requests after Reconcile is finished.
func SetupTestReconcile(inner reconcile.Reconciler) (reconcile.Reconciler, chan reconcile.Request, chan reconcile.Request) {
	requestsStart := make(chan reconcile.Request)
	requests := make(chan reconcile.Request)
	fn := reconcile.Func(func(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
		requestsStart <- req
		result, err := inner.Reconcile(ctx, req)
		requests <- req
		return result, err
	})
	return fn, requestsStart, requests
}

// Run runs the webhook server.
func Run(ctx context.Context, k8sManager ctrl.Manager) error {
	if err := k8sManager.Start(ctx); err != nil {
		return err
	}
	return nil
}

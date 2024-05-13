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
	"context"
	"log"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wave-k8s/wave/pkg/core"
	"github.com/wave-k8s/wave/test/utils"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

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

var requestsStart <-chan reconcile.Request
var requests <-chan reconcile.Request

var m utils.Matcher

var _ = BeforeSuite(func() {
	failurePolicy := admissionv1.Ignore
	sideEffects := admissionv1.SideEffectClassNone
	webhookPath := "/mutate-apps-v1-daemonset"
	webhookInstallOptions := envtest.WebhookInstallOptions{
		MutatingWebhooks: []*admissionv1.MutatingWebhookConfiguration{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "daemonset-operator",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "MutatingWebhookConfiguration",
					APIVersion: "admissionregistration.k8s.io/v1",
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name:                    "daemonsets.wave.pusher.com",
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
									Resources:   []string{"daemonsets"},
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

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	var err error
	if cfg, err = t.Start(); err != nil {
		log.Fatal(err)
	}

	// Reset the Prometheus Registry before each test to avoid errors
	metrics.Registry = prometheus.NewRegistry()

	mgr, err := manager.New(cfg, manager.Options{
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    (*t).WebhookInstallOptions.LocalServingHost,
			Port:    (*t).WebhookInstallOptions.LocalServingPort,
			CertDir: (*t).WebhookInstallOptions.LocalServingCertDir,
		}),
	})
	Expect(err).NotTo(HaveOccurred())

	c, cerr := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(cerr).NotTo(HaveOccurred())
	m = utils.Matcher{Client: c}

	var recFn reconcile.Reconciler
	r := newReconciler(mgr)
	recFn, requestsStart, requests = core.SetupControllerTestReconcile(r)
	Expect(add(mgr, recFn, r.handler)).NotTo(HaveOccurred())

	// register mutating pod webhook
	err = AddDaemonSetWebhook(mgr)
	Expect(err).ToNot(HaveOccurred())

	testCtx, testCancel = context.WithCancel(context.Background())
	go core.Run(testCtx, mgr)
})

var _ = AfterSuite(func() {
	testCancel()
	t.Stop()
})

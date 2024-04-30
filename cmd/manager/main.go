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

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/wave-k8s/wave/pkg/apis"
	"github.com/wave-k8s/wave/pkg/controller"
	"github.com/wave-k8s/wave/pkg/controller/daemonset"
	"github.com/wave-k8s/wave/pkg/controller/deployment"
	"github.com/wave-k8s/wave/pkg/controller/statefulset"
	k8swebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	leaderElection          = flag.Bool("leader-election", false, "Should the controller use leader election")
	leaderElectionID        = flag.String("leader-election-id", "", "Name of the configmap used by the leader election system")
	leaderElectionNamespace = flag.String("leader-election-namespace", "", "Namespace for the configmap used by the leader election system")
	syncPeriod              = flag.Duration("sync-period", 5*time.Minute, "Reconcile sync period")
	showVersion             = flag.Bool("version", false, "Show version and exit")
	enableWebhooks          = flag.Bool("enable-webhooks", false, "Enable webhooks")
	setupLog                = ctrl.Log.WithName("setup")
)

func main() {
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if *showVersion {
		fmt.Printf("wave %s (built with %s)\n", VERSION, runtime.Version())
		return
	}

	// Get a config to talk to the apiserver
	setupLog.Info("setting up client for manager")
	cfg, err := config.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to set up client config")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	setupLog.Info("setting up manager")
	var webhookServer k8swebhook.Server
	if *enableWebhooks {
		webhookServer = k8swebhook.NewServer(k8swebhook.Options{
			Port: 9443,
		})
	}
	mgr, err := manager.New(cfg, manager.Options{
		WebhookServer:           webhookServer,
		LeaderElection:          *leaderElection,
		LeaderElectionID:        *leaderElectionID,
		LeaderElectionNamespace: *leaderElectionNamespace,
		Cache: cache.Options{
			SyncPeriod: syncPeriod,
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	setupLog.Info("Registering Components.")

	// Setup Scheme for all resources
	setupLog.Info("setting up scheme")
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		setupLog.Error(err, "unable add APIs to scheme")
		os.Exit(1)
	}

	// Setup all Controllers
	setupLog.Info("Setting up controller")
	if err := controller.AddToManager(mgr); err != nil {
		setupLog.Error(err, "unable to register controllers to the manager")
		os.Exit(1)
	}
	if *enableWebhooks {
		if err := deployment.AddDeploymentWebhook(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Deployment")
			os.Exit(1)
		}

		if err := statefulset.AddStatefulSetWebhook(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "StatefulSet")
			os.Exit(1)
		}

		if err := daemonset.AddDaemonSetWebhook(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "DaemonSet")
			os.Exit(1)
		}
	}

	// Start the Cmd
	setupLog.Info("Starting the Cmd.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "unable to run the manager")
		os.Exit(1)
	}
}

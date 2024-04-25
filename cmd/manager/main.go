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
	goflag "flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/go-logr/glogr"
	flag "github.com/spf13/pflag"
	"github.com/wave-k8s/wave/pkg/apis"
	"github.com/wave-k8s/wave/pkg/controller"
	"github.com/wave-k8s/wave/pkg/webhook"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	leaderElection          = flag.Bool("leader-election", false, "Should the controller use leader election")
	leaderElectionID        = flag.String("leader-election-id", "", "Name of the configmap used by the leader election system")
	leaderElectionNamespace = flag.String("leader-election-namespace", "", "Namespace for the configmap used by the leader election system")
	syncPeriod              = flag.Duration("sync-period", 5*time.Minute, "Reconcile sync period")
	showVersion             = flag.Bool("version", false, "Show version and exit")
)

func main() {
	// Setup flags
	err := goflag.Lookup("logtostderr").Value.Set("true")
	if err != nil {
		fmt.Printf("unable to set logtostderr %v", err)
		os.Exit(1)
	}
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	if *showVersion {
		fmt.Printf("wave %s (built with %s)\n", VERSION, runtime.Version())
		return
	}

	logf.SetLogger(glogr.New())
	log := logf.Log.WithName("entrypoint")

	// Get a config to talk to the apiserver
	log.Info("setting up client for manager")
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "unable to set up client config")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	log.Info("setting up manager")
	mgr, err := manager.New(cfg, manager.Options{
		LeaderElection:          *leaderElection,
		LeaderElectionID:        *leaderElectionID,
		LeaderElectionNamespace: *leaderElectionNamespace,
		Cache: cache.Options{
			SyncPeriod: syncPeriod,
		},
	})
	if err != nil {
		log.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	log.Info("setting up scheme")
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "unable add APIs to scheme")
		os.Exit(1)
	}

	// Setup all Controllers
	log.Info("Setting up controller")
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "unable to register controllers to the manager")
		os.Exit(1)
	}

	log.Info("setting up webhooks")
	if err := webhook.AddToManager(mgr); err != nil {
		log.Error(err, "unable to register webhooks to the manager")
		os.Exit(1)
	}

	// Start the Cmd
	log.Info("Starting the Cmd.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "unable to run the manager")
		os.Exit(1)
	}
}

/*
Copyright 2023.

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
	"embed"
	"flag"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/cmd/options"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/secretresolver"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/openmcp-project/controller-utils/pkg/init/crds"
	"github.com/openmcp-project/controller-utils/pkg/init/webhooks"

	corev1beta1 "github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/internal/controller"
	"github.com/openmcp-project/control-plane-operator/internal/schemes"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/kubeconfiggen"
	// +kubebuilder:scaffold:imports
)

var (
	setupLog = ctrl.Log.WithName("setup")

	//go:embed embedded/crds
	crdFiles embed.FS

	crdFlags      = crds.BindFlags(flag.CommandLine)
	webhooksFlags = webhooks.BindFlags(flag.CommandLine)
)

func runInit(setupClient client.Client) {
	initContext := context.Background()

	if webhooksFlags.Install {
		// Generate webhook certificate
		if err := webhooks.GenerateCertificate(initContext, setupClient, webhooksFlags.CertOptions...); err != nil {
			setupLog.Error(err, "unable to generate webhook certificates")
			os.Exit(1)
		}

		// Install webhooks
		err := webhooks.Install(
			initContext,
			setupClient,
			schemes.Local,
			[]client.Object{
				&corev1beta1.ControlPlane{},
			},
		)
		if err != nil {
			setupLog.Error(err, "unable to configure webhooks")
			os.Exit(1)
		}
	}

	if crdFlags.Install {
		// Install CRDs
		if err := crds.Install(initContext, setupClient, crdFiles); err != nil {
			setupLog.Error(err, "unable to install Custom Resource Definitions")
			os.Exit(1)
		}
	}
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)

	var syncPeriod string
	flag.StringVar(&syncPeriod, "sync-period", "1m", "The period at which the controller will sync the resources.")

	// component flags
	var webhookMiddlewareName string
	flag.StringVar(&webhookMiddlewareName, "webhook-middleware-name", "",
		"Name of the middleware that should be used for the webhooks.")

	var webhookMiddlewareNamespace string
	flag.StringVar(&webhookMiddlewareNamespace, "webhook-middleware-namespace", "",
		"Namespace of the middleware that should be used for the webhooks.")

	options.AddOptions()

	// skip os.Args[1] which is the command (start or init)
	if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
		setupLog.Error(err, "failed to parse flags")
		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupContext := context.Background()

	setupClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: schemes.Local})
	if err != nil {
		setupLog.Error(err, "unable to create client")
		os.Exit(1)
	}

	if os.Args[1] == "init" {
		runInit(setupClient)
		return
	}

	fluxSecretResolver := secretresolver.NewFluxSecretResolver(setupClient)
	err = fluxSecretResolver.Start(setupContext)
	if err != nil {
		setupLog.Error(err, "failed to start FluxSecretResolver")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 schemes.Local,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "c627d721.core.orchestrate.cloud.sap",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	reconcilePeriod, errPeriod := time.ParseDuration(syncPeriod)
	if errPeriod != nil {
		reconcilePeriod = 1 * time.Minute
	}
	setupLog.Info("sync period set to", "syncPeriod", reconcilePeriod)

	if err = (&controller.ControlPlaneReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		Kubeconfiggen:      &kubeconfiggen.Default{},
		FluxSecretResolver: fluxSecretResolver,
		WebhookMiddleware: types.NamespacedName{
			Namespace: webhookMiddlewareNamespace,
			Name:      webhookMiddlewareName,
		},
		ReconcilePeriod:     reconcilePeriod,
		RemoteConfigBuilder: controller.NewRemoteConfigBuilder(),
		Recorder:            mgr.GetEventRecorderFor("controlplane-controller"),
		EmbeddedCRDs:        crdFiles,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ControlPlane")
		os.Exit(1)
	}
	if err = (&controller.SecretReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err = (&controller.ReleaseChannelReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Releasechannel")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

package main

import (
	"crypto/tls"
	"flag"

	runtime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/everoute/trafficredirect/api/trafficredirect/v1alpha1"
	"github.com/everoute/trafficredirect/pkg/config"
	"github.com/everoute/trafficredirect/pkg/constants"
)

var Scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(v1alpha1.AddToScheme(Scheme))
}

func main() {
	klog.InitFlags(nil)
	config.InitFlags(nil)
	flag.Parse()

	ctrl.SetLogger(klog.Background())
	stopCtx := ctrl.SetupSignalHandler()

	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                  Scheme,
		MetricsBindAddress:      config.Config.MetricsAddr,
		HealthProbeBindAddress:  config.Config.HealthAddr,
		LeaderElection:          config.Config.EnableLeaderElection,
		LeaderElectionNamespace: config.Config.LeaderElectionNamespace,
		LeaderElectionID:        config.Config.LeaderElectionName,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    config.Config.WebhookPort,
			CertDir: constants.WebhookCertPath,
			TLSOpts: []func(*tls.Config){
				func(conf *tls.Config) { conf.MinVersion = tls.VersionTLS13 },
			},
		}),
	})
	if err != nil {
		klog.Fatalf("unable to new controller manager: %s", err)
	}
	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		klog.Fatalf("Failed to add healthz ping checker")
	}

	if err := (&v1alpha1.Rule{}).SetupWebhookWithManager(mgr); err != nil {
		klog.Fatalf("unable to registry webhook for rule: %s", err)
	}

	klog.Info("Start controller manager")
	if err := mgr.Start(stopCtx); err != nil {
		klog.Fatalf("Failed to start controller manager: %s", err)
	}
}

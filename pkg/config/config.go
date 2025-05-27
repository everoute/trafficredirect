package config

import (
	"flag"
)

var Config T

type T struct {
	MetricsAddr string
	HealthAddr  string
	WebhookPort int

	EnableLeaderElection    bool
	LeaderElectionNamespace string
	LeaderElectionName      string
}

func InitFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	flagset.StringVar(&Config.MetricsAddr, "metrics-addr", ":9605", "the metrics address")
	flagset.StringVar(&Config.HealthAddr, "health-addr", ":9601", "the health address")
	flagset.IntVar(&Config.WebhookPort, "webhook-port", 9603, "the webhook port")
	flagset.BoolVar(&Config.EnableLeaderElection, "enable-leader-election", true, "enable leader election or not")
	flagset.StringVar(&Config.LeaderElectionNamespace, "leader-election-namespace", "kube-system", "the namespace of leader election lease")
	flagset.StringVar(&Config.LeaderElectionName, "leader-election-name", "tr-controller.leader-election.everoute.io", "the name of leader election lease")
}

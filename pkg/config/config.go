package config

import (
	"flag"
	"os"
	"time"
)

var Config T

type T struct {
	MetricsAddr string
	HealthAddr  string
	WebhookPort int

	EnableLeaderElection    bool
	LeaderElectionNamespace string
	LeaderElectionName      string

	Tower TowerOpts
}

type TowerOpts struct {
	AllowInsecure bool
	Addr          string
	Username      string
	Password      string
	Source        string
	APIUsername   string
	APIPassword   string
	ResyncPeriod  time.Duration
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

	flagset.BoolVar(&Config.Tower.AllowInsecure, "tower-allow-insecure", true, "tower allow-insecure for authenticate")
	flagset.StringVar(&Config.Tower.Addr, "tower-addr", "", "tower api address host:port")
	flagset.StringVar(&Config.Tower.Username, "tower-username", os.Getenv("TOWER_USERNAME"), "tower username for authenticate")
	flagset.StringVar(&Config.Tower.Password, "tower-password", os.Getenv("TOWER_PASSWORD"), "tower user password for authenticate")
	flagset.StringVar(&Config.Tower.Source, "tower-source", os.Getenv("TOWER_USERSOURCE"), "tower user source for authenticate")
	flagset.StringVar(&Config.Tower.APIUsername, "tower-api-username", os.Getenv("TOWER_API_USERNAME"), "tower username for api")
	flagset.StringVar(&Config.Tower.APIPassword, "tower-api-password", os.Getenv("TOWER_API_PASSWORD"), "tower password for api")
	flagset.DurationVar(&Config.Tower.ResyncPeriod, "tower-resync-period", 10*time.Hour, "tower resource resync period")
}

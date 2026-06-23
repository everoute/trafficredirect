package config

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

var Config T

type T struct {
	MetricsAddr string
	HealthAddr  string
	WebhookHost string
	WebhookPort int

	EnableLeaderElection    bool
	LeaderElectionNamespace string
	LeaderElectionName      string

	Tower TowerOpts
}

type TowerOpts struct {
	AllowInsecure      bool
	Addr               string
	Scheme             string
	Username           string
	Password           string
	Source             string
	APIUsername        string
	APIPassword        string
	CrcInterval        time.Duration
	CrcCatchUpInterval time.Duration
	CrcLimit           int32
}

type int32Value struct {
	p *int32
}

func newInt32Value(value int32, p *int32) *int32Value {
	*p = value
	return &int32Value{p: p}
}

func (v *int32Value) Set(s string) error {
	i, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return err
	}
	if i < math.MinInt32 || i > math.MaxInt32 {
		return fmt.Errorf("value %d overflows int32", i)
	}
	*v.p = int32(i)
	return nil
}

func (v *int32Value) String() string {
	if v == nil || v.p == nil {
		return ""
	}
	return strconv.FormatInt(int64(*v.p), 10)
}

func (v *int32Value) Get() any {
	if v == nil || v.p == nil {
		return int32(0)
	}
	return *v.p
}

func InitFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	flagset.StringVar(&Config.MetricsAddr, "metrics-addr", ":9605", "the metrics address")
	flagset.StringVar(&Config.HealthAddr, "health-addr", ":9601", "the health address")
	flagset.StringVar(&Config.WebhookHost, "webhook-host", "127.0.0.1", "the webhook host")
	flagset.IntVar(&Config.WebhookPort, "webhook-port", 9603, "the webhook port")
	flagset.BoolVar(&Config.EnableLeaderElection, "enable-leader-election", true, "enable leader election or not")
	flagset.StringVar(&Config.LeaderElectionNamespace, "leader-election-namespace", "kube-system", "the namespace of leader election lease")
	flagset.StringVar(&Config.LeaderElectionName, "leader-election-name", "tr-controller.leader-election.everoute.io", "the name of leader election lease")

	flagset.BoolVar(&Config.Tower.AllowInsecure, "tower-allow-insecure", true, "tower allow-insecure for authenticate")
	flagset.StringVar(&Config.Tower.Addr, "tower-addr", "", "tower api address host:port")
	flagset.StringVar(&Config.Tower.Scheme, "tower-scheme", "https", "tower api scheme")
	flagset.StringVar(&Config.Tower.Username, "tower-username", os.Getenv("TOWER_USERNAME"), "tower username for authenticate")
	flagset.StringVar(&Config.Tower.Password, "tower-password", os.Getenv("TOWER_PASSWORD"), "tower user password for authenticate")
	flagset.StringVar(&Config.Tower.Source, "tower-source", os.Getenv("TOWER_USERSOURCE"), "tower user source for authenticate")
	flagset.StringVar(&Config.Tower.APIUsername, "tower-api-username", os.Getenv("TOWER_API_USERNAME"), "tower username for api")
	flagset.StringVar(&Config.Tower.APIPassword, "tower-api-password", os.Getenv("TOWER_API_PASSWORD"), "tower password for api")
	flagset.DurationVar(&Config.Tower.CrcInterval, "tower-crc-interval", 10*time.Second, "tower resource change event watch polling interval")
	flagset.DurationVar(&Config.Tower.CrcCatchUpInterval, "tower-crc-catch-up-interval", 3*time.Second, "tower resource change event watch catch up polling interval")
	flagset.Var(newInt32Value(500, &Config.Tower.CrcLimit), "tower-crc-limit", "tower resource change event watch polling limit")
}

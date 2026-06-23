package config_test

import (
	"flag"
	"testing"

	"github.com/everoute/trafficredirect/pkg/config"
)

func TestInitFlagsDefaultScheme(t *testing.T) {
	config.Config = config.T{}
	flagset := flag.NewFlagSet("test", flag.ContinueOnError)
	config.InitFlags(flagset)

	if got := config.Config.Tower.Scheme; got != "https" {
		t.Fatalf("Tower.Scheme = %q, want %q", got, "https")
	}
}

func TestInitFlagsOverrideScheme(t *testing.T) {
	config.Config = config.T{}
	flagset := flag.NewFlagSet("test", flag.ContinueOnError)
	config.InitFlags(flagset)

	if err := flagset.Parse([]string{"--tower-scheme=http"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	if got := config.Config.Tower.Scheme; got != "http" {
		t.Fatalf("Tower.Scheme = %q, want %q", got, "http")
	}
}

func TestInitFlagsDefaultCRCLimit(t *testing.T) {
	config.Config = config.T{}
	flagset := flag.NewFlagSet("test", flag.ContinueOnError)
	config.InitFlags(flagset)

	if got := config.Config.Tower.CrcLimit; got != 500 {
		t.Fatalf("Tower.CrcLimit = %d, want %d", got, 500)
	}
}

func TestInitFlagsOverrideCRCLimit(t *testing.T) {
	config.Config = config.T{}
	flagset := flag.NewFlagSet("test", flag.ContinueOnError)
	config.InitFlags(flagset)

	if err := flagset.Parse([]string{"--tower-crc-limit=2147483647"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	if got := config.Config.Tower.CrcLimit; got != 2147483647 {
		t.Fatalf("Tower.CrcLimit = %d, want %d", got, 2147483647)
	}
}

func TestInitFlagsRejectOverflowCRCLimit(t *testing.T) {
	config.Config = config.T{}
	flagset := flag.NewFlagSet("test", flag.ContinueOnError)
	config.InitFlags(flagset)

	if err := flagset.Parse([]string{"--tower-crc-limit=2147483648"}); err == nil {
		t.Fatal("parse flags error = nil, want non-nil")
	}
}

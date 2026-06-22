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

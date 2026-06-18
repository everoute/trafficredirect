package config_test

import (
	"flag"
	"testing"

	"github.com/everoute/trafficredirect/pkg/config"
)

func TestTowerOptsAddrValues(t *testing.T) {
	tests := []struct {
		name      string
		opts      config.TowerOpts
		wantHTTPS string
		wantHTTP  string
	}{
		{
			name:      "uses explicit https and http addresses",
			opts:      config.TowerOpts{HTTPSAddr: "127.0.0.1:21003", HTTPAddr: "127.0.0.1:21002"},
			wantHTTPS: "127.0.0.1:21003",
			wantHTTP:  "127.0.0.1:21002",
		},
		{
			name:      "falls back http address to https address",
			opts:      config.TowerOpts{HTTPSAddr: "127.0.0.1:21003"},
			wantHTTPS: "127.0.0.1:21003",
			wantHTTP:  "127.0.0.1:21003",
		},
		{
			name:      "keeps legacy tower address as https fallback",
			opts:      config.TowerOpts{Addr: "tower.example:443"},
			wantHTTPS: "tower.example:443",
			wantHTTP:  "tower.example:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.HTTPSAddress(); got != tt.wantHTTPS {
				t.Fatalf("HTTPSAddress() = %q, want %q", got, tt.wantHTTPS)
			}
			if got := tt.opts.HTTPAddress(); got != tt.wantHTTP {
				t.Fatalf("HTTPAddress() = %q, want %q", got, tt.wantHTTP)
			}
		})
	}
}

func TestInitFlagsTowerAddrs(t *testing.T) {
	config.Config = config.T{}
	flagset := flag.NewFlagSet("test", flag.ContinueOnError)
	config.InitFlags(flagset)

	err := flagset.Parse([]string{
		"--tower-addr=tower.example:443",
		"--tower-https-addr=127.0.0.1:21003",
		"--tower-http-addr=127.0.0.1:21002",
	})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	if got := config.Config.Tower.HTTPSAddress(); got != "127.0.0.1:21003" {
		t.Fatalf("HTTPSAddress() = %q, want %q", got, "127.0.0.1:21003")
	}
	if got := config.Config.Tower.HTTPAddress(); got != "127.0.0.1:21002" {
		t.Fatalf("HTTPAddress() = %q, want %q", got, "127.0.0.1:21002")
	}
}

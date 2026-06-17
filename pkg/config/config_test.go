package config

import (
	"flag"
	"testing"
)

func TestInitFlagsTowerScheme(t *testing.T) {
	oldConfig := Config
	defer func() { Config = oldConfig }()

	Config = T{}
	flagset := flag.NewFlagSet("test", flag.ContinueOnError)
	InitFlags(flagset)

	if err := flagset.Parse(nil); err != nil {
		t.Fatalf("parse default flags: %v", err)
	}
	if Config.Tower.Scheme != "http" {
		t.Fatalf("expected default tower scheme http, got %q", Config.Tower.Scheme)
	}

	Config = T{}
	flagset = flag.NewFlagSet("test", flag.ContinueOnError)
	InitFlags(flagset)
	if err := flagset.Parse([]string{"--tower-scheme=https"}); err != nil {
		t.Fatalf("parse explicit flags: %v", err)
	}
	if Config.Tower.Scheme != "https" {
		t.Fatalf("expected explicit tower scheme https, got %q", Config.Tower.Scheme)
	}
}

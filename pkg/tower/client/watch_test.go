package client

import (
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/everoute/graphc/pkg/crcwatch"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
	"github.com/smartxworks/cloudtower-go-sdk/v2/watchor"

	"github.com/everoute/trafficredirect/pkg/config"
)

type fakeResourceChangeWatcher struct{}

func (f *fakeResourceChangeWatcher) Start(_ *watchor.ResourceChangeWatchStartParams) error {
	return nil
}

func (f *fakeResourceChangeWatcher) Channel() <-chan *models.ResourceChangeEvent {
	return nil
}

func (f *fakeResourceChangeWatcher) ErrorChannel() <-chan *watchor.ErrorEvent {
	return nil
}

func (f *fakeResourceChangeWatcher) WarningChannel() <-chan *watchor.WarningEvent {
	return nil
}

func TestNewCRCWatchConfiguresSchemeAndInsecure(t *testing.T) {
	config.Config = config.T{}
	config.Config.Tower = config.TowerOpts{
		Addr:               "127.0.0.1:21003",
		Scheme:             "http",
		AllowInsecure:      true,
		CrcInterval:        10 * time.Second,
		CrcCatchUpInterval: 3 * time.Second,
		CrcLimit:           500,
	}

	var gotOptions *crcwatch.Options
	patches := gomonkey.ApplyFunc(crcwatch.NewWatchClient,
		func(_ []string, opts *crcwatch.Options) (crcwatch.ResourceChangeWatcher, error) {
			gotOptions = opts
			return &fakeResourceChangeWatcher{}, nil
		})
	defer patches.Reset()

	if _, err := NewCRCWatch(nil); err != nil {
		t.Fatalf("NewCRCWatch() error = %v", err)
	}

	if gotOptions == nil {
		t.Fatal("crcwatch options were not captured")
	}
	if gotOptions.Host != "127.0.0.1:21003" {
		t.Fatalf("Host = %q, want %q", gotOptions.Host, "127.0.0.1:21003")
	}
	if gotOptions.Scheme != "http" {
		t.Fatalf("Scheme = %q, want %q", gotOptions.Scheme, "http")
	}
	if !gotOptions.AllowInsecure {
		t.Fatal("AllowInsecure = false, want true")
	}
	if gotOptions.Limit != 500 {
		t.Fatalf("Limit = %d, want %d", gotOptions.Limit, 500)
	}
}

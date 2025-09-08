package source

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Kind provides a source of events originating inside the cluster from Watches
// difference from source.Kind, add informer to factory on cache inject

func Kind(cache cache.Cache, object client.Object) source.SyncingSource {
	// should never hang on WaitForCacheSync
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// add typed informer to informers factory before controller start
	// make sure that when controllers start, all caches are synchronized
	if _, err := cache.GetInformer(ctx, object); err != nil && !errors.IsTimeout(err) {
		klog.Fatalf("Failed to add %v informer, err: %v", object, err)
	}

	return source.Kind(cache, object)
}

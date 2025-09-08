package log

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func GetAndSetLogForCtx(ctx context.Context, kvs ...any) (context.Context, logr.Logger) {
	log := ctrl.LoggerFrom(ctx)
	log = log.WithValues(kvs...)
	ctx = ctrl.LoggerInto(ctx, log)
	return ctx, log
}

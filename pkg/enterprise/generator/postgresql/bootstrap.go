package postgresql

import (
	"context"
	"fmt"
	"time"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/scheduler"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type PostgreSQLBootstrap struct {
	mgr    manager.Manager
	client client.Client
}

func NewPostgreSQLBootstrap(client client.Client, mgr manager.Manager) *PostgreSQLBootstrap {
	return &PostgreSQLBootstrap{
		client: client,
		mgr:    mgr,
	}
}

func (b *PostgreSQLBootstrap) Start(ctx context.Context) error {
	if ok := b.mgr.GetCache().WaitForCacheSync(ctx); !ok {
		return ctx.Err()
	}

	var list genv1alpha1.GeneratorStateList
	if err := b.mgr.GetClient().List(ctx, &list); err != nil {
		return err
	}

	for _, gs := range list.Items {
		spec, err := parseSpec(gs.Spec.Resource.Raw)
		if err != nil {
			return err
		}
		if spec.Kind != "PostgreSql" {
			// not a PostgreSql spec. skipping
			continue
		}

		cleanupPolicy := spec.Spec.CleanupPolicy
		if cleanupPolicy != nil && cleanupPolicy.Type == genv1alpha1.IdleCleanupPolicy {
			connectionId := fmt.Sprintf(schedIdFmt, spec.UID)

			scheduler.Global().ScheduleInterval(connectionId, spec.Spec.CleanupPolicy.ActivityTrackingInterval.Duration, time.Minute, func(ctx context.Context, log logr.Logger) {
				err := triggerSessionSnapshot(ctx, &spec.Spec, b.client, gs.GetNamespace())
				if err != nil {
					log.Error(err, "failed to trigger session observation")
					return
				}
			})
		}
	}

	return nil
}

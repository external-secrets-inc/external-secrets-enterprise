// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package postgresql

import (
	"context"
	"fmt"
	"time"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/scheduler"
	"github.com/go-logr/logr"
	"github.com/labstack/gommon/log"
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
		if gs.Spec.GarbageCollectionDeadline == nil {
			log.Info("skipping generator state without garbage collection deadline")
			continue
		}

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
			connectionId := fmt.Sprintf(schedIdFmt, spec.UID, spec.Spec.Host, spec.Spec.Port)

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

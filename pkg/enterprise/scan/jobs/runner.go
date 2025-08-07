// Copyright External Secrets Inc. 2025
// All Rights Reserved
package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	store "github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type JobRunner struct {
	client.Client
	logr.Logger
	Constraints *v1alpha1.JobConstraints
	mgr         *store.Manager
	Namespace   string
	memset      *MemorySet
}

func NewJobRunner(client client.Client, logger logr.Logger, namespace string, constraints *v1alpha1.JobConstraints) *JobRunner {
	mgr := store.NewManager(client, "", false)
	return &JobRunner{
		Client:      client,
		Logger:      logger,
		Constraints: constraints,
		Namespace:   namespace,
		mgr:         mgr,
		memset:      NewMemorySet(),
	}
}

func (j *JobRunner) Close(ctx context.Context) error {
	return j.mgr.Close(ctx)
}

func (j *JobRunner) Run(ctx context.Context) ([]v1alpha1.Finding, []esv1.SecretStore, error) {
	// List Secret Stores
	// TODO - apply constraints
	j.Logger.V(1).Info("Listing Secret Stores")
	usedStores := make([]esv1.SecretStore, 0)
	stores := &esv1.SecretStoreList{}
	if err := j.Client.List(ctx, stores, client.InNamespace(j.Namespace)); err != nil {
		return nil, nil, err
	}
	for i := range stores.Items {
		store := stores.Items[i]
		usedStores = append(usedStores, store)
		client, err := j.mgr.GetFromStore(ctx, &store, j.Namespace)
		if err != nil {
			return nil, nil, err
		}
		ref := esv1.ExternalSecretFind{
			Name: &esv1.FindName{
				RegExp: ".*",
			},
		}
		// For Each Secret Store, Get All Secrets;
		j.Logger.V(1).Info("Getting Secrets for store", "store", store.GetName())
		secrets, err := client.GetAllSecrets(ctx, ref)
		if err != nil {
			j.Logger.Error(err, "failed to get secrets from store", "store", store.GetName())
			continue
		}
		// For Each Secret, Calculate Duplicates

		j.Logger.V(1).Info("Calculating duplicates for store", "store", store.GetName())
		for key, value := range secrets {
			valueAsMap := map[string]interface{}{}
			if err := json.Unmarshal(value, &valueAsMap); err == nil {
				for k, v := range valueAsMap {
					switch v := v.(type) {
					case []byte:
						j.memset.Add(newStoreInRef(store.GetName(), key, k), v)
					case string:
						j.memset.Add(newStoreInRef(store.GetName(), key, k), []byte(v))
					default:
						return nil, nil, fmt.Errorf("no conversion for value of type %T", v)
					}
				}
			} else {
				// For Each duplicate found, create a Finding bound to that hash;
				j.memset.Add(newStoreInRef(store.GetName(), key, ""), value)
			}
		}
	}
	// Check All duplicates on all created targets
	j.Logger.V(1).Info("Getting Virtual Machine Targets")
	targets := &tgtv1alpha1.VirtualMachineList{}
	if err := j.Client.List(ctx, targets, client.InNamespace(j.Namespace)); err != nil {
		return nil, nil, err
	}
	for _, target := range targets.Items {
		j.Logger.V(1).Info("Scanning target", "target", target.GetName())
		prov, ok := tgtv1alpha1.GetTargetByName(target.GroupVersionKind().Kind)
		if !ok {
			return nil, nil, fmt.Errorf("target %q not found", target.GroupVersionKind().Kind)
		}
		client, err := prov.NewClient(ctx, j.Client, &target)
		if err != nil {
			return nil, nil, err
		}
		regexMap := j.memset.Regexes()
		for key, regexes := range regexMap {
			// TODO Fix Threshold
			locations, err := client.Scan(ctx, regexes, j.memset.GetThreshold())
			if err != nil {
				return nil, nil, err
			}
			for _, location := range locations {
				j.memset.AddByRegex(key, location)
			}
		}
	}
	j.Logger.V(1).Info("Run Complete")
	return j.memset.GetDuplicates(), usedStores, nil
}

func newStoreInRef(store, key, property string) tgtv1alpha1.SecretInStoreRef {
	return tgtv1alpha1.SecretInStoreRef{
		Name:       store,
		Kind:       "SecretStore",
		APIVersion: "external-secrets.io/v1",
		RemoteRef: tgtv1alpha1.RemoteRef{
			Key:      key,
			Property: property,
		},
	}
}

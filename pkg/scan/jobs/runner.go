// Copyright External Secrets Inc. 2025
// All Rights Reserved
package job

import (
	"context"
	"encoding/json"
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
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

func (j *JobRunner) Run(ctx context.Context) ([]v1alpha1.Finding, error) {
	// List Secret Stores
	// TODO - apply constraints
	stores := &esv1.SecretStoreList{}
	if err := j.Client.List(ctx, stores); err != nil {
		return nil, err
	}
	for _, store := range stores.Items {
		client, err := j.mgr.GetFromStore(ctx, &store, j.Namespace)
		if err != nil {
			return nil, err
		}
		ref := esv1.ExternalSecretFind{
			Name: &esv1.FindName{
				RegExp: ".*",
			},
		}
		// For Each Secret Store, Get All Secrets;

		secrets, err := client.GetAllSecrets(ctx, ref)
		if err != nil {
			return nil, err
		}
		// For Each Secret, Calculate Duplicates

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
						return nil, fmt.Errorf("no conversion for value of type %T", v)
					}
				}
			}

			// For Each duplicate found, create a Finding bound to that hash;
			j.memset.Add(newStoreInRef(store.GetName(), key, ""), value)
		}
	}
	// Check All duplicates on all created targets
	targets := &tgtv1alpha1.VirtualMachineList{}
	if err := j.Client.List(ctx, targets); err != nil {
		return nil, err
	}
	for _, target := range targets.Items {
		prov, ok := tgtv1alpha1.GetTargetByName(target.GroupVersionKind().Kind)
		if !ok {
			return nil, fmt.Errorf("target %q not found", target.GroupVersionKind().Kind)
		}
		client, err := prov.NewClient(ctx, j.Client, &target)
		if err != nil {
			return nil, err
		}
		regexMap := j.memset.Regexes()
		for key, regexes := range regexMap {
			// TODO Fix Threshold
			locations, err := client.Scan(ctx, regexes, j.memset.GetThreshold())
			if err != nil {
				return nil, err
			}
			for _, location := range locations {
				j.memset.AddByRegex(key, location)
			}
		}
	}

	return j.memset.GetDuplicates(), nil
}

func newStoreInRef(store, key, property string) tgtv1alpha1.SecretInStoreRef {
	return tgtv1alpha1.SecretInStoreRef{
		Name:       store,
		Kind:       "SecretStore",
		APIVersion: "externalsecrets.io/v1",
		RemoteRef: tgtv1alpha1.RemoteRef{
			Key:      key,
			Property: property,
		},
	}
}

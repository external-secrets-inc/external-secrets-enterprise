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
	Constraints    *v1alpha1.JobConstraints
	mgr            *store.Manager
	Namespace      string
	locationMemset *LocationMemorySet
	consumerMemset *ConsumerMemorySet
}

func NewJobRunner(client client.Client, logger logr.Logger, namespace string, constraints *v1alpha1.JobConstraints) *JobRunner {
	mgr := store.NewManager(client, "", false)
	return &JobRunner{
		Client:         client,
		Logger:         logger,
		Constraints:    constraints,
		Namespace:      namespace,
		mgr:            mgr,
		locationMemset: NewLocationMemorySet(),
		consumerMemset: NewConsumerMemorySet(),
	}
}

func (j *JobRunner) Close(ctx context.Context) error {
	return j.mgr.Close(ctx)
}

func (j *JobRunner) Run(ctx context.Context) ([]v1alpha1.Finding, []v1alpha1.Consumer, []esv1.SecretStore, error) {
	// List Secret Stores
	// TODO - apply constraints
	j.Logger.V(1).Info("Listing Secret Stores")
	usedStores := make([]esv1.SecretStore, 0)
	stores := &esv1.SecretStoreList{}
	if err := j.Client.List(ctx, stores, client.InNamespace(j.Namespace)); err != nil {
		return nil, nil, nil, err
	}

	secretValues := make(map[string]struct{}, 0)
	for i := range stores.Items {
		store := stores.Items[i]
		usedStores = append(usedStores, store)
		client, err := j.mgr.GetFromStore(ctx, &store, j.Namespace)
		if err != nil {
			j.Logger.Error(err, "failed to get store from manager")
			continue
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
						j.locationMemset.Add(newStoreInRef(store.GetName(), key, k), v)
						secretValues[string(v)] = struct{}{}
					case string:
						j.locationMemset.Add(newStoreInRef(store.GetName(), key, k), []byte(v))
						secretValues[v] = struct{}{}
					default:
						return nil, nil, nil, fmt.Errorf("no conversion for value of type %T", v)
					}
				}
			} else {
				// For Each duplicate found, create a Finding bound to that hash;
				j.locationMemset.Add(newStoreInRef(store.GetName(), key, ""), value)
				secretValues[string(value)] = struct{}{}
			}
		}
	}
	// Check All duplicates on all created targets
	j.Logger.V(1).Info("Getting Virtual Machine Targets")
	err := j.scanVirtualMachineTargets(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	j.Logger.V(1).Info("Getting Virtual Machine Targets")
	err = j.scanGithubRepositoryTargets(ctx, secretValues)
	if err != nil {
		return nil, nil, nil, err
	}

	findings := j.locationMemset.GetDuplicates()

	j.Logger.V(1).Info("Attributing Consumers across targets")
	if err := j.attributeConsumers(ctx, findings); err != nil {
		return nil, nil, nil, err
	}

	consumers := j.consumerMemset.List()

	j.Logger.V(1).Info("Run Complete")
	return findings, consumers, usedStores, nil
}

func (j JobRunner) scanVirtualMachineTargets(ctx context.Context) error {
	vmTargets := &tgtv1alpha1.VirtualMachineList{}
	if err := j.Client.List(ctx, vmTargets, client.InNamespace(j.Namespace)); err != nil {
		return err
	}
	for _, target := range vmTargets.Items {
		j.Logger.V(1).Info("Scanning target", "target", target.GetName())
		prov, ok := tgtv1alpha1.GetTargetByName(target.GroupVersionKind().Kind)
		if !ok {
			err := fmt.Errorf("target kind %q not supported", target.GetObjectKind().GroupVersionKind().Kind)
			j.Logger.Error(err, "failed to create new client for target", "target", target.GetName())
			continue
		}
		client, err := prov.NewClient(ctx, j.Client, &target)
		if err != nil {
			j.Logger.Error(err, "failed create new client for target", "target", target.GetName())
			continue
		}
		regexMap := j.locationMemset.Regexes()
		for key, regexes := range regexMap {
			// TODO Fix Threshold
			locations, err := client.ScanForSecrets(ctx, regexes, j.locationMemset.GetThreshold())
			if err != nil {
				j.Logger.Error(err, "failed scan target regexes", "regexes", regexes)
				continue
			}
			for _, location := range locations {
				j.locationMemset.AddByRegex(key, location)
			}
		}
	}
	return nil
}

func (j JobRunner) scanGithubRepositoryTargets(ctx context.Context, secretValues map[string]struct{}) error {
	ghTargets := &tgtv1alpha1.GithubRepositoryList{}
	if err := j.Client.List(ctx, ghTargets, client.InNamespace(j.Namespace)); err != nil {
		return err
	}
	for _, target := range ghTargets.Items {
		j.Logger.V(1).Info("Scanning target", "target", target.GetName())
		prov, ok := tgtv1alpha1.GetTargetByName(target.GroupVersionKind().Kind)
		if !ok {
			err := fmt.Errorf("target kind %q not supported", target.GetObjectKind().GroupVersionKind().Kind)
			j.Logger.Error(err, "failed to create new client for target", "target", target.GetName())
			continue
		}
		client, err := prov.NewClient(ctx, j.Client, &target)
		if err != nil {
			j.Logger.Error(err, "failed create new client for target", "target", target.GetName())
			continue
		}

		for value := range secretValues {
			locations, err := client.ScanForSecrets(ctx, []string{value}, 0)
			if err != nil {
				j.Logger.Error(err, "failed scan target value")
				continue
			}
			for _, location := range locations {
				j.locationMemset.Add(location, []byte(value))
			}
		}
	}
	return nil
}

func (j *JobRunner) attributeConsumers(ctx context.Context, findings []v1alpha1.Finding) error {
	locationsPerKindMap := make(map[string][]tgtv1alpha1.SecretInStoreRef, 0)
	for _, finding := range findings {
		for _, location := range finding.Status.Locations {
			locationsPerKindMap[location.Kind] = append(locationsPerKindMap[location.Kind], location)
		}
	}

	// VM targets
	vmTargets := &tgtv1alpha1.VirtualMachineList{}
	if err := j.Client.List(ctx, vmTargets, client.InNamespace(j.Namespace)); err != nil {
		return err
	}
	for _, target := range vmTargets.Items {
		kind := target.GroupVersionKind().Kind
		if err := j.attributeTargetConsumers(ctx, kind, target.GetName(), &target, locationsPerKindMap[kind]); err != nil {
			j.Logger.Error(err, "failed to attribute consumers on VM target", "target", target.GetName())
		}
	}

	// GitHub repo targets
	ghTargets := &tgtv1alpha1.GithubRepositoryList{}
	if err := j.Client.List(ctx, ghTargets, client.InNamespace(j.Namespace)); err != nil {
		return err
	}
	for _, target := range ghTargets.Items {
		kind := target.GroupVersionKind().Kind
		if err := j.attributeTargetConsumers(ctx, kind, target.GetName(), &target, locationsPerKindMap[kind]); err != nil {
			j.Logger.Error(err, "failed to attribute consumers on GitHub target", "target", target.GetName())
		}
	}
	return nil
}

func (j *JobRunner) attributeTargetConsumers(ctx context.Context, kind, name string, obj client.Object, locations []tgtv1alpha1.SecretInStoreRef) error {
	prov, ok := tgtv1alpha1.GetTargetByName(kind)
	if !ok {
		return fmt.Errorf("target kind %q not supported", kind)
	}
	cl, err := prov.NewClient(ctx, j.Client, obj)
	if err != nil {
		return err
	}

	for _, location := range locations {
		consumers, err := cl.ScanForConsumers(ctx, location)
		if err != nil {
			return err
		}

		targetRef := v1alpha1.TargetReference{
			Name:      name,
			Namespace: j.Namespace,
		}
		for _, f := range consumers {
			j.consumerMemset.Add(targetRef, f)
		}
	}
	return nil
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

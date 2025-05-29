// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	fedv1alpha1 "github.com/external-secrets/external-secrets/apis/federation/v1alpha1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	externalsecrets "github.com/external-secrets/external-secrets/pkg/controllers/externalsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/federation/server/auth"
	store "github.com/external-secrets/external-secrets/pkg/federation/store"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// KubernetesClaims holds specific claims related to a Kubernetes service account token.
type KubernetesIOInner struct {
	Namespace      string `json:"namespace"`
	ServiceAccount struct {
		Name string `json:"name"`
		UID  string `json:"uid"`
	} `json:"serviceaccount"`
	Pod *struct {
		Name string `json:"name"`
		UID  string `json:"uid"`
	} `json:"pod,omitempty"`
}
type KubernetesClaims struct {
	jwt.RegisteredClaims
	KubernetesIOInner `json:"kubernetes.io"`
}
type ServerHandler struct {
	reconciler             *externalsecrets.Reconciler
	mu                     sync.RWMutex
	port                   string
	generateSecretFn       func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, error)
	getSecretFn            func(ctx context.Context, storeName string, name string) ([]byte, error)
	deleteGeneratorStateFn func(ctx context.Context, namespace string, labels labels.Selector) error
}

func NewServerHandler(reconciler *externalsecrets.Reconciler, port string) *ServerHandler {
	s := &ServerHandler{
		reconciler: reconciler,
		mu:         sync.RWMutex{},
		port:       port,
	}
	s.generateSecretFn = s.generateSecret
	s.getSecretFn = s.getSecret
	s.deleteGeneratorStateFn = s.deleteGeneratorState
	return s
}

func (s *ServerHandler) SetupEcho(ctx context.Context) *echo.Echo {
	e := echo.New()
	e.Server.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	e.Use(s.authMiddleware)

	e.POST("/secretstore/:secretStoreName/secrets/:secretName", s.postSecrets)
	e.POST("/generators/:generatorNamespace/:generatorKind/:generatorName", s.generateSecrets)
	e.DELETE("/generators/:generatorNamespace/:generatorKind/:generatorName", s.revokeSelf)
	e.POST("/generators/:generatorNamespace/revoke", s.revokeCredentialsOf)
	e.Logger.Fatal(e.Start(s.port))
	return e
}

func (s *ServerHandler) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var lastErr error
		for _, auth := range auth.Registry {
			info, err := auth.Authenticate(c.Request())
			if err != nil {
				lastErr = err
				continue
			}
			c.Set("authInfo", info)
			return next(c)
		}
		return c.JSON(http.StatusUnauthorized, lastErr.Error())
	}
}

func (s *ServerHandler) generateSecrets(c echo.Context) error {
	authInfo := c.Get("authInfo").(*auth.AuthInfo)

	AuthorizationSpecs := store.Get(authInfo.Provider)
	generatorName := c.Param("generatorName")
	generatorKind := c.Param("generatorKind")
	generatorNamespace := c.Param("generatorNamespace")
	d := fedv1alpha1.AllowedGenerator{
		Name:      generatorName,
		Kind:      generatorKind,
		Namespace: generatorNamespace,
	}

	if authInfo.KubeAttributes == nil {
		return c.JSON(http.StatusBadRequest, "missing kubernetes attributes")
	}

	if authInfo.KubeAttributes.ServiceAccount == nil {
		return c.JSON(http.StatusBadRequest, "missing kubernetes service account")
	}

	owner := authInfo.KubeAttributes.ServiceAccount.Name
	if authInfo.KubeAttributes.Pod != nil {
		owner = authInfo.KubeAttributes.Pod.Name
	}

	resource := &Resource{
		Name:       generatorName,
		AuthMethod: "KubernetesServiceAccount",
		Owner:      owner,
		OwnerAttributes: map[string]string{
			"namespace":            authInfo.KubeAttributes.Namespace,
			"issuer":               authInfo.Provider,
			"serviceaccount-uid":   authInfo.KubeAttributes.ServiceAccount.UID,
			"service-account-name": authInfo.KubeAttributes.ServiceAccount.Name,
		},
	}
	if authInfo.KubeAttributes.Pod != nil {
		resource.OwnerAttributes["pod-uid"] = authInfo.KubeAttributes.Pod.UID
	}
	for _, spec := range AuthorizationSpecs {
		if contains(spec.AllowedGenerators, d) && spec.Subject.Subject == authInfo.Subject {
			secret, err := s.generateSecretFn(c.Request().Context(), generatorName, generatorKind, generatorNamespace, resource)
			if err != nil {
				return c.JSON(http.StatusBadRequest, err.Error())
			}
			return c.JSON(http.StatusOK, secret)
		}
	}
	return c.JSON(http.StatusNotFound, "Not Found")
}

func contains[T fedv1alpha1.AllowedGenerator | fedv1alpha1.AllowedGeneratorState](slice []T, item T) bool {
	switch any(item).(type) {
	case fedv1alpha1.AllowedGenerator:
		for _, v := range slice {
			sliceGen := any(v).(fedv1alpha1.AllowedGenerator)
			itemGen := any(item).(fedv1alpha1.AllowedGenerator)

			if sliceGen.Name == itemGen.Name && sliceGen.Kind == itemGen.Kind && sliceGen.Namespace == itemGen.Namespace {
				return true
			}
		}
	case fedv1alpha1.AllowedGeneratorState:
		for _, v := range slice {
			sliceState := any(v).(fedv1alpha1.AllowedGeneratorState)
			itemState := any(item).(fedv1alpha1.AllowedGeneratorState)
			if sliceState.Namespace == itemState.Namespace {
				return true
			}
		}
	}
	return false
}

func (s *ServerHandler) postSecrets(c echo.Context) error {
	authInfo := c.Get("authInfo").(*auth.AuthInfo)

	AuthorizationSpecs := store.Get(authInfo.Provider)
	storeName := c.Param("secretStoreName")
	name := c.Param("secretName")
	for _, spec := range AuthorizationSpecs {
		if slices.Contains(spec.AllowedClusterSecretStores, storeName) && spec.Subject.Subject == authInfo.Subject {
			secret, err := s.getSecretFn(c.Request().Context(), storeName, name)
			if err != nil {
				return c.JSON(http.StatusBadRequest, err.Error())
			}
			return c.JSON(http.StatusOK, string(secret))
		}
	}
	return c.JSON(http.StatusNotFound, "Not Found")
}

func (s *ServerHandler) revokeSelf(c echo.Context) error {
	authInfo := c.Get("authInfo").(*auth.AuthInfo)

	AuthorizationSpecs := store.Get(authInfo.Provider)
	generatorNamespace := c.Param("generatorNamespace")
	generatorName := c.Param("generatorName")
	generatorKind := c.Param("generatorKind")

	if authInfo.KubeAttributes == nil {
		return c.JSON(http.StatusBadRequest, "missing kubernetes attributes")
	}

	if authInfo.KubeAttributes.ServiceAccount == nil {
		return c.JSON(http.StatusBadRequest, "missing kubernetes service account")
	}

	for _, spec := range AuthorizationSpecs {
		if !contains(spec.AllowedGenerators, fedv1alpha1.AllowedGenerator{
			Name:      generatorName,
			Kind:      generatorKind,
			Namespace: generatorNamespace,
		}) || spec.Subject.Subject != authInfo.Subject {
			continue
		}
		owner := authInfo.KubeAttributes.ServiceAccount.Name
		if authInfo.KubeAttributes.Pod != nil {
			owner = authInfo.KubeAttributes.Pod.Name
		}
		labels := labels.SelectorFromSet(labels.Set{
			"federation.externalsecrets.com/owner":          owner,
			"federation.externalsecrets.com/generator":      generatorName,
			"federation.externalsecrets.com/generator-kind": generatorKind,
		})
		err := s.deleteGeneratorStateFn(c.Request().Context(), generatorNamespace, labels)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, nil)
	}
	return c.JSON(http.StatusNotFound, "Not Found")
}

type deleteRequest struct {
	Owner     string `json:"owner"`
	Namespace string `json:"namespace"`
}

func (s *ServerHandler) revokeCredentialsOf(c echo.Context) error {
	var req deleteRequest
	err := c.Bind(&req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	authInfo := c.Get("authInfo").(*auth.AuthInfo)

	AuthorizationSpecs := store.Get(authInfo.Provider)
	generatorNamespace := c.Param("generatorNamespace")
	for _, spec := range AuthorizationSpecs {
		if contains(spec.AllowedGeneratorStates, fedv1alpha1.AllowedGeneratorState{
			Namespace: generatorNamespace,
		}) && spec.Subject.Subject == authInfo.Subject {
			labels := labels.SelectorFromSet(labels.Set{
				"federation.externalsecrets.com/owner": req.Owner,
			})
			err = s.deleteGeneratorStateFn(c.Request().Context(), req.Namespace, labels)
			if err != nil {
				return c.JSON(http.StatusBadRequest, err.Error())
			}
			return c.JSON(http.StatusOK, "GeneratorState deleted")
		}
	}
	return c.JSON(http.StatusNotFound, "Not Found")
}

func (s *ServerHandler) deleteGeneratorState(ctx context.Context, namespace string, labels labels.Selector) error {
	generators := &genv1alpha1.GeneratorStateList{}
	err := s.reconciler.Client.List(ctx, generators, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels,
	})
	if err != nil {
		return err
	}
	for _, generator := range generators.Items {
		err := s.reconciler.Client.Delete(ctx, &generator)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ServerHandler) getSecret(ctx context.Context, storeName, name string) ([]byte, error) {
	storeRef := esv1.SecretStoreRef{
		Name: storeName,
		Kind: esv1.ClusterSecretStoreKind,
	}
	mgr := secretstore.NewManager(s.reconciler.Client, s.reconciler.ControllerClass, s.reconciler.EnableFloodGate)
	client, err := mgr.Get(ctx, storeRef, "", nil)
	if err != nil {
		return nil, err
	}
	ref := esv1.ExternalSecretDataRemoteRef{
		Key: name,
	}
	return client.GetSecret(ctx, ref)
}

func (s *ServerHandler) generateSecret(ctx context.Context, generatorName, generatorKind, namespace string, resource *Resource) (map[string]string, error) {
	if resource == nil {
		return nil, errors.New("resource not found")
	}
	generatorRef := esv1.GeneratorRef{
		Name:       generatorName,
		Kind:       generatorKind,
		APIVersion: "generators.external-secrets.io/v1alpha1",
	}
	generator, obj, err := resolvers.GeneratorRef(ctx, s.reconciler.Client, s.reconciler.Scheme, namespace, &generatorRef)
	if err != nil {
		return nil, err
	}
	if generator == nil {
		return nil, errors.New("generator not found")
	}
	data, stateJson, err := generator.Generate(ctx, obj, s.reconciler.Client, namespace)
	if err != nil {
		return nil, err
	}
	attributes, err := json.Marshal(resource.OwnerAttributes)
	if err != nil {
		return nil, err
	}
	if stateJson == nil {
		stateJson = &apiextensions.JSON{Raw: []byte("{}")}
	}
	generatorState := genv1alpha1.GeneratorState{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-%s-", strings.ToLower(generatorKind), strings.ToLower(generatorName), strings.ToLower(resource.Owner)),
			Namespace:    namespace,
			Labels: map[string]string{
				"federation.externalsecrets.com/owner":          resource.Owner,
				"federation.externalsecrets.com/generator":      generatorName,
				"federation.externalsecrets.com/generator-kind": generatorKind,
			},
			Annotations: map[string]string{
				"federation.externalsecrets.com/owner-attributes": string(attributes),
			},
		},
		Spec: genv1alpha1.GeneratorStateSpec{
			Resource: obj,
			State:    stateJson,
		},
	}
	var cobj client.Object
	if _, ok := resource.OwnerAttributes["pod-uid"]; ok {
		pod := &v1.Pod{}
		err := s.reconciler.Client.Get(ctx, client.ObjectKey{Name: resource.Owner, Namespace: resource.OwnerAttributes["namespace"]}, pod)
		if err != nil {
			return nil, err
		}
		cobj = pod
	} else {
		sa := &v1.ServiceAccount{}
		err := s.reconciler.Client.Get(ctx, client.ObjectKey{Name: resource.Owner, Namespace: resource.OwnerAttributes["namespace"]}, sa)
		if err != nil {
			return nil, err
		}
		cobj = sa
	}
	if err := controllerutil.SetOwnerReference(cobj, &generatorState, s.reconciler.Scheme); err != nil {
		return nil, err
	}

	err = s.reconciler.Client.Create(ctx, &generatorState)
	if err != nil {
		return nil, err
	}
	stringData := map[string]string{}
	for k, v := range data {
		stringData[k] = string(v)
	}
	return stringData, nil
}

type Resource struct {
	Name            string            `json:"name"`
	Owner           string            `json:"owner"`
	OwnerAttributes map[string]string `json:"ownerAttributes"`
	AuthMethod      string            `json:"authMethod"`
}

// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.
package server

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
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
	specMap                map[string][]*fedv1alpha1.AuthorizationSpec
	port                   string
	genParseTokenFn        func(ctx context.Context, onlyToken string, caCrt []byte) func(token *jwt.Token) (interface{}, error)
	generateSecretFn       func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, error)
	getSecretFn            func(ctx context.Context, storeName string, name string) ([]byte, error)
	deleteGeneratorStateFn func(ctx context.Context, namespace string, labels labels.Selector) error
}

func NewServerHandler(reconciler *externalsecrets.Reconciler, port string) *ServerHandler {
	s := &ServerHandler{
		reconciler: reconciler,
		mu:         sync.RWMutex{},
		specMap:    map[string][]*fedv1alpha1.AuthorizationSpec{},
		port:       port,
	}
	s.generateSecretFn = s.generateSecret
	s.getSecretFn = s.getSecret
	s.genParseTokenFn = s.genParseToken
	s.deleteGeneratorStateFn = s.deleteGeneratorState
	return s
}

func (s *ServerHandler) SetupEcho(ctx context.Context) *echo.Echo {
	e := echo.New()
	e.Server.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	e.POST("/secretstore/:secretStoreName/secrets/:secretName", s.postSecrets)
	e.POST("/generators/:generatorNamespace/:generatorKind/:generatorName", s.generateSecrets)
	e.DELETE("/generators/:generatorNamespace/:generatorKind/:generatorName", s.revokeSelf)
	e.POST("/generators/:generatorNamespace/revoke", s.revokeCredentialsOf)
	e.Logger.Fatal(e.Start(s.port))
	return e
}

func parseRSAPublicKey(key map[string]string) (*rsa.PublicKey, error) {
	nval, ok := key["n"]
	if !ok {
		return nil, errors.New("n not found in key")
	}
	n, err := base64.RawURLEncoding.DecodeString(nval)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}
	eval, ok := key["e"]
	if !ok {
		return nil, errors.New("e not found in key")
	}
	e, err := base64.RawURLEncoding.DecodeString(eval)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}
	// Convert decoded values to big.Int
	modulus := new(big.Int).SetBytes(n)
	exponent := new(big.Int).SetBytes(e)
	// Create RSA public key
	return &rsa.PublicKey{
		N: modulus,
		E: int(exponent.Int64()),
	}, nil
}

func findJWKS(ctx context.Context, issuer, onlyToken, caCrt string) (map[*fedv1alpha1.AuthorizationSpec]map[string]map[string]string, error) {
	var specs map[*fedv1alpha1.AuthorizationSpec]map[string]map[string]string
	var errs error
	authorizationSpecs := store.Get(issuer)
	// Needs to be individual as at this stage we are filling the store
	for _, spec := range authorizationSpecs {
		jwks, err := store.GetJWKS(ctx, []*fedv1alpha1.AuthorizationSpec{spec}, onlyToken, issuer, []byte(caCrt))
		if err != nil {
			errs = errors.Join(errs, err)
		}
		if jwks == nil {
			continue
		}
		if specs == nil {
			specs = map[*fedv1alpha1.AuthorizationSpec]map[string]map[string]string{}
		}
		specs[spec] = jwks
	}
	return specs, errs
}
func (s *ServerHandler) getJWKS(ctx context.Context, issuer, onlyToken, caCrt string) (map[string]map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	specs, ok := s.specMap[caCrt]
	if !ok {
		specs, err := findJWKS(ctx, issuer, onlyToken, caCrt)
		if err != nil {
			return nil, err
		}
		for spec := range specs {
			s.specMap[caCrt] = append(s.specMap[caCrt], spec)
		}
	}
	return store.GetJWKS(ctx, specs, onlyToken, issuer, []byte(caCrt))
}

func (s *ServerHandler) genParseToken(ctx context.Context, onlyToken string, caCrt []byte) func(token *jwt.Token) (interface{}, error) {
	return func(token *jwt.Token) (interface{}, error) {
		var err error
		issuer, err := token.Claims.GetIssuer()
		if err != nil {
			return nil, err
		}
		jwks, err := s.getJWKS(ctx, issuer, onlyToken, string(caCrt))
		if err != nil {
			return nil, err
		}
		kid := token.Header["kid"].(string)
		key, ok := jwks[kid]
		if key == nil || !ok {
			return nil, errors.New("found right store, but kid not found in jwks")
		}
		alg := key["alg"]
		switch alg {
		case "RS256":
			return parseRSAPublicKey(key)
		case "RS384":
			return parseRSAPublicKey(key)
		case "RS512":
			return parseRSAPublicKey(key)
		default:
			return nil, fmt.Errorf("algorithm %v not supported", alg)
		}
	}
}

func (s *ServerHandler) processRequest(c echo.Context, caCrt []byte) (*KubernetesClaims, error) {
	// Get Token from header
	token := c.Request().Header.Get("Authorization")
	onlyToken := strings.TrimPrefix(token, "Bearer ")
	parsedToken, err := jwt.ParseWithClaims(onlyToken, &KubernetesClaims{}, s.genParseTokenFn(c.Request().Context(), onlyToken, caCrt))
	if err != nil {
		return nil, err
	}
	claim, ok := parsedToken.Claims.(*KubernetesClaims)
	if !ok {
		return nil, errors.New("failed to parse token")
	}
	return claim, nil
}

type generateSecretRequest struct {
	CaCrt string `json:"ca.crt"`
}

func (s *ServerHandler) generateSecrets(c echo.Context) error {
	var req generateSecretRequest
	err := c.Bind(&req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	claim, err := s.processRequest(c, []byte(req.CaCrt))
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	AuthorizationSpecs := store.Get(claim.Issuer)
	generatorName := c.Param("generatorName")
	generatorKind := c.Param("generatorKind")
	generatorNamespace := c.Param("generatorNamespace")
	d := fedv1alpha1.AllowedGenerator{
		Name:      generatorName,
		Kind:      generatorKind,
		Namespace: generatorNamespace,
	}
	owner := claim.KubernetesIOInner.ServiceAccount.Name
	if claim.KubernetesIOInner.Pod != nil {
		owner = claim.KubernetesIOInner.Pod.Name
	}

	resource := &Resource{
		Name:       generatorName,
		AuthMethod: "KubernetesServiceAccount",
		Owner:      owner,
		OwnerAttributes: map[string]string{
			"namespace":            claim.KubernetesIOInner.Namespace,
			"issuer":               claim.Issuer,
			"serviceaccount-uid":   claim.KubernetesIOInner.ServiceAccount.UID,
			"service-account-name": claim.KubernetesIOInner.ServiceAccount.Name,
		},
	}
	if claim.KubernetesIOInner.Pod != nil {
		resource.OwnerAttributes["pod-uid"] = claim.KubernetesIOInner.Pod.UID
	}
	for _, spec := range AuthorizationSpecs {
		if contains(spec.AllowedGenerators, d) && spec.Subject.Subject == claim.Subject {
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

type postSecretRequest struct {
	CaCrt string `json:"ca.crt"`
}

func (s *ServerHandler) postSecrets(c echo.Context) error {
	var req postSecretRequest
	err := c.Bind(&req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	claim, err := s.processRequest(c, []byte(req.CaCrt))
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	AuthorizationSpecs := store.Get(claim.Issuer)
	storeName := c.Param("secretStoreName")
	name := c.Param("secretName")
	for _, spec := range AuthorizationSpecs {
		if slices.Contains(spec.AllowedClusterSecretStores, storeName) && spec.Subject.Subject == claim.Subject {
			secret, err := s.getSecretFn(c.Request().Context(), storeName, name)
			if err != nil {
				return c.JSON(http.StatusBadRequest, err.Error())
			}
			return c.JSON(http.StatusOK, string(secret))
		}
	}
	return c.JSON(http.StatusNotFound, "Not Found")
}

type revokeSelfRequest struct {
	CaCrt string `json:"ca.crt"`
}

func (s *ServerHandler) revokeSelf(c echo.Context) error {
	var req revokeSelfRequest
	err := c.Bind(&req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	claim, err := s.processRequest(c, []byte(req.CaCrt))
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	AuthorizationSpecs := store.Get(claim.Issuer)
	generatorNamespace := c.Param("generatorNamespace")
	generatorName := c.Param("generatorName")
	generatorKind := c.Param("generatorKind")
	for _, spec := range AuthorizationSpecs {
		if !contains(spec.AllowedGenerators, fedv1alpha1.AllowedGenerator{
			Name:      generatorName,
			Kind:      generatorKind,
			Namespace: generatorNamespace,
		}) || spec.Subject.Subject != claim.Subject {
			continue
		}
		owner := claim.KubernetesIOInner.ServiceAccount.Name
		if claim.KubernetesIOInner.Pod != nil {
			owner = claim.KubernetesIOInner.Pod.Name
		}
		labels := labels.SelectorFromSet(labels.Set{
			"federation.externalsecrets.com/owner":          owner,
			"federation.externalsecrets.com/generator":      generatorName,
			"federation.externalsecrets.com/generator-kind": generatorKind,
		})
		err = s.deleteGeneratorStateFn(c.Request().Context(), generatorNamespace, labels)
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
	CaCert    string `json:"ca.crt"`
}

func (s *ServerHandler) revokeCredentialsOf(c echo.Context) error {
	var req deleteRequest
	err := c.Bind(&req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	claim, err := s.processRequest(c, []byte(req.CaCert))
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	AuthorizationSpecs := store.Get(claim.Issuer)
	generatorNamespace := c.Param("generatorNamespace")
	for _, spec := range AuthorizationSpecs {
		if contains(spec.AllowedGeneratorStates, fedv1alpha1.AllowedGeneratorState{
			Namespace: generatorNamespace,
		}) && spec.Subject.Subject == claim.Subject {
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

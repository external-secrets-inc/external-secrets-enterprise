// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.
package server

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	fedv1alpha1 "github.com/external-secrets/external-secrets/apis/federation/v1alpha1"
	externalsecrets "github.com/external-secrets/external-secrets/pkg/controllers/externalsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	store "github.com/external-secrets/external-secrets/pkg/federation/store"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type ServerHandler struct {
	reconciler       *externalsecrets.Reconciler
	mu               sync.RWMutex
	specMap          map[string][]*fedv1alpha1.AuthorizationSpec
	port             string
	genParseTokenFn  func(ctx context.Context, onlyToken string, caCrt []byte) func(token *jwt.Token) (interface{}, error)
	generateSecretFn func(ctx context.Context, generatorName string, generatorKind string, namespace string) (map[string]string, error)
	getSecretFn      func(ctx context.Context, storeName string, name string) ([]byte, error)
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
	return s
}

func (s *ServerHandler) SetupEcho(ctx context.Context) *echo.Echo {
	e := echo.New()
	e.Server.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	e.POST("/secretstore/:secretStoreName/secrets/:secretName", s.postSecrets)
	e.POST("/generate/:generatorNamespace/:generatorKind/:generatorName", s.generateSecrets)
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

func (s *ServerHandler) processRequest(c echo.Context) (string, string, error) {
	// Get Token from header
	token := c.Request().Header.Get("Authorization")
	onlyToken := strings.TrimPrefix(token, "Bearer ")
	payload := map[string]string{}
	err := c.Bind(&payload)
	if err != nil {
		return "", "", err
	}
	caCrt := payload["ca.crt"]
	parsedToken, err := jwt.Parse(onlyToken, s.genParseTokenFn(c.Request().Context(), onlyToken, []byte(caCrt)))
	if err != nil {
		return "", "", err
	}
	issuer, err := parsedToken.Claims.GetIssuer()
	if err != nil {
		return "", "", err
	}
	sub, err := parsedToken.Claims.GetSubject()
	if err != nil {
		return "", "", err
	}
	return issuer, sub, nil
}

func (s *ServerHandler) generateSecrets(c echo.Context) error {
	issuer, sub, err := s.processRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	AuthorizationSpecs := store.Get(issuer)
	generatorName := c.Param("generatorName")
	generatorKind := c.Param("generatorKind")
	generatorNamespace := c.Param("generatorNamespace")
	d := fedv1alpha1.AllowedGenerator{
		Name:      generatorName,
		Kind:      generatorKind,
		Namespace: generatorNamespace,
	}
	for _, spec := range AuthorizationSpecs {
		if contains(spec.AllowedGenerators, d) && spec.Subject.Subject == sub {
			secret, err := s.generateSecretFn(c.Request().Context(), generatorName, generatorKind, generatorNamespace)
			if err != nil {
				return c.JSON(http.StatusBadRequest, err.Error())
			}
			return c.JSON(http.StatusOK, secret)
		}
	}
	return c.JSON(http.StatusNotFound, "Not Found")
}

func contains(slice []fedv1alpha1.AllowedGenerator, item fedv1alpha1.AllowedGenerator) bool {
	for _, v := range slice {
		if v.Name == item.Name && v.Kind == item.Kind && v.Namespace == item.Namespace {
			return true
		}
	}
	return false
}
func (s *ServerHandler) postSecrets(c echo.Context) error {
	issuer, sub, err := s.processRequest(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	AuthorizationSpecs := store.Get(issuer)
	storeName := c.Param("secretStoreName")
	name := c.Param("secretName")
	for _, spec := range AuthorizationSpecs {
		if slices.Contains(spec.AllowedClusterSecretStores, storeName) && spec.Subject.Subject == sub {
			secret, err := s.getSecretFn(c.Request().Context(), storeName, name)
			if err != nil {
				return c.JSON(http.StatusBadRequest, err.Error())
			}
			return c.JSON(http.StatusOK, string(secret))
		}
	}
	return c.JSON(http.StatusNotFound, "Not Found")
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

func (s *ServerHandler) generateSecret(ctx context.Context, generatorName, generatorKind, namespace string) (map[string]string, error) {
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
	// TODO[gusfcarvalho]: Generator State is currently IGNORED.
	// Meaning, we cannot trigger any API to delete the generated information so far
	// We need to get the Generator State and be sure to create / update it.
	// And then add another endpoint to trigger a reconcile of it.
	data, _, err := generator.Generate(ctx, obj, s.reconciler.Client, namespace)
	if err != nil {
		return nil, err
	}
	stringData := map[string]string{}
	for k, v := range data {
		stringData[k] = string(v)
	}
	return stringData, nil
}

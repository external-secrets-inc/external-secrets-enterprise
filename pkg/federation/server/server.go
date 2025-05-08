// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package server

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net"
	"slices"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/federation/v1alpha1"
	externalsecrets "github.com/external-secrets/external-secrets/pkg/controllers/externalsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	store "github.com/external-secrets/external-secrets/pkg/federation/store"
)

type ServerHandler struct {
	reconciler *externalsecrets.Reconciler
	mu         sync.RWMutex
	specMap    map[string][]*v1alpha1.AuthorizationSpec
	port       string
}

func NewServerHandler(reconciler *externalsecrets.Reconciler, port string) *ServerHandler {
	return &ServerHandler{
		reconciler: reconciler,
		mu:         sync.RWMutex{},
		specMap:    map[string][]*v1alpha1.AuthorizationSpec{},
		port:       port,
	}
}

func (s *ServerHandler) SetupEcho(ctx context.Context) *echo.Echo {
	e := echo.New()
	e.Server.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	e.POST("/secretstore/:secretStoreName/secrets/:secretName", s.postSecrets)
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

func findJWKS(ctx context.Context, issuer, onlyToken, caCrt string) (map[*v1alpha1.AuthorizationSpec]map[string]map[string]string, error) {
	var specs map[*v1alpha1.AuthorizationSpec]map[string]map[string]string
	var errs error
	authorizationSpecs := store.Get(issuer)
	// Needs to be individual as at this stage we are filling the store
	for _, spec := range authorizationSpecs {
		jwks, err := store.GetJWKS(ctx, []*v1alpha1.AuthorizationSpec{spec}, onlyToken, issuer, []byte(caCrt))
		if err != nil {
			errs = errors.Join(errs, err)
		}
		if jwks == nil {
			continue
		}
		if specs == nil {
			specs = map[*v1alpha1.AuthorizationSpec]map[string]map[string]string{}
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
func (s *ServerHandler) postSecrets(c echo.Context) error {
	// Get Token from header
	token := c.Request().Header.Get("Authorization")
	onlyToken := strings.TrimPrefix(token, "Bearer ")
	payload := map[string]string{}
	err := c.Bind(&payload)
	if err != nil {
		return c.JSON(400, err.Error())
	}
	caCrt := payload["ca.crt"]
	parsedToken, err := jwt.Parse(onlyToken, s.genParseToken(c.Request().Context(), onlyToken, []byte(caCrt)))
	if err != nil {
		return c.JSON(401, err.Error())
	}
	issuer, err := parsedToken.Claims.GetIssuer()
	if err != nil {
		return c.JSON(402, err.Error())
	}
	sub, err := parsedToken.Claims.GetSubject()
	if err != nil {
		return c.JSON(403, err.Error())
	}

	AuthorizationSpecs := store.Get(issuer)
	storeName := c.Param("secretStoreName")
	name := c.Param("secretName")
	for _, spec := range AuthorizationSpecs {
		if slices.Contains(spec.AllowedClusterSecretStores, storeName) && spec.Subject.Subject == sub {
			secret, err := s.getSecret(c.Request().Context(), storeName, name)
			if err != nil {
				return c.JSON(400, err.Error())
			}
			return c.JSON(200, string(secret))
		}
	}
	return c.JSON(404, "Not Found")
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

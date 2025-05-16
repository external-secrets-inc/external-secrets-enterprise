// Copyright External Secrets Inc. All Rights Reserved

package ssh

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

type Generator struct{}

const (
	defaultKeyType = genv1alpha1.SSHKeyTypeRSA
	defaultBits    = 4096

	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
	errGetToken  = "unable to get authorization token: %w"
)

type RSAGenerateFunc func(
	bits int,
) (string, string, error)

func (g *Generator) Generate(_ context.Context, jsonSpec *apiextensions.JSON, _ client.Client, _ string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(
		jsonSpec,
		generateRSA,
	)
}

func (g *Generator) Cleanup(_ context.Context, jsonSpec *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(jsonSpec *apiextensions.JSON, rsaGen RSAGenerateFunc) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	keyType := defaultKeyType
	if res.Spec.KeyType != "" {
		keyType = res.Spec.KeyType
	}

	if keyType == genv1alpha1.SSHKeyTypeRSA {
		bits := defaultBits
		if res.Spec.RSAConfig.Bits > 0 {
			bits = res.Spec.RSAConfig.Bits
		}

		privPEM, pubKey, err := rsaGen(bits)
		if err != nil {
			return nil, nil, err
		}
		return map[string][]byte{
			"id_rsa":     []byte(privPEM),
			"id_rsa.pub": []byte(pubKey),
		}, nil, nil
	}
	return nil, nil, fmt.Errorf("unsupported key type: %s", res.Spec.KeyType)
}

func generateRSA(
	bits int,
) (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key as PEM
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privPEM := new(bytes.Buffer)
	err = pem.Encode(privPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to encode private key: %w", err)
	}

	// Generate public key in authorized_keys format
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate public key: %w", err)
	}
	pubKey := string(ssh.MarshalAuthorizedKey(pub))

	return privPEM.String(), pubKey, nil
}

func parseSpec(data []byte) (*genv1alpha1.SSH, error) {
	var spec genv1alpha1.SSH
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.SSHKind, &Generator{})
}

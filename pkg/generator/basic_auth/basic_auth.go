// Copyright External Secrets Inc. All Rights Reserved

package basic_auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/sethvargo/go-password/password"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/nwtgck/go-fakelish"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

type Generator struct{}

const (
	defaultPasswordLength = 24
	defaultSymbolChars    = "~!@#$%^&*()_+`-={}|[]\\:\"<>?,./"
	digitFactor           = 0.25
	symbolFactor          = 0.25

	defaultUsernameLength = 8
	defaultWordCount      = 1
	defaultSeparator      = "_"

	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
	errGetToken  = "unable to get authorization token: %w"
)

type usernameGenerateFunc func(
	len int,
	prefix string,
	sufix string,
	wordCount int,
	separator string,
	includeNumbers bool,
) (string, error)

type passwordGenerateFunc func(
	len int,
	symbols int,
	symbolCharacters string,
	digits int,
	noUpper bool,
	allowRepeat bool,
) (string, error)

func (g *Generator) Generate(_ context.Context, jsonSpec *apiextensions.JSON, _ client.Client, _ string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(
		jsonSpec,
		generateUsername,
		generateSafePassword,
	)
}

func (g *Generator) Cleanup(_ context.Context, jsonSpec *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(
	jsonSpec *apiextensions.JSON,
	userGen usernameGenerateFunc,
	passGen passwordGenerateFunc,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	usernameSpec := res.Spec.Username

	usernameLen := defaultUsernameLength
	if usernameSpec.Length > 0 {
		usernameLen = usernameSpec.Length
	}
	usernamePrefix := ""
	if usernameSpec.Prefix != nil {
		usernamePrefix = *usernameSpec.Prefix
	}
	usernameSufix := ""
	if usernameSpec.Sufix != nil {
		usernameSufix = *usernameSpec.Sufix
	}
	usernameWordCount := defaultWordCount
	if usernameSpec.WordCount > 0 {
		usernameWordCount = usernameSpec.WordCount
	}
	usernameSeparator := defaultSeparator
	if usernameSpec.Separator != nil {
		usernameSeparator = *usernameSpec.Separator
	}

	user, err := userGen(
		usernameLen,
		usernamePrefix,
		usernameSufix,
		usernameWordCount,
		usernameSeparator,
		usernameSpec.IncludeNumbers,
	)
	if err != nil {
		return nil, nil, err
	}

	passwordSpec := res.Spec.Password

	symbolCharacters := defaultSymbolChars
	if passwordSpec.SymbolCharacters != nil {
		symbolCharacters = *passwordSpec.SymbolCharacters
	}
	passLen := defaultPasswordLength
	if passwordSpec.Length > 0 {
		passLen = passwordSpec.Length
	}
	digits := int(float32(passLen) * digitFactor)
	if passwordSpec.Digits != nil {
		digits = *passwordSpec.Digits
	}
	symbols := int(float32(passLen) * symbolFactor)
	if passwordSpec.Symbols != nil {
		symbols = *passwordSpec.Symbols
	}
	pass, err := passGen(
		passLen,
		symbols,
		symbolCharacters,
		digits,
		passwordSpec.NoUpper,
		passwordSpec.AllowRepeat,
	)
	if err != nil {
		return nil, nil, err
	}

	return map[string][]byte{
		"username": []byte(user),
		"password": []byte(pass),
	}, nil, nil
}

func generateUsername(
	userLen int,
	prefix string,
	sufix string,
	wordCount int,
	separator string,
	includeNumbers bool,
) (string, error) {
	if wordCount <= 0 || userLen <= 0 {
		return "", fmt.Errorf("invalid wordCount=%d or len=%d", wordCount, userLen)
	}

	generatedUsername := ""
	for i := 0; i < wordCount; i++ {
		// Generate a fake word
		fakeWord := fakelish.GenerateFakeWordByLength(userLen)
		// Capitalize the first letter
		fakeWord = strings.ToLower(fakeWord)

		// Append the separator if it's not the first word
		if i != 0 {
			generatedUsername += separator
		}

		// Append the fake word
		generatedUsername += fakeWord
	}

	generatedUsername = prefix + generatedUsername + sufix

	// If the user wants numbers
	if includeNumbers {
		// Loop through the numbers count
		for i := 0; i < 4; i++ {
			// Create ioReader
			ioReader := rand.Reader
			// Random integer between 0 and 9
			randomNumber, err := rand.Int(ioReader, new(big.Int).SetInt64(9))
			if err != nil {
				return "", err
			}
			// Append the number to the initial string
			generatedUsername += fmt.Sprintf("%d", randomNumber)
		}
	}
	return generatedUsername, nil
}

func generateSafePassword(
	passLen int,
	symbols int,
	symbolCharacters string,
	digits int,
	noUpper bool,
	allowRepeat bool,
) (string, error) {
	gen, err := password.NewGenerator(&password.GeneratorInput{
		Symbols: symbolCharacters,
	})
	if err != nil {
		return "", err
	}
	return gen.Generate(
		passLen,
		digits,
		symbols,
		noUpper,
		allowRepeat,
	)
}

func parseSpec(data []byte) (*genv1alpha1.BasicAuth, error) {
	var spec genv1alpha1.BasicAuth
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.BasicAuthKind, &Generator{})
}

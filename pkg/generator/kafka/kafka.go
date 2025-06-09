// Copyright External Secrets Inc. All Rights Reserved

package kafka

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/generator/password"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Generator struct{}

const (
	defaultPort       = "5432"
	defaultUser       = "kafka"
	defaultDbName     = "kafka"
	defaultSuffixSize = 8
)

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, err
	}

	client, err := newClient(ctx, &res.Spec, kube, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create client connection: %w", err)
	}

	user, err := createUser(ctx, client, &res.Spec)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create or update user: %w", err)
	}

	username, ok := user["username"]
	if !ok {
		return nil, nil, fmt.Errorf("user not found in response")
	}

	rawState, err := json.Marshal(&genv1alpha1.KafkaUserState{
		Username: string(username),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal state: %w", err)
	}

	return user, &apiextensions.JSON{Raw: rawState}, nil
}

func (g *Generator) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, previousStatus genv1alpha1.GeneratorProviderState, kclient client.Client, namespace string) error {
	if previousStatus == nil {
		return fmt.Errorf("missing previous status")
	}
	status, err := parseStatus(previousStatus.Raw)
	if err != nil {
		return err
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return err
	}
	client, err := newClient(ctx, &res.Spec, kclient, namespace)
	if err != nil {
		return err
	}

	err = dropUser(ctx, client, status.Username)
	if err != nil {
		return fmt.Errorf("unable to drop user: %w", err)
	}

	return nil
}

func newClient(ctx context.Context, spec *genv1alpha1.KafkaSpec, kclient client.Client, ns string) (*kafka.Client, error) {
	password, err := resolvers.SecretKeyRef(ctx, kclient, resolvers.EmptyStoreKind, ns, &esmeta.SecretKeySelector{
		Namespace: &ns,
		Name:      spec.Auth.Password.Name,
		Key:       spec.Auth.Password.Key,
	})
	if err != nil {
		return nil, err
	}

	mechanism, err := scram.Mechanism(scram.SHA512, spec.Auth.Username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create SCRAM mechanism: %w", err)
	}

	sharedTransport := &kafka.Transport{
		SASL: mechanism,
	}

	return &kafka.Client{
		Addr:      kafka.TCP(spec.Addresses...),
		Timeout:   10 * time.Second,
		Transport: sharedTransport,
	}, nil
}

func createUser(ctx context.Context, client *kafka.Client, spec *genv1alpha1.KafkaSpec) (map[string][]byte, error) {
	username := spec.User.Username
	suffixSize := defaultSuffixSize
	if spec.User.SuffixSize != nil {
		suffixSize = *spec.User.SuffixSize
	}
	suffix, err := generateRandomString(suffixSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random suffix: %w", err)
	}

	if suffix != "" {
		username = fmt.Sprintf("%s_%s", username, suffix)
	}

	pass, err := generatePassword(genv1alpha1.Password{
		Spec: genv1alpha1.PasswordSpec{
			SymbolCharacters: ptr.To("~!@#$%^&*()_+-={}|[]:<>?,./"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	upsertion, err := generateSCRAMCredentials(username, string(pass))
	if err != nil {
		return nil, fmt.Errorf("failed to generate SCRAM credentials: %w", err)
	}

	resp, err := client.AlterUserScramCredentials(ctx, &kafka.AlterUserScramCredentialsRequest{
		Addr:       client.Addr,
		Upsertions: []kafka.UserScramCredentialsUpsertion{upsertion},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to alter SCRAM credentials for user %s: %w", username, err)
	}

	for _, result := range resp.Results {
		if result.Error != nil {
			return nil, fmt.Errorf("failed to create SCRAM credentials for user %s: %w", result.User, result.Error)
		}
	}

	err = addPermissionsToUser(ctx, client, username, spec.User.Permissions)
	if err != nil {
		errPermision := fmt.Errorf("failed to add permissions for user %s: %w", username, err)
		errDrop := dropUser(ctx, client, username)
		if errDrop != nil {
			return nil, errors.Join(errPermision, fmt.Errorf("failed to drop user %s after permission error: %w", username, err))
		}
		return nil, errPermision
	}

	return map[string][]byte{
		"username": []byte(username),
		"password": pass,
	}, nil
}

func dropUser(ctx context.Context, client *kafka.Client, username string) error {
	resp, err := client.AlterUserScramCredentials(ctx, &kafka.AlterUserScramCredentialsRequest{
		Addr: client.Addr,
		Deletions: []kafka.UserScramCredentialsDeletion{{
			Name:      username,
			Mechanism: kafka.ScramMechanismSha512,
		}},
	})
	if err != nil {
		return fmt.Errorf("failed to alter SCRAM credentials for user %s: %w", username, err)
	}

	for _, result := range resp.Results {
		if result.Error != nil {
			return fmt.Errorf("failed to delete SCRAM credentials for user %s: %w", result.User, result.Error)
		}
	}

	return dropUserPermissions(ctx, client, username)
}

func generateSCRAMCredentials(username, password string) (kafka.UserScramCredentialsUpsertion, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return kafka.UserScramCredentialsUpsertion{}, err
	}

	iterations := 4096
	saltedPassword, err := pbkdf2.Key(sha512.New, password, salt, iterations, sha512.Size)
	if err != nil {
		return kafka.UserScramCredentialsUpsertion{}, fmt.Errorf("failed to generate salted password: %w", err)
	}

	return kafka.UserScramCredentialsUpsertion{
		Name:           username,
		Mechanism:      kafka.ScramMechanismSha512,
		Iterations:     iterations,
		Salt:           salt,
		SaltedPassword: saltedPassword,
	}, nil
}

func addPermissionsToUser(ctx context.Context, client *kafka.Client, username string, permissions []genv1alpha1.KafkaUserPermissions) error {
	// Ensure the principal follows the expected format: "User:<username>"
	principal := fmt.Sprintf("User:%s", username)

	var acls []kafka.ACLEntry
	for _, permission := range permissions {
		host := permission.Host
		if host == "" {
			host = client.Addr.String()
		}

		var resourceType kafka.ResourceType
		if err := resourceType.UnmarshalText([]byte(permission.ResourceType)); err != nil {
			return fmt.Errorf("invalid resourceType %q: %w", permission.ResourceType, err)
		}

		var patternType kafka.PatternType
		if err := patternType.UnmarshalText([]byte(permission.ResourcePatternType)); err != nil {
			return fmt.Errorf("invalid resourcePatternType %q: %w", permission.ResourcePatternType, err)
		}

		var operation kafka.ACLOperationType
		if err := operation.UnmarshalText([]byte(permission.Operation)); err != nil {
			return fmt.Errorf("invalid operation %q: %w", permission.Operation, err)
		}

		var permissionType kafka.ACLPermissionType
		if err := permissionType.UnmarshalText([]byte(permission.PermissionType)); err != nil {
			return fmt.Errorf("invalid permissionType %q: %w", permission.PermissionType, err)
		}

		acls = append(acls, kafka.ACLEntry{
			ResourceType:        resourceType,
			ResourceName:        permission.ResourceName,
			ResourcePatternType: patternType,
			Principal:           principal,
			Host:                host,
			Operation:           operation,
			PermissionType:      permissionType,
		})
	}

	resp, err := client.CreateACLs(ctx, &kafka.CreateACLsRequest{
		Addr: client.Addr,
		ACLs: acls,
	})
	if err != nil {
		return fmt.Errorf("failed to create ACLs: %w", err)
	}

	var allErrs []error
	for i, aclErr := range resp.Errors {
		if aclErr != nil {
			allErrs = append(allErrs, fmt.Errorf("ACL entry %d for user %s failed: %w", i, username, aclErr))
		}
	}

	if len(allErrs) > 0 {
		return errors.Join(allErrs...)
	}

	return nil
}

func dropUserPermissions(ctx context.Context, client *kafka.Client, username string) error {
	principal := fmt.Sprintf("User:%s", username)

	// Create a broad filter: delete *any* ACL with this principal
	filter := kafka.DeleteACLsFilter{
		ResourceTypeFilter:        kafka.ResourceTypeAny,
		ResourceNameFilter:        "",
		ResourcePatternTypeFilter: kafka.PatternTypeAny,
		PrincipalFilter:           principal,
		HostFilter:                "*",
		Operation:                 kafka.ACLOperationTypeAny,
		PermissionType:            kafka.ACLPermissionTypeAny,
	}

	resp, err := client.DeleteACLs(ctx, &kafka.DeleteACLsRequest{
		Addr:    client.Addr,
		Filters: []kafka.DeleteACLsFilter{filter},
	})
	if err != nil {
		return fmt.Errorf("failed to send DeleteACLs request: %w", err)
	}

	var allErrs []error
	for i, result := range resp.Results {
		if result.Error != nil {
			allErrs = append(allErrs, fmt.Errorf("failed to delete ACLs for filter %d: %w", i, result.Error))
		}
	}

	if len(allErrs) > 0 {
		return errors.Join(allErrs...)
	}
	return nil
}

func generatePassword(
	passSpec genv1alpha1.Password,
) ([]byte, error) {
	gen := password.Generator{}
	rawPassSpec, err := yaml.Marshal(passSpec)
	if err != nil {
		return nil, err
	}
	passMap, _, err := gen.Generate(context.TODO(), &apiextensions.JSON{Raw: rawPassSpec}, nil, "")

	if err != nil {
		return nil, err
	}

	pass, ok := passMap["password"]
	if !ok {
		return nil, fmt.Errorf("password not found in generated map")
	}
	return pass, nil
}

func generateRandomString(size int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	limit := big.NewInt(int64(len(charset)))

	b := make([]byte, size)
	for i := range b {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

func parseSpec(data []byte) (*genv1alpha1.Kafka, error) {
	var spec genv1alpha1.Kafka
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func parseStatus(data []byte) (*genv1alpha1.KafkaUserState, error) {
	var state genv1alpha1.KafkaUserState
	err := json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.KafkaKind, &Generator{})
}

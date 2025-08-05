// Copyright External Secrets Inc. All Rights Reserved

package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	enterprise "github.com/external-secrets/external-secrets/apis/enterprise/generators/v1alpha1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/generator/password"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
	"github.com/labstack/gommon/log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	errFetchSecretRef   = "could not fetch secret ref: %w"
	errGeneratePassword = "could not generate password"
	errCreateUser       = "could not create user: %w"
	errUpdateUser       = "could not update user: %w"
	errDeleteUser       = "could not delete user %s: %w"
	errCloseClient      = "could not close mongo client: %w"
	errGenerateUsername = "could not generate username: %w"
	errMongoDBConnect   = "could not connect to mongoDB: %w"
	errMissingState     = "missing generator state"
	errCreateState      = "could not create generator state: %w"
	errMissingAdminUser = "missing admin username"
)

const (
	DefaultUsernameLength = 8
)

type MongoClient interface {
	Ping(ctx context.Context, rp *readpref.ReadPref) error
	Database(name string, opts ...*options.DatabaseOptions) *mongo.Database
	Disconnect(ctx context.Context) error
}

type ClientFactory interface {
	New(ctx context.Context, uri string) (MongoClient, error)
}

type MongoDB struct {
	clientFactory ClientFactory
}

func (g *MongoDB) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kclient client.Client, ns string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	gen, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, err
	}

	adminUsername, adminPwd, err := getAdminCredentials(ctx, gen, kclient, ns)
	if err != nil {
		return nil, nil, err
	}
	mongoURI := buildMongoURI(adminUsername, adminPwd, gen.Spec.Database.Host, gen.Spec.Database.Port, gen.Spec.Database.AdminDB)

	client, err := g.clientFactory.New(ctx, mongoURI)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Warnf("failed to disconnect mongo client: %v", err)
		}
	}()

	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf(errMongoDBConnect, err)
	}

	adminDB := client.Database(gen.Spec.Database.AdminDB)

	username, err := buildUsername(gen.Spec.User.Name)
	if err != nil {
		return nil, nil, err
	}
	password, err := generatePassword()
	if err != nil {
		return nil, nil, err
	}

	err = ensureUser(ctx, adminDB, username, password, gen.Spec.User.Roles)
	if err != nil {
		return nil, nil, err
	}

	rawState, err := json.Marshal(&enterprise.MongoDBUserState{
		User: username,
	})
	if err != nil {
		return nil, nil, fmt.Errorf(errCreateState, err)
	}
	return map[string][]byte{
		"username": []byte(username),
		"password": []byte(password),
	}, &apiextensions.JSON{Raw: rawState}, nil
}

func (g *MongoDB) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, generatorState genv1alpha1.GeneratorProviderState, kclient client.Client, ns string) error {
	if generatorState == nil {
		return fmt.Errorf(errMissingState)
	}

	state, err := parseState(generatorState.Raw)
	if err != nil {
		return err
	}

	gen, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return err
	}

	adminUsername, adminPwd, err := getAdminCredentials(ctx, gen, kclient, ns)
	if err != nil {
		return err
	}
	mongoURI := buildMongoURI(adminUsername, adminPwd, gen.Spec.Database.Host, gen.Spec.Database.Port, gen.Spec.Database.AdminDB)

	client, err := g.clientFactory.New(ctx, mongoURI)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Warnf("failed to disconnect mongo client: %v", err)
		}
	}()

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf(errMongoDBConnect, err)
	}

	adminDB := client.Database(gen.Spec.Database.AdminDB)
	cmd := bson.D{{Key: "dropUser", Value: state.User}}
	res := adminDB.RunCommand(ctx, cmd)
	if err := res.Err(); err != nil {
		return fmt.Errorf(errDeleteUser, state.User, err)
	}

	return nil
}

func (g *MongoDB) GetCleanupPolicy(obj *apiextensions.JSON) (*genv1alpha1.CleanupPolicy, error) {
	return nil, nil
}

func (g *MongoDB) LastActivityTime(ctx context.Context, obj *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, kube client.Client, namespace string) (time.Time, bool, error) {
	return time.Time{}, false, nil
}
func (g *MongoDB) GetKeys() map[string]string {
	return map[string]string{
		"username": "MongoDB user login name",
		"password": "MongoDB user password",
	}
}

func ensureUser(ctx context.Context, db *mongo.Database, username, password string, rolesSpec []enterprise.MongoDBRole) error {
	err := manageUser(ctx, db, "createUser", username, password, rolesSpec)
	if err != nil {
		if func() mongo.CommandError {
			var target mongo.CommandError
			_ = errors.As(err, &target)
			return target
		}().Code != 51003 {
			return fmt.Errorf(errCreateUser, err)
		}
		err = manageUser(ctx, db, "updateUser", username, password, rolesSpec)
		if err != nil {
			return fmt.Errorf(errUpdateUser, err)
		}
	}
	return nil
}

func getAdminCredentials(
	ctx context.Context,
	spec *enterprise.MongoDB,
	kube client.Client,
	ns string,
) (string, string, error) {
	var adminUser *string
	if spec.Spec.Auth.SCRAM.SecretRef.Username != nil {
		userSelector := spec.Spec.Auth.SCRAM.SecretRef.Username
		adminUserFromRef, err := getFromSecretRef(ctx, userSelector, "", kube, ns)

		if err == nil {
			adminUser = &adminUserFromRef
		}
	}
	if adminUser == nil {
		adminUser = spec.Spec.Auth.SCRAM.Username
		if adminUser == nil || *adminUser == "" {
			return "", "", fmt.Errorf(errMissingAdminUser)
		}
	}

	pwdSelector := &spec.Spec.Auth.SCRAM.SecretRef.Password
	adminPwd, err := getFromSecretRef(ctx, pwdSelector, "", kube, ns)
	if err != nil {
		return "", "", err
	}
	return *adminUser, adminPwd, nil
}

func manageUser(ctx context.Context, db *mongo.Database, action, username, password string, rolesSpec []enterprise.MongoDBRole) error {
	roles := make([]interface{}, len(rolesSpec))
	for i, r := range rolesSpec {
		roles[i] = bson.D{{Key: "role", Value: r.Name}, {Key: "db", Value: r.DB}}
	}

	cmd := bson.D{
		{Key: action, Value: username},
		{Key: "pwd", Value: password},
		{Key: "roles", Value: roles},
	}
	res := db.RunCommand(ctx, cmd)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func parseSpec(data []byte) (*enterprise.MongoDB, error) {
	var spec enterprise.MongoDB
	err := json.Unmarshal(data, &spec)
	return &spec, err
}

func parseState(data []byte) (*enterprise.MongoDBUserState, error) {
	var state enterprise.MongoDBUserState
	err := json.Unmarshal(data, &state)
	return &state, err
}

func getFromSecretRef(ctx context.Context, keySelector *esmeta.SecretKeySelector, storeKind string, kube client.Client, namespace string) (string, error) {
	value, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, keySelector)
	if err != nil {
		return "", fmt.Errorf(errFetchSecretRef, err)
	}

	return value, err
}

func buildUsername(prefix string) (string, error) {
	suffix, err := utils.GenerateRandomString(DefaultUsernameLength)
	if err != nil {
		return "", fmt.Errorf(errGenerateUsername, err)
	}
	if prefix != "" {
		return prefix + "_" + suffix, nil
	}
	return suffix, nil
}

func generatePassword() (string, error) {
	symbolChars := "~!$^&*()_+`-={}|<>,."
	passwordSpec := genv1alpha1.PasswordSpec{
		SymbolCharacters: &symbolChars,
	}

	passwordJSON, err := yaml.Marshal(genv1alpha1.Password{Spec: passwordSpec})
	if err != nil {
		return "", err
	}

	passwordGen := password.Generator{}
	passwordMap, _, err := passwordGen.Generate(context.TODO(), &apiextensions.JSON{Raw: passwordJSON}, nil, "")

	if err != nil {
		return "", err
	}

	passwordBytes, ok := passwordMap["password"]
	if !ok {
		return "", fmt.Errorf(errGeneratePassword)
	}

	return string(passwordBytes), nil
}

func buildMongoURI(user, pwd, host string, port int, authDB string) string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=%s", user, pwd, host, port, authDB)
}

type defaultClientFactory struct{}

func (defaultClientFactory) New(ctx context.Context, uri string) (MongoClient, error) {
	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func init() {
	genv1alpha1.Register(enterprise.MongoDBKind, &MongoDB{clientFactory: defaultClientFactory{}})
	genv1alpha1.RegisterGeneric(enterprise.MongoDBKind, &enterprise.MongoDB{})
}

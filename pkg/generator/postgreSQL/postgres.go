// Copyright External Secrets Inc. All Rights Reserved

package postgresql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"strings"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/generator/password"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Generator struct{}

const (
	defaultPort   = "5432"
	defaultUser   = "postgres"
	defaultDbName = "postgres"
)

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, err
	}

	db, err := newConnection(ctx, &res.Spec, kube, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create db connection: %w", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			fmt.Printf("failed to close db: %v", err)
		}
	}()

	err = db.Ping()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to ping the database: %w", err)
	}

	user, err := createOrReplaceUser(db, &res.Spec)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create or replace user: %w", err)
	}

	username, ok := user["user"]
	if !ok {
		return nil, nil, fmt.Errorf("user not found in response")
	}

	rawState, err := json.Marshal(&genv1alpha1.PostgreSqlUserState{
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
	db, err := newConnection(ctx, &res.Spec, kclient, namespace)
	if err != nil {
		return err
	}
	defer func() {
		err := db.Close()
		if err != nil {
			fmt.Printf("failed to close db: %v", err)
		}
	}()

	err = db.Ping()
	if err != nil {
		return fmt.Errorf("unable to ping the database: %w", err)
	}

	err = dropUser(db, status.Username, res.Spec.Auth.Username, res.Spec.User.DestructiveCleanup)
	if err != nil {
		return fmt.Errorf("unable to drop user: %w", err)
	}

	return nil
}

func newConnection(ctx context.Context, spec *genv1alpha1.PostgreSqlSpec, kclient client.Client, ns string) (*sql.DB, error) {
	dbName := defaultDbName
	if spec.Database != "" {
		dbName = spec.Database
	}

	port := defaultPort
	if spec.Port != "" {
		port = spec.Port
	}

	user := defaultUser
	if spec.Auth.Username != "" {
		user = spec.Auth.Username
	}
	password, err := resolvers.SecretKeyRef(ctx, kclient, resolvers.EmptyStoreKind, ns, &esmeta.SecretKeySelector{
		Namespace: &ns,
		Name:      spec.Auth.Password.Name,
		Key:       spec.Auth.Password.Key,
	})
	if err != nil {
		return nil, err
	}

	psqlInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		spec.Host, port, user, password, dbName,
	)

	return sql.Open("postgres", psqlInfo)
}

func getExistingRoles(db *sql.DB) ([]string, error) {
	var current_rows = make([]string, 0)
	rows, err := db.Query("SELECT rolname FROM pq_roles")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var rolname string
		err = rows.Scan(&rolname)
		if err != nil {
			return nil, err
		}
		current_rows = append(current_rows, rolname)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return current_rows, nil
}

func createRole(db *sql.DB, roleName string, attributes []genv1alpha1.PostgreSqlUserAttributes) error {
	var query strings.Builder
	query.WriteString("CREATE ROLE $1")
	if len(attributes) > 0 {
		query.WriteString(" WITH ")
		for i, attr := range attributes {
			if i > 0 {
				query.WriteString(", ")
			}
			query.WriteString(string(attr))
		}
	}
	_, err := db.Exec(query.String(), roleName)
	return err
}

func createOrReplaceUser(db *sql.DB, spec *genv1alpha1.PostgreSqlSpec) (map[string][]byte, error) {
	username := spec.User.Username
	if spec.User.RandomSufix {
		ioReader := rand.Reader
		randomNumber, err := rand.Int(ioReader, new(big.Int).SetInt64(10000))
		if err != nil {
			return nil, err
		}
		username += fmt.Sprintf("%04d", randomNumber)
	}

	userAttributes, err := ConvertStringArrayToAttributes(spec.User.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert user attributes: %w", err)
	}
	err = createRole(db, username, userAttributes)
	if err != nil {
		return nil, fmt.Errorf("failed to create role %s: %w", username, err)
	}

	userAttributesQuery := `CREATE ROLE $1 WITH LOGIN PASSWORD '$2'`

	pass, err := generatePassword(genv1alpha1.Password{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	_, err = db.Exec(userAttributesQuery, username, string(pass))
	if err != nil {
		return nil, fmt.Errorf("failed to create user %s: %w", username, err)
	}

	err = addRolesToUser(db, username, spec.User.Roles)
	if err != nil {
		return nil, fmt.Errorf("failed to add roles to user %s: %w", username, err)
	}

	return map[string][]byte{
		"user":     []byte(username),
		"password": pass,
	}, nil
}

func addRolesToUser(db *sql.DB, username string, roles []string) error {
	current_roles, err := getExistingRoles(db)
	if err != nil {
		return fmt.Errorf("failed to get existing roles: %w", err)
	}

	for _, role := range roles {
		if !slices.Contains(current_roles, role) {
			err = createRole(db, role, nil)
			if err != nil {
				return fmt.Errorf("failed to create role %s: %w", role, err)
			}
		}

		grantRoleQuery := `GRANT $1 TO $2`
		_, err = db.Exec(grantRoleQuery, role, username)
		if err != nil {
			return fmt.Errorf("failed to grant role %s to user %s: %w", role, username, err)
		}
	}
	return nil
}

func dropUser(db *sql.DB, username, adminUser string, destructive bool) error {
	if !destructive {
		_, err := db.Exec(`REASSIGN OWNED BY $1 TO $2`, username, adminUser)
		if err != nil {
			return err
		}
	}
	dropQueries := []string{
		`DROP OWNED BY $1`,
		`DROP ROLE $1`,
	}
	for _, query := range dropQueries {
		_, err := db.Exec(query, username)
		if err != nil {
			return err
		}
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

var validAttributes = map[string]genv1alpha1.PostgreSqlUserAttributes{
	string(genv1alpha1.PostgreSqlUserSuperUser):   genv1alpha1.PostgreSqlUserSuperUser,
	string(genv1alpha1.PostgreSqlUserCreateDb):    genv1alpha1.PostgreSqlUserCreateDb,
	string(genv1alpha1.PostgreSqlUserCreateRole):  genv1alpha1.PostgreSqlUserCreateRole,
	string(genv1alpha1.PostgreSqlUserReplication): genv1alpha1.PostgreSqlUserReplication,
}

// ConvertStringArrayToAttributes converts []string to []PostgreSqlUserAttributes
func ConvertStringArrayToAttributes(input []string) ([]genv1alpha1.PostgreSqlUserAttributes, error) {
	var attrs []genv1alpha1.PostgreSqlUserAttributes
	for _, val := range input {
		attr, ok := validAttributes[strings.ToUpper(val)]
		if !ok {
			return nil, fmt.Errorf("invalid attribute: %s", val)
		}
		attrs = append(attrs, attr)
	}
	return attrs, nil
}

func parseSpec(data []byte) (*genv1alpha1.PostgreSql, error) {
	var spec genv1alpha1.PostgreSql
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func parseStatus(data []byte) (*genv1alpha1.PostgreSqlUserState, error) {
	var state genv1alpha1.PostgreSqlUserState
	err := json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.PostgreSqlKind, &Generator{})
}

// sqlStatement := `
// INSERT INTO users (age, email, first_name, last_name)
// VALUES ($1, $2, $3, $4)
// RETURNING id`
// id := 0
// err := db.QueryRow(sqlStatement, 30, "jon@calhoun.io", "Jonathan", "Calhoun").Scan(&id)
// if err != nil {
// 	panic(err)
// }
// fmt.Println("New record ID is:", id)

// sqlStatement = `
// DELETE FROM users
// WHERE id = $1;`
// res, err := db.Exec(sqlStatement, 1)
// if err != nil {
// 	panic(err)
// }
// count, err := res.RowsAffected()
// if err != nil {
// 	panic(err)
// }
// fmt.Println(count)

// sqlStatement = `SELECT id, email FROM users WHERE id=$1;`
// var email string
// // Replace 3 with an ID from your database or another random
// // value to test the no rows use case.
// row := db.QueryRow(sqlStatement, 3)
// switch err := row.Scan(&id, &email); err {
// case sql.ErrNoRows:
// 	fmt.Println("No rows were returned!")
// case nil:
// 	fmt.Println(id, email)
// default:
// 	panic(err)
// }

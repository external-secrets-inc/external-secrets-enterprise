// Copyright External Secrets Inc. All Rights Reserved

package postgresql

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	enterprise "github.com/external-secrets/external-secrets/apis/enterprise/generators/v1alpha1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/generator/password"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Generator struct{}

const (
	defaultPort       = "5432"
	defaultUser       = "postgres"
	defaultDbName     = "postgres"
	defaultSuffixSize = 8
)

var mapAttributes = map[string]enterprise.PostgreSqlUserAttributesEnum{
	string(enterprise.PostgreSqlUserSuperUser):   enterprise.PostgreSqlUserSuperUser,
	string(enterprise.PostgreSqlUserCreateDb):    enterprise.PostgreSqlUserCreateDb,
	string(enterprise.PostgreSqlUserCreateRole):  enterprise.PostgreSqlUserCreateRole,
	string(enterprise.PostgreSqlUserReplication): enterprise.PostgreSqlUserReplication,
	string(enterprise.PostgreSqlUserNoInherit):   enterprise.PostgreSqlUserNoInherit,
	string(enterprise.PostgreSqlUserByPassRls):   enterprise.PostgreSqlUserByPassRls,
	"CONNECTION_LIMIT":                           enterprise.PostgreSqlUserConnectionLimit,
	string(enterprise.PostgreSqlUserLogin):       enterprise.PostgreSqlUserLogin,
	string(enterprise.PostgreSqlUserPassword):    enterprise.PostgreSqlUserPassword,
}

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
		err := db.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close db: %v", err)
		}
	}()

	err = db.Ping(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to ping the database: %w", err)
	}

	user, err := createUser(ctx, db, &res.Spec)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create or update user: %w", err)
	}

	username, ok := user["username"]
	if !ok {
		return nil, nil, fmt.Errorf("user not found in response")
	}

	rawState, err := json.Marshal(&enterprise.PostgreSqlUserState{
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
		err := db.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close db: %v", err)
		}
	}()

	err = db.Ping(ctx)
	if err != nil {
		return fmt.Errorf("unable to ping the database: %w", err)
	}

	err = dropUser(ctx, db, status.Username, res.Spec)
	if err != nil {
		return fmt.Errorf("unable to drop user: %w", err)
	}

	return nil
}

func (g *Generator) GetCleanupPolicy(obj *apiextensions.JSON) (*genv1alpha1.CleanupPolicy, error) {
	return nil, nil
}

func (g *Generator) LastActivityTime(ctx context.Context, obj *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, kube client.Client, namespace string) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

func (g *Generator) GetKeys() map[string]string {
	return map[string]string{
		"username": "PostgreSQL database username",
		"password": "PostgreSQL user password",
	}
}

func newConnection(ctx context.Context, spec *enterprise.PostgreSqlSpec, kclient client.Client, ns string) (*pgx.Conn, error) {
	dbName := defaultDbName
	if spec.Database != "" {
		dbName = spec.Database
	}

	port := defaultPort
	if spec.Port != "" {
		port = spec.Port
	}

	username := defaultUser
	if spec.Auth.Username != "" {
		username = spec.Auth.Username
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
		spec.Host, port, username, password, dbName,
	)

	return pgx.Connect(ctx, psqlInfo)
}

func getExistingRoles(ctx context.Context, db *pgx.Conn) ([]string, error) {
	var current_rows = make([]string, 0)
	rows, err := db.Query(ctx, "SELECT rolname FROM pg_roles")
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

func addRolesAttributesToQueryString(query *strings.Builder, attributes []enterprise.PostgreSqlUserAttribute) {
	if len(attributes) > 0 {
		query.WriteString(" WITH ")
		for i, attr := range attributes {
			if i > 0 {
				query.WriteString(" ")
			}
			if attr.Value != nil {
				if string(mapAttributes[attr.Name]) == string(enterprise.PostgreSqlUserPassword) {
					fmt.Fprintf(query, `%s '%s'`, string(mapAttributes[attr.Name]), *attr.Value)
				} else {
					fmt.Fprintf(query, `%s %s`, string(mapAttributes[attr.Name]), *attr.Value)
				}
			} else {
				query.WriteString(string(mapAttributes[attr.Name]))
			}
		}
	}
}

func createRole(ctx context.Context, db *pgx.Conn, roleName string, attributes []enterprise.PostgreSqlUserAttribute) error {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("CREATE ROLE %s", pgx.Identifier{roleName}.Sanitize()))
	addRolesAttributesToQueryString(&query, attributes)
	_, err := db.Exec(ctx, query.String())
	return err
}

func updateRole(ctx context.Context, db *pgx.Conn, roleName string, attributes []enterprise.PostgreSqlUserAttribute) error {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("ALTER ROLE %s", pgx.Identifier{roleName}.Sanitize()))
	addRolesAttributesToQueryString(&query, attributes)
	_, err := db.Exec(ctx, query.String())
	return err
}

func resetRole(ctx context.Context, db *pgx.Conn, roleName string) error {
	sanitizedRole := pgx.Identifier{roleName}.Sanitize()

	_, err := db.Exec(ctx, fmt.Sprintf(`
		ALTER ROLE %s WITH NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOLOGIN NOREPLICATION NOBYPASSRLS
	`, sanitizedRole))
	if err != nil {
		return fmt.Errorf("failed to reset attributes for role %s: %w", roleName, err)
	}

	rows, err := db.Query(ctx, `
		SELECT r.rolname
		FROM pg_auth_members m
		JOIN pg_roles r ON r.oid = m.roleid
		JOIN pg_roles u ON u.oid = m.member
		WHERE u.rolname = $1
	`, roleName)
	if err != nil {
		return fmt.Errorf("failed to list granted roles for %s: %w", roleName, err)
	}
	defer rows.Close()

	var grantedRoles []string
	for rows.Next() {
		var grantedRole string
		if err := rows.Scan(&grantedRole); err != nil {
			return fmt.Errorf("failed to scan granted role: %w", err)
		}
		grantedRoles = append(grantedRoles, pgx.Identifier{grantedRole}.Sanitize())
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating granted roles: %w", err)
	}

	rolesCSV := strings.Join(grantedRoles, ", ")

	_, err = db.Exec(ctx, fmt.Sprintf("REVOKE %s FROM %s", rolesCSV, sanitizedRole))
	if err != nil {
		return fmt.Errorf("failed to revoke roles [%s] from %s: %w", rolesCSV, roleName, err)
	}

	return nil
}

func createUser(ctx context.Context, db *pgx.Conn, spec *enterprise.PostgreSqlSpec) (map[string][]byte, error) {
	username := spec.User.Username
	suffixSize := defaultSuffixSize
	if spec.User.SuffixSize != nil {
		suffixSize = *spec.User.SuffixSize
	}
	suffix, err := utils.GenerateRandomString(suffixSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random suffix: %w", err)
	}

	if suffix != "" {
		username = fmt.Sprintf("%s_%s", username, suffix)
	}

	current_roles, err := getExistingRoles(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing roles: %w", err)
	}

	pass, err := generatePassword(genv1alpha1.Password{
		Spec: genv1alpha1.PasswordSpec{
			SymbolCharacters: ptr.To("~!@#$%^&*()_+-={}|[]:<>?,./"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	spec.User.Attributes = append(spec.User.Attributes,
		enterprise.PostgreSqlUserAttribute{
			Name: string(enterprise.PostgreSqlUserLogin),
		}, enterprise.PostgreSqlUserAttribute{
			Name:  string(enterprise.PostgreSqlUserPassword),
			Value: ptr.To(string(pass)),
		},
	)

	if !slices.Contains(current_roles, username) {
		err = createRole(ctx, db, username, spec.User.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to create role %s: %w", username, err)
		}
	} else {
		err = resetRole(ctx, db, username)
		if err != nil {
			return nil, fmt.Errorf("failed to reset role %s: %w", username, err)
		}
		err = updateRole(ctx, db, username, spec.User.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to create role %s: %w", username, err)
		}
	}

	err = grantRolesToUser(ctx, db, username, spec.User.Roles, current_roles)
	if err != nil {
		return nil, fmt.Errorf("failed to add roles to user %s: %w", username, err)
	}

	return map[string][]byte{
		"username": []byte(username),
		"password": pass,
	}, nil
}

func grantRolesToUser(ctx context.Context, db *pgx.Conn, username string, roles, current_roles []string) error {
	sanitizedUsername := pgx.Identifier{username}.Sanitize()

	toGrant := make([]string, 0, len(roles))
	for _, role := range roles {
		if !slices.Contains(current_roles, role) {
			if err := createRole(ctx, db, role, nil); err != nil {
				return fmt.Errorf("failed to create role %s: %w", role, err)
			}
		}
		toGrant = append(toGrant, pgx.Identifier{role}.Sanitize())
	}

	if len(toGrant) == 0 {
		return nil
	}

	rolesCSV := strings.Join(toGrant, ", ")
	query := fmt.Sprintf("GRANT %s TO %s", rolesCSV, sanitizedUsername)

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to grant roles [%s] to user %s: %w", rolesCSV, username, err)
	}

	return nil
}

func dropUser(ctx context.Context, db *pgx.Conn, username string, spec enterprise.PostgreSqlSpec) error {
	sanitizedUsername := pgx.Identifier{username}.Sanitize()
	if !spec.User.DestructiveCleanup {
		reassignToUser := spec.Auth.Username
		if spec.User.ReassignTo != nil && *spec.User.ReassignTo != "" {
			reassignToUser = *spec.User.ReassignTo
		}

		current_roles, err := getExistingRoles(ctx, db)
		if err != nil {
			return fmt.Errorf("failed to get existing roles: %w", err)
		}
		if !slices.Contains(current_roles, reassignToUser) {
			err = createRole(ctx, db, reassignToUser, nil)
			if err != nil {
				return fmt.Errorf("failed to create role %s: %w", reassignToUser, err)
			}
		}

		_, err = db.Exec(ctx, fmt.Sprintf(`REASSIGN OWNED BY %s TO %s`, sanitizedUsername, pgx.Identifier{reassignToUser}.Sanitize()))
		if err != nil {
			return fmt.Errorf("failed to reassign owned by %s to %s: %w", username, reassignToUser, err)
		}
	}
	dropQueries := []string{
		`DROP OWNED BY %s`,
		`DROP ROLE %s`,
	}
	for _, query := range dropQueries {
		_, err := db.Exec(ctx, fmt.Sprintf(query, sanitizedUsername))
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

func parseSpec(data []byte) (*enterprise.PostgreSql, error) {
	var spec enterprise.PostgreSql
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func parseStatus(data []byte) (*enterprise.PostgreSqlUserState, error) {
	var state enterprise.PostgreSqlUserState
	err := json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, err
}

func init() {
	genv1alpha1.Register(enterprise.PostgreSqlKind, &Generator{})
	genv1alpha1.RegisterGeneric(enterprise.PostgreSqlKind, &enterprise.PostgreSql{})
}
